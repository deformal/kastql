package planner

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	gqlparser "github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/validator/rules"
	"go.uber.org/zap"

	"github.com/deformal/kastql/internal/metadata"
	"github.com/deformal/kastql/internal/registry"
)

// PermissionChecker is implemented by auth.Checker to enforce field-level access.
// Defined here to avoid a circular import between auth ↔ planner.
type PermissionChecker interface {
	CanAccess(ctx context.Context, role, typeName, fieldName string) (bool, error)
}

// Planner plans GraphQL queries against a merged schema.
type Planner struct {
	mu      sync.RWMutex
	merged  *MergedSchema
	store   *metadata.Store
	log     *zap.Logger
	stepSeq atomic.Uint64
	checker PermissionChecker // optional; nil = allow everything
}

// New creates a Planner. Call Update whenever the service registry changes.
func New(store *metadata.Store, log *zap.Logger) *Planner {
	return &Planner{store: store, log: log}
}

// SetChecker attaches a permission checker. Call once at startup.
func (p *Planner) SetChecker(c PermissionChecker) {
	p.checker = c
}

// Update rebuilds the merged schema from the current registry state.
// Call this after any service add/remove/reload.
func (p *Planner) Update(entries []*registry.ServiceEntry) error {
	if len(entries) == 0 {
		p.mu.Lock()
		p.merged = nil
		p.mu.Unlock()
		return nil
	}
	merged, err := Merge(entries)
	if err != nil {
		return fmt.Errorf("merge schemas: %w", err)
	}
	p.mu.Lock()
	p.merged = merged
	p.mu.Unlock()
	p.log.Info("planner schema updated",
		zap.Int("services", len(entries)),
		zap.Int("query_fields", len(merged.QueryOwnership)),
	)
	return nil
}

// ResolveSubscriptionURL returns the upstream service URL that owns the first
// subscription root field in the query. Returns "" if it cannot be determined.
func (p *Planner) ResolveSubscriptionURL(query string) string {
	p.mu.RLock()
	merged := p.merged
	p.mu.RUnlock()
	if merged == nil || merged.Schema == nil {
		return ""
	}

	doc, gqlErr := gqlparser.LoadQueryWithRules(merged.Schema, query, rules.NewDefaultRules())
	if gqlErr != nil {
		return ""
	}
	for _, op := range doc.Operations {
		if op.Operation != ast.Subscription {
			continue
		}
		for _, sel := range op.SelectionSet {
			if field, ok := sel.(*ast.Field); ok {
				svcName := merged.SubscriptionOwnership[field.Name]
				if svcName == "" {
					return ""
				}
				url := merged.ServiceURLs[svcName]
				// Convert http(s) URL to ws(s)
				if len(url) > 7 && url[:7] == "http://" {
					return "ws://" + url[7:]
				}
				if len(url) > 8 && url[:8] == "https://" {
					return "wss://" + url[8:]
				}
				return url
			}
		}
	}
	return ""
}

// IntrospectionTarget returns the URL and service-level headers of the first
// available service. Used by the router to forward introspection queries directly
// (meta-fields like __schema/__type are not in the planner's ownership maps).
func (p *Planner) IntrospectionTarget() (url string, headers map[string]string) {
	p.mu.RLock()
	merged := p.merged
	p.mu.RUnlock()
	if merged == nil {
		return "", nil
	}
	for name, u := range merged.ServiceURLs {
		return u, merged.ServiceHeaders[name]
	}
	return "", nil
}

// MergedSDL returns the current merged schema SDL (empty if no services loaded).
func (p *Planner) MergedSDL() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.merged == nil {
		return ""
	}
	return p.merged.SDL
}

// Plan parses the query and produces a QueryPlan. Returns an error if the
// query is invalid, no schema is loaded, or a permission check fails.
func (p *Planner) Plan(_ context.Context, query string, variables map[string]any, role string) (*QueryPlan, error) {
	p.mu.RLock()
	merged := p.merged
	p.mu.RUnlock()

	if merged == nil || merged.Schema == nil {
		return nil, errors.New("no schema loaded — register at least one service")
	}

	doc, gqlErr := gqlparser.LoadQueryWithRules(merged.Schema, query, rules.NewDefaultRules())
	if gqlErr != nil {
		return nil, gqlErr
	}

	ps := &planSession{
		p:         p,
		merged:    merged,
		variables: variables,
		fragments: doc.Fragments,
		store:     p.store,
		role:      role,
		checker:   p.checker,
	}

	var allSteps []*Step
	plan := &QueryPlan{}
	for i, op := range doc.Operations {
		if i == 0 {
			plan.OperationType = strings.ToLower(string(op.Operation))
			plan.OperationName = op.Name
		}
		steps, err := ps.planOperation(op)
		if err != nil {
			return nil, err
		}
		allSteps = append(allSteps, steps...)
	}

	plan.Steps = allSteps
	return plan, nil
}

