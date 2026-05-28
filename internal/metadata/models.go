package metadata

import "time"

type ServiceType string

const (
	ServiceTypeFederation ServiceType = "federation"
	ServiceTypeStitching  ServiceType = "stitching"
)

type Service struct {
	ID        int64       `json:"id"`
	Name      string      `json:"name"`
	URL       string      `json:"url"`
	Type      ServiceType `json:"type"`
	Headers   string      `json:"headers"` // JSON map
	Enabled   bool        `json:"enabled"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

type Relationship struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	SourceService string    `json:"source_service"`
	SourceType    string    `json:"source_type"`
	SourceField   string    `json:"source_field"`
	TargetService string    `json:"target_service"`
	TargetType    string    `json:"target_type"`
	JoinConfig    string    `json:"join_config"` // JSON
	CreatedAt     time.Time `json:"created_at"`
}

type Permission struct {
	ID        int64     `json:"id"`
	Role      string    `json:"role"`
	Service   string    `json:"service"`
	TypeName  string    `json:"type_name"`
	FieldName string    `json:"field_name"`
	Allow     bool      `json:"allow"`
	Condition string    `json:"condition"` // JSON
	CreatedAt time.Time `json:"created_at"`
}

type RESTEndpoint struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Method       string    `json:"method"`
	Path         string    `json:"path"`
	GraphQLQuery string    `json:"graphql_query"`
	Variables    string    `json:"variables"` // JSON
	CreatedAt    time.Time `json:"created_at"`
}

type SchemaCache struct {
	ID          int64     `json:"id"`
	ServiceName string    `json:"service_name"`
	SDL         string    `json:"sdl"`
	FetchedAt   time.Time `json:"fetched_at"`
}
