package executor

// Result is the final merged GraphQL response.
type Result struct {
	Data   map[string]any `json:"data"`
	Errors []GQLError     `json:"errors,omitempty"`
}

// GQLError is a GraphQL-spec error object.
type GQLError struct {
	Message    string          `json:"message"`
	Locations  []ErrorLocation `json:"locations,omitempty"`
	Path       []any           `json:"path,omitempty"`
	Extensions map[string]any  `json:"extensions,omitempty"`
}

// ErrorLocation is a line/column pointer into a query document.
type ErrorLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}
