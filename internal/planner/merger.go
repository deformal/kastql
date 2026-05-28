package planner

import (
	"encoding/json"
	"fmt"
	"strings"

	gqlparser "github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"

	"github.com/deformal/kastql/internal/metadata"
	"github.com/deformal/kastql/internal/registry"
)

// federationPreamble defines federation v2 directives and scalars so that
// gqlparser can parse federation-annotated SDL without errors.
const federationPreamble = `
scalar _Any
scalar _FieldSet
scalar FieldSet
scalar link__Import

directive @key(fields: _FieldSet!, resolvable: Boolean) repeatable on OBJECT | INTERFACE
directive @external on OBJECT | FIELD_DEFINITION
directive @requires(fields: _FieldSet!) on FIELD_DEFINITION
directive @provides(fields: _FieldSet!) on FIELD_DEFINITION
directive @shareable on OBJECT | FIELD_DEFINITION
directive @inaccessible on FIELD_DEFINITION | OBJECT | INTERFACE | UNION | ARGUMENT_DEFINITION | SCALAR | ENUM | ENUM_VALUE | INPUT_OBJECT | INPUT_FIELD_DEFINITION
directive @override(from: String!) on FIELD_DEFINITION
directive @link(url: String!, import: [link__Import]) repeatable on SCHEMA
directive @extends on OBJECT | INTERFACE
directive @tag(name: String!) repeatable on FIELD_DEFINITION | OBJECT | INTERFACE | UNION | ARGUMENT_DEFINITION | SCALAR | ENUM | ENUM_VALUE | INPUT_OBJECT | INPUT_FIELD_DEFINITION
`

// Merge combines SDL from all registered services into a single MergedSchema.
// Federation services have @key / @external etc; stitching services have plain SDL.
func Merge(entries []*registry.ServiceEntry) (*MergedSchema, error) {
	if len(entries) == 0 {
		return nil, fmt.Errorf("no services registered")
	}

	result := &MergedSchema{
		QueryOwnership:        make(map[string]string),
		MutationOwnership:     make(map[string]string),
		SubscriptionOwnership: make(map[string]string),
		TypeOwnership:         make(map[string]string),
		EntityKeys:            make(map[string]map[string][]string),
		ServiceURLs:           make(map[string]string),
		ServiceTypes:          make(map[string]string),
		ServiceHeaders:        make(map[string]map[string]string),
	}

	// accumulated type definitions (name → definition)
	queryFields := map[string]*ast.FieldDefinition{}
	mutationFields := map[string]*ast.FieldDefinition{}
	subscriptionFields := map[string]*ast.FieldDefinition{}
	otherTypes := map[string]*ast.Definition{}

	for _, entry := range entries {
		result.ServiceURLs[entry.Name] = entry.URL
		result.ServiceTypes[entry.Name] = string(entry.Type)
		if h, err := jsonToHeaderMap(entry.Headers); err == nil && len(h) > 0 {
			result.ServiceHeaders[entry.Name] = h
		}

		doc, err := parseServiceSDL(entry)
		if err != nil {
			return nil, fmt.Errorf("parse SDL for %s: %w", entry.Name, err)
		}

		// Process top-level type definitions
		for _, def := range doc.Definitions {
			if def.BuiltIn {
				continue
			}
			processDefinition(def, entry.Name, false, result,
				queryFields, mutationFields, subscriptionFields, otherTypes)
		}

		// Process extension types (extend type X { ... })
		for _, ext := range doc.Extensions {
			if ext.BuiltIn {
				continue
			}
			processDefinition(ext, entry.Name, true, result,
				queryFields, mutationFields, subscriptionFields, otherTypes)
		}
	}

	// Build the merged SDL string
	result.SDL = buildMergedSDL(queryFields, mutationFields, subscriptionFields, otherTypes)

	// Parse + resolve the merged SDL into *ast.Schema for query planning
	schema, gqlErr := gqlparser.LoadSchema(&ast.Source{Name: "merged", Input: result.SDL})
	if gqlErr != nil {
		return nil, fmt.Errorf("load merged schema: %w", gqlErr)
	}
	result.Schema = schema

	return result, nil
}

// parseServiceSDL parses a service's SDL into a *ast.SchemaDocument.
// For federation services it prepends the federation directive definitions.
func parseServiceSDL(entry *registry.ServiceEntry) (*ast.SchemaDocument, error) {
	sdl := sanitizeFederationSDL(entry.SDL)

	var sources []*ast.Source
	if entry.Type == metadata.ServiceTypeFederation {
		sources = append(sources, &ast.Source{Name: "federation_preamble", Input: federationPreamble})
	}
	sources = append(sources, &ast.Source{Name: entry.Name + ".graphql", Input: sdl})

	doc, gqlErr := parser.ParseSchemas(sources...)
	if gqlErr != nil {
		return nil, gqlErr
	}
	return doc, nil
}