// planSession holds per-request state for planning.
type planSession struct {
	p         *Planner
	merged    *MergedSchema
	variables map[string]any
	fragments ast.FragmentDefinitionList
	store     *metadata.Store
	role      string
	checker   PermissionChecker
}

func (ps *planSession) nextID() string {
	return fmt.Sprintf("step_%d", ps.p.stepSeq.Add(1))
}

// planOperation builds steps for one operation (query/mutation/subscription).
func (ps *planSession) planOperation(op *ast.OperationDefinition) ([]*Step, error) {
	opType := strings.ToLower(string(op.Operation))

	var ownershipMap map[string]string
	switch op.Operation {
	case ast.Query:
		ownershipMap = ps.merged.QueryOwnership
	case ast.Mutation:
		ownershipMap = ps.merged.MutationOwnership
	case ast.Subscription:
		ownershipMap = ps.merged.SubscriptionOwnership
	default:
		return nil, fmt.Errorf("unknown operation type: %s", op.Operation)
	}

	// Group root selections by owning service
	byService := map[string]ast.SelectionSet{}
	for _, sel := range op.SelectionSet {
		field, ok := sel.(*ast.Field)
		if !ok {
			continue // inline fragments at root — skip for now
		}
		svc := ownershipMap[field.Name]
		if svc == "" {
			return nil, fmt.Errorf("field %q not found in any registered service", field.Name)
		}
		byService[svc] = append(byService[svc], sel)
	}

	var steps []*Step
	for svcName, selections := range byService {
		rootSteps, err := ps.buildRootStep(opType, op.Name, op.VariableDefinitions, svcName, selections)
		if err != nil {
			return nil, err
		}
		steps = append(steps, rootSteps...)
	}
	return steps, nil
}

// buildRootStep creates a Step for a service's slice of root selections and
// any dependent steps needed for cross-service fields within.
func (ps *planSession) buildRootStep(
	opType, opName string,
	varDefs ast.VariableDefinitionList,
	serviceName string,
	selections ast.SelectionSet,
) ([]*Step, error) {
	rootID := ps.nextID()

	// Walk selections to detect cross-service nested fields and build the
	// service-local selection (possibly injecting @key fields for federation).
	localSel, dependents, err := ps.walkSelections(selections, rootID, serviceName, "Query", nil)
	if err != nil {
		return nil, err
	}

	// Build the sub-query string
	var qb strings.Builder
	qb.WriteString(opType)
	if opName != "" {
		fmt.Fprintf(&qb, " %s", opName)
	}
	vars := varDefsToSDL(varDefs)
	if vars != "" {
		qb.WriteString(vars)
	}
	qb.WriteString(" {\n")
	qb.WriteString(selectionToQueryString(localSel, nil, "  "))
	qb.WriteString("}")

	root := &Step{
		ID:          rootID,
		ServiceName: serviceName,
		ServiceURL:  ps.merged.ServiceURLs[serviceName],
		ServiceType: ps.merged.ServiceTypes[serviceName],
		Query:       qb.String(),
		Variables:   ps.variables,
		MergePath:   nil,
		Meta:        StepMeta{Kind: StepKindRoot},
	}

	all := append([]*Step{root}, dependents...)
	return all, nil
}

