package planner

import "github.com/vektah/gqlparser/v2/ast"

// MergedSchema is the unified view of all registered services.
type MergedSchema struct {
	SDL    string      // clean merged SDL for client-facing introspection
	Schema *ast.Schema // resolved schema for query validation and planning

	// Root field ownership: field name → service name
	QueryOwnership        map[string]string
	MutationOwnership     map[string]string
	SubscriptionOwnership map[string]string

	// Primary type owner: type name → service name
	// For federation entities this is the service with the non-@external definition.
	TypeOwnership map[string]string

	// Federation entity keys: type name → service name → key fields
	// e.g. EntityKeys["User"]["users-svc"] = ["id"]
	EntityKeys map[string]map[string][]string

	// Per-service metadata
	ServiceURLs    map[string]string            // name → URL
	ServiceTypes   map[string]string            // name → "federation"|"stitching"
	ServiceHeaders map[string]map[string]string // name → headers to send upstream
}

// QueryPlan describes how to execute a GraphQL operation across multiple services.
type QueryPlan struct {
	Steps         []*Step
	OperationType string // "query" | "mutation" | "subscription"
	OperationName string // named operation, or "" for anonymous
}

// Step is one upstream call inside a QueryPlan.
type Step struct {
	ID          string
	ServiceName string
	ServiceURL  string
	ServiceType string // "federation" | "stitching"

	Query     string         // sub-query to send to this service
	Variables map[string]any // variables for this step (may be subset of original)

	// Step IDs that must complete before this step can run.
	DependsOn []string

	// Where in the final merged response this step's data lands.
	// Empty = response root; ["users"] = response.data.users.
	MergePath []string

	Meta StepMeta
}

// StepKind classifies a Step.
type StepKind string

const (
	StepKindRoot   StepKind = "root"   // initial root-level query
	StepKindEntity StepKind = "entity" // federation _entities resolution
	StepKindJoin   StepKind = "join"   // stitching in-memory join
)

// StepMeta carries kind-specific planning data used by the executor.
type StepMeta struct {
	Kind   StepKind
	Entity *EntityMeta // non-nil when Kind == StepKindEntity
	Join   *JoinMeta   // non-nil when Kind == StepKindJoin
}

// EntityMeta describes a federation _entities resolution step.
type EntityMeta struct {
	TypeName     string   // e.g. "User"
	KeyFields    []string // e.g. ["id"]
	ParentStepID string   // step that provides the entity key values
	ParentPath   []string // path in parent result where the parent objects live
	Selection    string   // selection set to fetch, e.g. "{ name email }"
}

// JoinMeta describes a stitching in-memory join step.
type JoinMeta struct {
	RelationshipName string
	ParentStepID     string
	ParentKeyField   string // field in parent result used as join key
	TargetField      string // root field on target service to call
	TargetArgName    string // argument name to pass the join key value as
}
