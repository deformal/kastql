package router

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"go.uber.org/zap"

	"github.com/deformal/kastql/internal/auth"
	"github.com/deformal/kastql/internal/executor"
	"github.com/deformal/kastql/internal/metadata"
	"github.com/deformal/kastql/internal/planner"
)

// ServeREST is the catch-all handler for saved REST endpoints.
// It matches the incoming method+path against stored patterns, extracts path
// params, merges with query/body vars, runs the stored GraphQL query, and
// returns the result JSON directly.
func ServeREST(store *metadata.Store, p *planner.Planner, exec *executor.Executor, log *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eps, err := store.ListRESTEndpoints()
		if err != nil {
			writeGQLError(w, "failed to load rest endpoints: "+err.Error(), http.StatusInternalServerError)
			return
		}

		var matched *metadata.RESTEndpoint
		var pathParams map[string]string

		for _, ep := range eps {
			if !strings.EqualFold(ep.Method, r.Method) {
				continue
			}
			if params, ok := matchPath(ep.Path, r.URL.Path); ok {
				matched = ep
				pathParams = params
				break
			}
		}

		if matched == nil {
			http.NotFound(w, r)
			return
		}

		vars := buildVars(matched, pathParams, r)

		role := auth.GetRole(r.Context())
		plan, err := p.Plan(r.Context(), matched.GraphQLQuery, vars, role)
		if err != nil {
			writeGQLError(w, err.Error(), http.StatusOK)
			return
		}

		result, err := exec.Execute(r.Context(), plan, forwardHeaders(r))
		if err != nil {
			writeGQLError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
	}
}

func buildVars(ep *metadata.RESTEndpoint, pathParams map[string]string, r *http.Request) map[string]any {
	vars := map[string]any{}
	if ep.Variables != "" && ep.Variables != "{}" {
		_ = json.Unmarshal([]byte(ep.Variables), &vars)
	}
	for k, v := range pathParams {
		vars[k] = v
	}
	for k, vs := range r.URL.Query() {
		vars[k] = vs[0]
	}
	if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
			for k, v := range body {
				vars[k] = v
			}
		}
	}
	return vars
}

// matchPath tests whether urlPath matches the pattern (e.g. /users/:id/orders/:orderId)
// and returns extracted parameters. Returns (nil, false) if no match.
func matchPath(pattern, urlPath string) (map[string]string, bool) {
	patParts := splitPath(pattern)
	urlParts := splitPath(urlPath)
	if len(patParts) != len(urlParts) {
		return nil, false
	}
	params := map[string]string{}
	for i, seg := range patParts {
		if strings.HasPrefix(seg, ":") {
			params[seg[1:]] = urlParts[i]
		} else if seg != urlParts[i] {
			return nil, false
		}
	}
	return params, true
}

var multiSlash = regexp.MustCompile(`/+`)

func splitPath(p string) []string {
	p = multiSlash.ReplaceAllString(p, "/")
	p = strings.Trim(p, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}