// walkSelections iterates a SelectionSet for a given service and:
//   - Keeps fields that belong to this service (including scalar fields of
//     cross-service types that this service resolves as part of its schema)
//   - Injects @key fields for federation entity types that point elsewhere
//   - Creates dependent EntityStep / JoinStep for cross-service selections
//
// It returns the local SelectionSet (to send to this service) plus dependent steps.
func (ps *planSession) walkSelections(
	selections ast.SelectionSet,
	parentStepID string,
	currentService string,
	currentType string,
	parentPath []string,
) (ast.SelectionSet, []*Step, error) {
	var localSel ast.SelectionSet
	var dependents []*Step

	for _, sel := range selections {
		field, ok := sel.(*ast.Field)
		if !ok {
			// Inline fragments / fragment spreads: include as-is
			localSel = append(localSel, sel)
			continue
		}

		if field.Definition == nil || field.Definition.Type == nil {
			// Cannot determine type — include field unchanged
			localSel = append(localSel, sel)
			continue
		}

		// Permission check before we go any further.
		if ps.checker != nil {
			allowed, err := ps.checker.CanAccess(context.Background(), ps.role, currentType, field.Name)
			if err != nil {
				return nil, nil, fmt.Errorf("permission check error: %w", err)
			}
			if !allowed {
				return nil, nil, fmt.Errorf("permission denied: role %q cannot access %s.%s", ps.role, currentType, field.Name)
			}
		}

		returnType := namedTypeName(field.Definition.Type)
		if returnType == "" || isScalarOrEnum(returnType, ps.merged.Schema) {
			// Scalar / enum — always stays with the current service
			localSel = append(localSel, sel)
			continue
		}

		// Determine who owns the return type
		typeOwner := ps.merged.TypeOwnership[returnType]

		if typeOwner == "" || typeOwner == currentService {
			// Same service (or unknown) — recurse into nested selection
			if len(field.SelectionSet) > 0 {
				nestedPath := append(append([]string{}, parentPath...), field.Name)
				nestedSel, deps, err := ps.walkSelections(field.SelectionSet, parentStepID, currentService, returnType, nestedPath)
				if err != nil {
					return nil, nil, err
				}
				// Replace the field's selection set with the local-only subset
				localField := cloneFieldWithSel(field, nestedSel)
				localSel = append(localSel, localField)
				dependents = append(dependents, deps...)
			} else {
				localSel = append(localSel, sel)
			}
			continue
		}

		// Cross-service field: returnType is owned by a different service.
		fieldPath := append(append([]string{}, parentPath...), field.Name)

		// Determine resolution strategy
		if ps.merged.ServiceTypes[currentService] == "federation" ||
			ps.merged.ServiceTypes[typeOwner] == "federation" {
			// Federation entity resolution
			keyFields := ps.entityKeyFields(returnType, typeOwner)
			if len(keyFields) == 0 {
				// No @key known — include as-is and let the upstream handle it
				localSel = append(localSel, sel)
				continue
			}

			// Inject @key fields into the local selection for this field
			localField := injectKeyFields(field, keyFields)
			localSel = append(localSel, localField)

			// Build the _entities sub-query selection
			entitySel := onlyNonKeyFields(field.SelectionSet, keyFields)
			if len(entitySel) == 0 {
				continue // nothing to fetch from entity service
			}
			entitySelStr := "{\n" + selectionToQueryString(entitySel, nil, "    ") + "  }"

			depID := ps.nextID()
			dep := &Step{
				ID:          depID,
				ServiceName: typeOwner,
				ServiceURL:  ps.merged.ServiceURLs[typeOwner],
				ServiceType: ps.merged.ServiceTypes[typeOwner],
				Query:       buildEntitiesQuery(returnType, keyFields, entitySelStr),
				Variables:   ps.variables,
				DependsOn:   []string{parentStepID},
				MergePath:   fieldPath,
				Meta: StepMeta{
					Kind: StepKindEntity,
					Entity: &EntityMeta{
						TypeName:     returnType,
						KeyFields:    keyFields,
						ParentStepID: parentStepID,
						ParentPath:   fieldPath,
						Selection:    entitySelStr,
					},
				},
			}
			dependents = append(dependents, dep)

		} else {
			// Stitching join — look up relationship from metadata
			rel, err := ps.findRelationship(currentService, currentType, field.Name)
			if err != nil || rel == nil {
				// No relationship configured — include the field as-is
				localSel = append(localSel, sel)
				continue
			}

			// Ensure the join key field is in the local selection
			localField := injectFieldByName(field, rel.SourceField)
			localSel = append(localSel, localField)

			depID := ps.nextID()
			dep := &Step{
				ID:          depID,
				ServiceName: typeOwner,
				ServiceURL:  ps.merged.ServiceURLs[typeOwner],
				ServiceType: ps.merged.ServiceTypes[typeOwner],
				Query:       buildJoinQuery(rel.TargetType, rel.SourceField, field.SelectionSet),
				Variables:   ps.variables,
				DependsOn:   []string{parentStepID},
				MergePath:   fieldPath,
				Meta: StepMeta{
					Kind: StepKindJoin,
					Join: &JoinMeta{
						RelationshipName: rel.Name,
						ParentStepID:     parentStepID,
						ParentKeyField:   rel.SourceField,
						TargetField:      rel.TargetType,
						TargetArgName:    rel.SourceField,
					},
				},
			}
			dependents = append(dependents, dep)
		}
	}

	return localSel, dependents, nil
}

