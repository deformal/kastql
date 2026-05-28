package metaapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/deformal/kastql/internal/metadata"
)

// ── add_remote_schema ─────────────────────────────────────────────────────────

type addRemoteSchemaArgs struct {
	Name    string            `json:"name"`
	URL     string            `json:"url"`
	Type    string            `json:"type"` // "federation" | "stitching"
	Headers map[string]string `json:"headers"`
	Enabled *bool             `json:"enabled"`
}

func (h *Handler) addRemoteSchema(ctx context.Context, raw json.RawMessage) (any, error) {
	var args addRemoteSchemaArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	if args.Name == "" || args.URL == "" {
		return nil, fmt.Errorf("name and url are required")
	}
	if args.Type == "" {
		args.Type = "stitching"
	}

	enabled := true
	if args.Enabled != nil {
		enabled = *args.Enabled
	}

	headersJSON := "{}"
	if len(args.Headers) > 0 {
		b, _ := json.Marshal(args.Headers)
		headersJSON = string(b)
	}

	svc := &metadata.Service{
		Name:    args.Name,
		URL:     args.URL,
		Type:    metadata.ServiceType(args.Type),
		Headers: headersJSON,
		Enabled: enabled,
	}

	if err := h.registry.Add(ctx, svc); err != nil {
		return nil, fmt.Errorf("add remote schema %s: %w", args.Name, err)
	}
	h.refreshPlanner()
	return map[string]string{"message": "remote schema added", "name": args.Name}, nil
}

// ── remove_remote_schema ──────────────────────────────────────────────────────

type nameArgs struct {
	Name string `json:"name"`
}

func (h *Handler) removeRemoteSchema(raw json.RawMessage) (any, error) {
	var args nameArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	if args.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if err := h.registry.Remove(args.Name); err != nil {
		return nil, err
	}
	h.refreshPlanner()
	return map[string]string{"message": "remote schema removed", "name": args.Name}, nil
}

// ── reload_remote_schema ──────────────────────────────────────────────────────

func (h *Handler) reloadRemoteSchema(ctx context.Context, raw json.RawMessage) (any, error) {
	var args nameArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	if args.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if err := h.registry.Reload(ctx, args.Name); err != nil {
		return nil, err
	}
	h.refreshPlanner()
	return map[string]string{"message": "remote schema reloaded", "name": args.Name}, nil
}

// ── add_relationship ──────────────────────────────────────────────────────────

type addRelationshipArgs struct {
	Name          string         `json:"name"`
	SourceService string         `json:"source_service"`
	SourceType    string         `json:"source_type"`
	SourceField   string         `json:"source_field"`
	TargetService string         `json:"target_service"`
	TargetType    string         `json:"target_type"`
	JoinConfig    map[string]any `json:"join_config"`
}

func (h *Handler) addRelationship(raw json.RawMessage) (any, error) {
	var args addRelationshipArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	if args.Name == "" || args.SourceType == "" || args.TargetService == "" || args.TargetType == "" {
		return nil, fmt.Errorf("name, source_type, target_service, and target_type are required")
	}

	joinJSON := "{}"
	if len(args.JoinConfig) > 0 {
		b, _ := json.Marshal(args.JoinConfig)
		joinJSON = string(b)
	}

	rel := &metadata.Relationship{
		Name:          args.Name,
		SourceService: args.SourceService,
		SourceType:    args.SourceType,
		SourceField:   args.SourceField,
		TargetService: args.TargetService,
		TargetType:    args.TargetType,
		JoinConfig:    joinJSON,
	}

	if err := h.store.UpsertRelationship(rel); err != nil {
		return nil, err
	}
	return map[string]string{"message": "relationship added", "name": args.Name}, nil
}

// ── remove_relationship ───────────────────────────────────────────────────────

func (h *Handler) removeRelationship(raw json.RawMessage) (any, error) {
	var args nameArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	if args.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if err := h.store.DeleteRelationship(args.Name); err != nil {
		return nil, err
	}
	return map[string]string{"message": "relationship removed", "name": args.Name}, nil
}

// ── create_permission ─────────────────────────────────────────────────────────

type createPermissionArgs struct {
	Role      string `json:"role"`
	Service   string `json:"service"`
	TypeName  string `json:"type_name"`
	FieldName string `json:"field_name"`
	Allow     *bool  `json:"allow"`
}

