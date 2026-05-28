package router

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/deformal/kastql/internal/adminapi"
	"github.com/deformal/kastql/internal/auth"
	"github.com/deformal/kastql/internal/executor"
	"github.com/deformal/kastql/internal/metaapi"
	"github.com/deformal/kastql/internal/metadata"
	"github.com/deformal/kastql/internal/metrics"
	"github.com/deformal/kastql/internal/planner"
	"github.com/deformal/kastql/internal/playground"
	"github.com/deformal/kastql/internal/security"
)

type Server struct {
	http     *http.Server
	router   chi.Router
	log      *zap.Logger
	planner  *planner.Planner
	executor *executor.Executor
	meta     *metaapi.Handler
	store    *metadata.Store
	metrics  *metrics.Store
	secMgr   *security.Manager
}

func New(
	port int,
	log *zap.Logger,
	jwtMiddleware func(http.Handler) http.Handler,
	p *planner.Planner,
	exec *executor.Executor,
	meta *metaapi.Handler,
	store *metadata.Store,
	metricsStore *metrics.Store,
	adminHandler *adminapi.Handler,
	session *auth.SessionManager,
	secMgr *security.Manager,
) *Server {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	s := &Server{
		router:   r,
		log:      log,
		planner:  p,
		executor: exec,
		meta:     meta,
		store:    store,
		metrics:  metricsStore,
		secMgr:   secMgr,
		http: &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			Handler:      r,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 60 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
	}

	s.registerRoutes(jwtMiddleware, adminHandler, session)
	return s
}

func (s *Server) registerRoutes(jwtMW func(http.Handler) http.Handler, adminH *adminapi.Handler, session *auth.SessionManager) {
	assets := playground.Handler()

	// ── Public ────────────────────────────────────────────────────────────────
	s.router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	s.router.Handle("/assets/*", assets)
	s.router.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		assets.ServeHTTP(w, r)
	})

	// User auth (playground login)
	s.router.Get("/login", adminH.ServeUserLoginPage)
	s.router.Post("/login", adminH.HandleUserLogin)
	s.router.Post("/logout", adminH.HandleUserLogout)

	// Admin auth
	s.router.Get("/admin/login", adminH.ServeAdminLoginPage)
	s.router.Post("/admin/login", adminH.HandleAdminLogin)
	s.router.Post("/admin/logout", adminH.HandleAdminLogout)

	// ── Admin-protected ───────────────────────────────────────────────────────
	s.router.Group(func(r chi.Router) {
		r.Use(auth.AdminGuard(session))

		r.Post("/v1/metadata", s.meta.ServeHTTP)

		if s.metrics != nil {
			r.Get("/v1/metrics", metrics.NewHandler(s.metrics).ServeHTTP)
		}

		// Users
		r.Get("/v1/admin/users", adminH.ListUsers)
		r.Post("/v1/admin/users", adminH.CreateUser)
		r.Delete("/v1/admin/users/{id}", adminH.DeleteUser)

		// JWT secrets
		r.Get("/v1/admin/jwt-secrets", adminH.ListJWTSecrets)
		r.Post("/v1/admin/jwt-secrets", adminH.AddJWTSecret)
		r.Delete("/v1/admin/jwt-secrets/{id}", adminH.DeactivateJWTSecret)

		// Router keys
		r.Get("/v1/admin/router-keys", adminH.ListRouterKeys)
		r.Post("/v1/admin/router-keys", adminH.CreateRouterKey)
		r.Delete("/v1/admin/router-keys/{id}", adminH.DeactivateRouterKey)

		// Settings
		r.Get("/v1/admin/settings", adminH.GetSettings)
		r.Put("/v1/admin/settings/{key}", adminH.UpdateSetting)

		// CORS origins
		r.Get("/v1/admin/cors-origins", adminH.ListCORSOrigins)
		r.Post("/v1/admin/cors-origins", adminH.AddCORSOrigin)
		r.Delete("/v1/admin/cors-origins/{id}", adminH.DeleteCORSOrigin)

		// IP rules
		r.Get("/v1/admin/ip-rules", adminH.ListIPRules)
		r.Post("/v1/admin/ip-rules", adminH.AddIPRule)
		r.Delete("/v1/admin/ip-rules/{id}", adminH.DeleteIPRule)

		// Persisted queries
		r.Get("/v1/admin/persisted-queries", adminH.ListPersistedQueries)
		r.Post("/v1/admin/persisted-queries", adminH.AddPersistedQuery)
		r.Delete("/v1/admin/persisted-queries/{id}", adminH.DeletePersistedQuery)

		// Audit & blocked logs
		r.Get("/v1/admin/audit-log", adminH.ListAuditLog)
		r.Get("/v1/admin/blocked-requests", adminH.ListBlockedRequests)

		// Admin panel SPA
		r.Get("/admin", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/admin/", http.StatusFound)
		})
		r.Get("/admin/*", playground.ServeIndex)
	})

	// ── API routes (CORS + IP filter + rate limit + router key + JWT) ─────────
	s.router.Group(func(r chi.Router) {
		if s.secMgr != nil {
			r.Use(security.CORSMiddleware(s.secMgr))
			r.Use(security.IPFilterMiddleware(s.secMgr))
			r.Use(security.RateLimitMiddleware(s.secMgr))
			r.Use(security.RequestSizeLimitMiddleware(s.secMgr))
		}
		r.Use(auth.RouterKeyMiddleware(s.store))
		r.Use(jwtMW)

		gql := &graphqlHandler{
			planner:  s.planner,
			executor: s.executor,
			metrics:  s.metrics,
			log:      s.log,
			secMgr:   s.secMgr,
			introspectionEnabled: func() bool {
				val, _, err := s.store.GetSetting("introspection_enabled")
				if err != nil {
					return true
				}
				return val != "0"
			},
		}

		ws := &wsHandler{
			planner: s.planner,
			log:     s.log,
			secMgr:  s.secMgr,
		}

		r.Post("/graphql", gql.ServeHTTP)
		r.Get("/graphql", func(w http.ResponseWriter, r *http.Request) {
			if websocket.IsWebSocketUpgrade(r) {
				ws.ServeHTTP(w, r)
				return
			}
			writeGQLError(w, "use POST for queries/mutations or WebSocket for subscriptions", http.StatusMethodNotAllowed)
		})

		restFn := ServeREST(s.store, s.planner, s.executor, s.log)
		r.Get("/api/*", restFn)
		r.Post("/api/*", restFn)
		r.Put("/api/*", restFn)
		r.Patch("/api/*", restFn)
		r.Delete("/api/*", restFn)
	})

	// ── Playground (session required) ─────────────────────────────────────────
	s.router.Group(func(r chi.Router) {
		r.Use(auth.UserGuard(session))
		r.Get("/", playground.ServeIndex)
	})
}

func (s *Server) Router() chi.Router { return s.router }

func (s *Server) Start() error {
	s.log.Info("kastql listening", zap.String("addr", s.http.Addr))
	return s.http.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}