// entityKeyFields returns the @key fields for a type in a given service.
// Falls back to any service's keys if the target service isn't found.
func (ps *planSession) entityKeyFields(typeName, ownerService string) []string {
	if keys, ok := ps.merged.EntityKeys[typeName]; ok {
		if fields, ok := keys[ownerService]; ok {
			return fields
		}
		// Fallback: use keys from any service
		for _, fields := range keys {
			return fields
		}
	}
	return nil
}

// findRelationship looks up a stitching relationship from the metadata store.
func (ps *planSession) findRelationship(sourceService, sourceType, fieldName string) (*metadata.Relationship, error) {
	rows, err := ps.store.DB().QueryContext(
		context.Background(),
		`SELECT id, name, source_service, source_type, source_field, target_service, target_type, join_config, created_at
		 FROM relationships
		 WHERE source_service = ? AND source_type = ? AND source_field = ?
		 LIMIT 1`,
		sourceService, sourceType, fieldName,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	var rel metadata.Relationship
	var createdAt string
	if err := rows.Scan(&rel.ID, &rel.Name, &rel.SourceService, &rel.SourceType,
		&rel.SourceField, &rel.TargetService, &rel.TargetType, &rel.JoinConfig, &createdAt); err != nil {
		return nil, err
	}
	return &rel, nil
}

// ---- query building helpers ----

// buildEntitiesQuery produces:
//
//	query($representations:[_Any!]!) {
//	  _entities(representations:$representations) {
//	    ... on TypeName { <selection> }
//	  }
//	}
func buildEntitiesQuery(typeName string, _ []string, selectionStr string) string {
	return fmt.Sprintf(
		"query($representations:[_Any!]!) {\n  _entities(representations:$representations) {\n    ... on %s %s\n  }\n}",
		typeName, selectionStr,
	)
}

// buildJoinQuery builds a simple root query for a stitching join step.
// The executor will call this repeatedly with different argument values.
func buildJoinQuery(fieldName, argName string, sel ast.SelectionSet) string {
	var b strings.Builder
	fmt.Fprintf(&b, "query($arg: ID!) {\n  %s(%s: $arg) {\n", fieldName, argName)
	b.WriteString(selectionToQueryString(sel, nil, "    "))
	b.WriteString("  }\n}")
	return b.String()
}

// cloneFieldWithSel returns a shallow copy of field with a replaced SelectionSet.
func cloneFieldWithSel(f *ast.Field, sel ast.SelectionSet) *ast.Field {
	clone := *f
	clone.SelectionSet = sel
	return &clone
}

// injectKeyFields ensures @key fields are present in a field's SelectionSet.
func injectKeyFields(f *ast.Field, keyFields []string) *ast.Field {
	sel := f.SelectionSet
	for _, kf := range keyFields {
		found := false
		for _, s := range sel {
			if sf, ok := s.(*ast.Field); ok && sf.Name == kf {
				found = true
				break
			}
		}
		if !found {
			sel = append(sel, &ast.Field{Name: kf, Alias: kf})
		}
	}
	return cloneFieldWithSel(f, sel)
}

// injectFieldByName ensures a specific field name is present in the SelectionSet.
func injectFieldByName(f *ast.Field, fieldName string) *ast.Field {
	for _, s := range f.SelectionSet {
		if sf, ok := s.(*ast.Field); ok && sf.Name == fieldName {
			return f // already present
		}
	}
	sel := append(f.SelectionSet, &ast.Field{Name: fieldName, Alias: fieldName})
	return cloneFieldWithSel(f, sel)
}

// onlyNonKeyFields returns the sub-selections that are NOT key fields.
// These are the fields we need to fetch from the entity's owning service.
func onlyNonKeyFields(sel ast.SelectionSet, keyFields []string) ast.SelectionSet {
	keySet := map[string]bool{}
	for _, k := range keyFields {
		keySet[k] = true
	}
	var out ast.SelectionSet
	for _, s := range sel {
		if f, ok := s.(*ast.Field); ok && keySet[f.Name] {
			continue
		}
		out = append(out, s)
	}
	return out
}

// isScalarOrEnum returns true if typeName is a scalar or enum in the schema.
func isScalarOrEnum(typeName string, schema *ast.Schema) bool {
	if schema == nil {
		return false
	}
	t := schema.Types[typeName]
	if t == nil {
		return false
	}
	return t.Kind == ast.Scalar || t.Kind == ast.Enum
}