func (h *Handler) createPermission(raw json.RawMessage) (any, error) {
	var args createPermissionArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	if args.Role == "" || args.TypeName == "" {
		return nil, fmt.Errorf("role and type_name are required")
	}

	allow := true
	if args.Allow != nil {
		allow = *args.Allow
	}

	perm := &metadata.Permission{
		Role:      args.Role,
		Service:   args.Service,
		TypeName:  args.TypeName,
		FieldName: args.FieldName,
		Allow:     allow,
	}

	if err := h.store.UpsertPermission(perm); err != nil {
		return nil, err
	}
	return map[string]string{"message": "permission created"}, nil
}

// ── drop_permission ───────────────────────────────────────────────────────────

type dropPermissionArgs struct {
	Role      string `json:"role"`
	Service   string `json:"service"`
	TypeName  string `json:"type_name"`
	FieldName string `json:"field_name"`
}

func (h *Handler) dropPermission(raw json.RawMessage) (any, error) {
	var args dropPermissionArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	if args.Role == "" || args.TypeName == "" {
		return nil, fmt.Errorf("role and type_name are required")
	}
	if err := h.store.DeletePermission(args.Role, args.Service, args.TypeName, args.FieldName); err != nil {
		return nil, err
	}
	return map[string]string{"message": "permission dropped"}, nil
}

// ── create_rest_endpoint ──────────────────────────────────────────────────────

type createRESTEndpointArgs struct {
	Name         string         `json:"name"`
	Method       string         `json:"method"`
	Path         string         `json:"path"`
	GraphQLQuery string         `json:"graphql_query"`
	Variables    map[string]any `json:"variables"`
}

func (h *Handler) createRESTEndpoint(raw json.RawMessage) (any, error) {
	var args createRESTEndpointArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	if args.Name == "" || args.Method == "" || args.Path == "" || args.GraphQLQuery == "" {
		return nil, fmt.Errorf("name, method, path, and graphql_query are required")
	}

	varsJSON := "{}"
	if len(args.Variables) > 0 {
		b, _ := json.Marshal(args.Variables)
		varsJSON = string(b)
	}

	ep := &metadata.RESTEndpoint{
		Name:         args.Name,
		Method:       args.Method,
		Path:         args.Path,
		GraphQLQuery: args.GraphQLQuery,
		Variables:    varsJSON,
	}

	if err := h.store.UpsertRESTEndpoint(ep); err != nil {
		return nil, err
	}
	return map[string]string{"message": "rest endpoint created", "path": args.Method + " " + args.Path}, nil
}

// ── drop_rest_endpoint ────────────────────────────────────────────────────────

func (h *Handler) dropRESTEndpoint(raw json.RawMessage) (any, error) {
	var args nameArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	if args.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if err := h.store.DeleteRESTEndpoint(args.Name); err != nil {
		return nil, err
	}
	return map[string]string{"message": "rest endpoint dropped", "name": args.Name}, nil
}

// ── reload_metadata ───────────────────────────────────────────────────────────

func (h *Handler) reloadMetadata(ctx context.Context) (any, error) {
	if err := h.registry.ReloadAll(ctx); err != nil {
		return nil, err
	}
	h.refreshPlanner()
	return map[string]string{"message": "metadata reloaded"}, nil
}

// ── export_metadata ───────────────────────────────────────────────────────────

type exportedMetadata struct {
	Services      []*metadata.Service      `json:"services"`
	Relationships []*metadata.Relationship `json:"relationships"`
	Permissions   []*metadata.Permission   `json:"permissions"`
	RESTEndpoints []*metadata.RESTEndpoint `json:"rest_endpoints"`
}

func (h *Handler) exportMetadata() (any, error) {
	svcs, err := h.store.ListServices()
	if err != nil {
		return nil, fmt.Errorf("list services: %w", err)
	}
	rels, err := h.store.ListRelationships()
	if err != nil {
		return nil, fmt.Errorf("list relationships: %w", err)
	}
	perms, err := h.store.ListPermissions()
	if err != nil {
		return nil, fmt.Errorf("list permissions: %w", err)
	}
	eps, err := h.store.ListRESTEndpoints()
	if err != nil {
		return nil, fmt.Errorf("list rest endpoints: %w", err)
	}
	return &exportedMetadata{
		Services:      svcs,
		Relationships: rels,
		Permissions:   perms,
		RESTEndpoints: eps,
	}, nil
}