// sanitizeFederationSDL removes scalar/type declarations from the service SDL
// that would conflict with the federation preamble we prepend.
func sanitizeFederationSDL(sdl string) string {
	conflicts := []string{
		"scalar _Any", "scalar _FieldSet", "scalar FieldSet",
		"scalar _Service", "scalar link__Import", "scalar link__Purpose",
		"type _Service", "union _Entity",
	}
	lines := strings.Split(sdl, "\n")
	out := make([]string, 0, len(lines))
	skipBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect multi-line block starts to skip (e.g. type _Service { ... })
		if skipBlock {
			if trimmed == "}" {
				skipBlock = false
			}
			continue
		}

		skip := false
		for _, c := range conflicts {
			if strings.HasPrefix(trimmed, c) {
				skip = true
				if strings.Contains(trimmed, "{") && !strings.Contains(trimmed, "}") {
					skipBlock = true
				}
				break
			}
		}
		// Also drop schema @link annotations
		if strings.HasPrefix(trimmed, "extend schema") || strings.HasPrefix(trimmed, "schema {") {
			skip = true
		}

		if !skip {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

// processDefinition extracts type information from a parsed definition and
// updates the ownership maps and type collections.
func processDefinition(
	def *ast.Definition,
	serviceName string,
	isExtension bool,
	result *MergedSchema,
	queryFields, mutationFields, subscriptionFields map[string]*ast.FieldDefinition,
	otherTypes map[string]*ast.Definition,
) {
	// Skip federation internal types
	if federationScalarNames[def.Name] || strings.HasPrefix(def.Name, "__") {
		return
	}

	switch def.Name {
	case "Query":
		for _, f := range def.Fields {
			if _, exists := queryFields[f.Name]; !exists {
				queryFields[f.Name] = f
				result.QueryOwnership[f.Name] = serviceName
			}
		}
		return
	case "Mutation":
		for _, f := range def.Fields {
			if _, exists := mutationFields[f.Name]; !exists {
				mutationFields[f.Name] = f
				result.MutationOwnership[f.Name] = serviceName
			}
		}
		return
	case "Subscription":
		for _, f := range def.Fields {
			if _, exists := subscriptionFields[f.Name]; !exists {
				subscriptionFields[f.Name] = f
				result.SubscriptionOwnership[f.Name] = serviceName
			}
		}
		return
	}

	if def.Kind != ast.Object && def.Kind != ast.Interface &&
		def.Kind != ast.InputObject && def.Kind != ast.Union &&
		def.Kind != ast.Enum && def.Kind != ast.Scalar {
		return
	}

	// Record federation entity keys (@key directive)
	for _, dir := range def.Directives {
		if dir.Name != "key" {
			continue
		}
		fieldsArg := dir.Arguments.ForName("fields")
		if fieldsArg == nil {
			continue
		}
		keyFields := strings.Fields(fieldsArg.Value.Raw)

		if result.EntityKeys[def.Name] == nil {
			result.EntityKeys[def.Name] = make(map[string][]string)
		}
		result.EntityKeys[def.Name][serviceName] = keyFields
	}

	// Determine primary type ownership:
	// The primary owner is the first (non-extension) definition that has
	// non-@external fields. Extensions just contribute additional fields.
	isStubOnly := isExtension && allFieldsExternal(def)

	existing, exists := otherTypes[def.Name]
	if !exists {
		if !isStubOnly {
			// First real definition — record ownership
			result.TypeOwnership[def.Name] = serviceName
			otherTypes[def.Name] = stripFederationFromDef(def)
		}
		return
	}

	// Type already registered — merge additional non-@external fields
	// (happens in federation when multiple services extend a type)
	for _, f := range def.Fields {
		if f.Directives.ForName("external") != nil {
			continue
		}
		if existing.Fields.ForName(f.Name) == nil {
			existing.Fields = append(existing.Fields, f)
		}
	}
}

// allFieldsExternal returns true if every field in the definition is @external,
// meaning this service is just referencing a type it doesn't own.
func allFieldsExternal(def *ast.Definition) bool {
	if len(def.Fields) == 0 {
		return true
	}
	for _, f := range def.Fields {
		if f.Directives.ForName("external") == nil {
			return false
		}
	}
	return true
}

// stripFederationFromDef returns a copy of def with federation-specific
// directives removed from both the type and its fields.
func stripFederationFromDef(def *ast.Definition) *ast.Definition {
	stripped := *def

	// Strip type-level federation directives
	var cleanDirs ast.DirectiveList
	for _, d := range def.Directives {
		if !federationDirectiveNames[d.Name] {
			cleanDirs = append(cleanDirs, d)
		}
	}
	stripped.Directives = cleanDirs

	// Strip field-level federation directives and @external fields
	var cleanFields ast.FieldList
	for _, f := range def.Fields {
		if f.Directives.ForName("external") != nil {
			continue
		}
		cf := *f
		var cfd ast.DirectiveList
		for _, d := range f.Directives {
			if !federationDirectiveNames[d.Name] {
				cfd = append(cfd, d)
			}
		}
		cf.Directives = cfd
		cleanFields = append(cleanFields, &cf)
	}
	stripped.Fields = cleanFields

	return &stripped
}

// buildMergedSDL constructs the final merged SDL string from collected types.
func buildMergedSDL(
	queryFields, mutationFields, subscriptionFields map[string]*ast.FieldDefinition,
	otherTypes map[string]*ast.Definition,
) string {
	var b strings.Builder

	if len(queryFields) > 0 {
		b.WriteString("type Query {\n")
		for _, f := range queryFields {
			fieldDefToSDL(&b, f, "  ")
		}
		b.WriteString("}\n\n")
	}

	if len(mutationFields) > 0 {
		b.WriteString("type Mutation {\n")
		for _, f := range mutationFields {
			fieldDefToSDL(&b, f, "  ")
		}
		b.WriteString("}\n\n")
	}

	if len(subscriptionFields) > 0 {
		b.WriteString("type Subscription {\n")
		for _, f := range subscriptionFields {
			fieldDefToSDL(&b, f, "  ")
		}
		b.WriteString("}\n\n")
	}

	for _, def := range otherTypes {
		definitionToSDL(&b, def)
	}

	return b.String()
}

func jsonToHeaderMap(raw string) (map[string]string, error) {
	if raw == "" || raw == "{}" {
		return nil, nil
	}
	var h map[string]string
	return h, json.Unmarshal([]byte(raw), &h)
}
