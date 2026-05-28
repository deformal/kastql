package metaapi

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"github.com/deformal/kastql/internal/metadata"
	"github.com/deformal/kastql/internal/planner"
	"github.com/deformal/kastql/internal/registry"
)

type Handler struct {
	store    *metadata.Store
	registry *registry.Registry
	planner  *planner.Planner
	log      *zap.Logger
}

func New(store *metadata.Store, reg *registry.Registry, p *planner.Planner, log *zap.Logger) *Handler {
	return &Handler{store: store, registry: reg, planner: p, log: log}
}

type metaRequest struct {
	Type string          `json:"type"`
	Args json.RawMessage `json:"args"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req metaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	var (
		result any
		err    error
	)

	switch req.Type {
	case "add_remote_schema":
		result, err = h.addRemoteSchema(r.Context(), req.Args)
	case "remove_remote_schema":
		result, err = h.removeRemoteSchema(req.Args)
	case "reload_remote_schema":
		result, err = h.reloadRemoteSchema(r.Context(), req.Args)
	case "add_relationship":
		result, err = h.addRelationship(req.Args)
	case "remove_relationship":
		result, err = h.removeRelationship(req.Args)
	case "create_permission":
		result, err = h.createPermission(req.Args)
	case "drop_permission":
		result, err = h.dropPermission(req.Args)
	case "create_rest_endpoint":
		result, err = h.createRESTEndpoint(req.Args)
	case "drop_rest_endpoint":
		result, err = h.dropRESTEndpoint(req.Args)
	case "reload_metadata":
		result, err = h.reloadMetadata(r.Context())
	case "export_metadata":
		result, err = h.exportMetadata()
	default:
		writeError(w, "unknown action type: "+req.Type, http.StatusBadRequest)
		return
	}

	if err != nil {
		h.log.Warn("metadata action failed", zap.String("type", req.Type), zap.Error(err))
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

func writeError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// refreshPlanner re-merges all registry entries into the planner after a change.
func (h *Handler) refreshPlanner() {
	entries := h.registry.List()
	if len(entries) == 0 {
		return
	}
	if err := h.planner.Update(entries); err != nil {
		h.log.Warn("planner update after metadata change failed", zap.Error(err))
	}
}
