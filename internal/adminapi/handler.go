package adminapi

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/deformal/kastql/internal/auth"
	"github.com/deformal/kastql/internal/metadata"
)

type Config struct {
	AdminUser     string
	AdminPassword string
}

// SecurityInvalidator is implemented by security.Manager.
// Defined as an interface here to avoid an import cycle (adminapi → security → metadata → adminapi).
type SecurityInvalidator interface {
	InvalidateCache()
}

type Handler struct {
	cfg      Config
	store    *metadata.Store
	session  *auth.SessionManager
	log      *zap.Logger
	secInv   SecurityInvalidator // optional; nil until SetSecurityInvalidator is called
}

func New(cfg Config, store *metadata.Store, session *auth.SessionManager, log *zap.Logger) *Handler {
	return &Handler{cfg: cfg, store: store, session: session, log: log}
}

// SetSecurityInvalidator wires the security manager cache invalidator.
// Call once after both adminapi.Handler and security.Manager are constructed.
func (h *Handler) SetSecurityInvalidator(inv SecurityInvalidator) {
	h.secInv = inv
}

func (h *Handler) invalidateSecurity() {
	if h.secInv != nil {
		h.secInv.InvalidateCache()
	}
}

// adminUsername returns the logged-in admin username from the session cookie.
func (h *Handler) adminUsername(r *http.Request) string {
	user, err := h.session.Validate(r, auth.AdminCookieName)
	if err != nil {
		return "unknown"
	}
	return user
}

func (h *Handler) auditLog(r *http.Request, action, detail string) {
	settings, err := h.store.AllSettings()
	if err != nil || settings["audit_log_enabled"] != "1" {
		return
	}
	admin := h.adminUsername(r)
	ip := clientIP(r)
	_ = h.store.AppendAuditLog(admin, action, detail, ip)
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}

// ── Admin login/logout ────────────────────────────────────────────────────────

func (h *Handler) ServeAdminLoginPage(w http.ResponseWriter, r *http.Request) {
	// Already logged in → go to admin
	if _, err := h.session.Validate(r, auth.AdminCookieName); err == nil {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(loginPage("Admin Login", "/admin/login", "kastql Admin")))
}

func (h *Handler) HandleAdminLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/admin/login?error=1", http.StatusFound)
		return
	}
	username := r.FormValue("username")
	password := r.FormValue("password")

	userOK := subtle.ConstantTimeCompare([]byte(username), []byte(h.cfg.AdminUser)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(password), []byte(h.cfg.AdminPassword)) == 1

	if !userOK || !passOK {
		http.Redirect(w, r, "/admin/login?error=1", http.StatusFound)
		return
	}

	h.session.Issue(w, auth.AdminCookieName, username)
	http.Redirect(w, r, "/admin", http.StatusFound)
}

func (h *Handler) HandleAdminLogout(w http.ResponseWriter, r *http.Request) {
	h.session.Clear(w, auth.AdminCookieName)
	http.Redirect(w, r, "/admin/login", http.StatusFound)
}

// ── User login/logout (playground users) ─────────────────────────────────────

func (h *Handler) ServeUserLoginPage(w http.ResponseWriter, r *http.Request) {
	if _, err := h.session.Validate(r, auth.UserCookieName); err == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(loginPage("Login", "/login", "kastql Playground")))
}

func (h *Handler) HandleUserLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/login?error=1", http.StatusFound)
		return
	}
	username := r.FormValue("username")
	password := r.FormValue("password")

	if _, err := h.store.AuthenticateUser(username, password); err != nil {
		http.Redirect(w, r, "/login?error=1", http.StatusFound)
		return
	}

	h.session.Issue(w, auth.UserCookieName, username)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *Handler) HandleUserLogout(w http.ResponseWriter, r *http.Request) {
	h.session.Clear(w, auth.UserCookieName)
	http.Redirect(w, r, "/login", http.StatusFound)
}

// ── User management (admin only) ──────────────────────────────────────────────

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.store.ListUsers()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	type userView struct {
		ID        int64  `json:"id"`
		Username  string `json:"username"`
		CreatedAt string `json:"created_at"`
	}
	out := make([]userView, 0, len(users))
	for _, u := range users {
		out = append(out, userView{ID: u.ID, Username: u.Username, CreatedAt: u.CreatedAt})
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if body.Username == "" || body.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username and password required"})
		return
	}

	u, err := h.store.CreateUser(body.Username, body.Password)
	if err != nil {
		if err == metadata.ErrUserExists {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "username already taken"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":         u.ID,
		"username":   u.Username,
		"created_at": u.CreatedAt,
	})
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user id"})
		return
	}
	if err := h.store.DeleteUser(id); err != nil {
		if err == metadata.ErrUserNotFound {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Settings ──────────────────────────────────────────────────────────────────

func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	m, err := h.store.AllSettings()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (h *Handler) UpdateSetting(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	var body struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if err := h.store.SetSetting(key, body.Value); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	h.invalidateSecurity()
	h.auditLog(r, "update_setting", key+"="+body.Value)
	writeJSON(w, http.StatusOK, map[string]string{"key": key, "value": body.Value})
}

// ── JWT Secrets ───────────────────────────────────────────────────────────────

func (h *Handler) ListJWTSecrets(w http.ResponseWriter, r *http.Request) {
	secrets, err := h.store.ListJWTSecrets()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if secrets == nil {
		secrets = []*metadata.JWTSecret{}
	}
	writeJSON(w, http.StatusOK, secrets)
}

func (h *Handler) AddJWTSecret(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name      string `json:"name"`
		Secret    string `json:"secret"`
		Algorithm string `json:"algorithm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if body.Name == "" || body.Secret == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and secret required"})
		return
	}
	s, err := h.store.AddJWTSecret(body.Name, body.Secret, body.Algorithm)
	if err != nil {
		if err == metadata.ErrNameTaken {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "name already taken"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, s)
}

func (h *Handler) DeactivateJWTSecret(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.store.DeactivateJWTSecret(id); err != nil {
		if err == metadata.ErrSecretNotFound {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "secret not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Router Keys ───────────────────────────────────────────────────────────────

func (h *Handler) ListRouterKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := h.store.ListRouterKeys()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if keys == nil {
		keys = []*metadata.RouterKey{}
	}
	writeJSON(w, http.StatusOK, keys)
}

func (h *Handler) CreateRouterKey(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if body.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name required"})
		return
	}

	rawKey, err := metadata.GenerateRouterKey()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	k, err := h.store.AddRouterKey(body.Name, rawKey)
	if err != nil {
		if err == metadata.ErrNameTaken {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "name already taken"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// raw key returned once — never retrievable again
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":         k.ID,
		"name":       k.Name,
		"active":     k.Active,
		"created_at": k.CreatedAt,
		"key":        rawKey,
	})
}

func (h *Handler) DeactivateRouterKey(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.store.DeactivateRouterKey(id); err != nil {
		if err == metadata.ErrRouterKeyNotFound {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "key not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── CORS Origins ──────────────────────────────────────────────────────────────

func (h *Handler) ListCORSOrigins(w http.ResponseWriter, r *http.Request) {
	origins, err := h.store.ListCORSOrigins()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if origins == nil {
		origins = []*metadata.CORSOrigin{}
	}
	writeJSON(w, http.StatusOK, origins)
}

func (h *Handler) AddCORSOrigin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Origin string `json:"origin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if body.Origin == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "origin required"})
		return
	}
	o, err := h.store.AddCORSOrigin(body.Origin)
	if err != nil {
		if err == metadata.ErrNameTaken {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "origin already exists"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	h.invalidateSecurity()
	h.auditLog(r, "add_cors_origin", body.Origin)
	writeJSON(w, http.StatusCreated, o)
}

func (h *Handler) DeleteCORSOrigin(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.store.DeleteCORSOrigin(id); err != nil {
		if err == metadata.ErrCORSOriginNotFound {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "origin not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	h.invalidateSecurity()
	h.auditLog(r, "delete_cors_origin", strconv.FormatInt(id, 10))
	w.WriteHeader(http.StatusNoContent)
}

// ── IP Rules ──────────────────────────────────────────────────────────────────

func (h *Handler) ListIPRules(w http.ResponseWriter, r *http.Request) {
	rules, err := h.store.ListIPRules()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if rules == nil {
		rules = []*metadata.IPRule{}
	}
	writeJSON(w, http.StatusOK, rules)
}

func (h *Handler) AddIPRule(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CIDR string `json:"cidr"`
		Mode string `json:"mode"`
		Note string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if body.CIDR == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cidr required"})
		return
	}
	if body.Mode == "" {
		body.Mode = "deny"
	}
	rule, err := h.store.AddIPRule(body.CIDR, body.Mode, body.Note)
	if err != nil {
		if err == metadata.ErrNameTaken {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "cidr already exists"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	h.invalidateSecurity()
	h.auditLog(r, "add_ip_rule", body.Mode+":"+body.CIDR)
	writeJSON(w, http.StatusCreated, rule)
}

func (h *Handler) DeleteIPRule(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.store.DeleteIPRule(id); err != nil {
		if err == metadata.ErrIPRuleNotFound {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "rule not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	h.invalidateSecurity()
	h.auditLog(r, "delete_ip_rule", strconv.FormatInt(id, 10))
	w.WriteHeader(http.StatusNoContent)
}

// ── Persisted Queries ─────────────────────────────────────────────────────────

func (h *Handler) ListPersistedQueries(w http.ResponseWriter, r *http.Request) {
	queries, err := h.store.ListPersistedQueries()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if queries == nil {
		queries = []*metadata.PersistedQuery{}
	}
	writeJSON(w, http.StatusOK, queries)
}

func (h *Handler) AddPersistedQuery(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Query string `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if body.ID == "" || body.Query == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id and query required"})
		return
	}
	if body.Name == "" {
		body.Name = body.ID
	}
	q, err := h.store.AddPersistedQuery(body.ID, body.Name, body.Query)
	if err != nil {
		if err == metadata.ErrPersistedQueryExists {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "query id already exists"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	h.auditLog(r, "add_persisted_query", body.ID)
	writeJSON(w, http.StatusCreated, q)
}

func (h *Handler) DeletePersistedQuery(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store.DeletePersistedQuery(id); err != nil {
		if err == metadata.ErrPersistedQueryNotFound {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "query not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	h.auditLog(r, "delete_persisted_query", id)
	w.WriteHeader(http.StatusNoContent)
}

// ── Audit & Blocked Logs ──────────────────────────────────────────────────────

func (h *Handler) ListAuditLog(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	entries, err := h.store.ListAuditLog(limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if entries == nil {
		entries = []*metadata.AuditEntry{}
	}
	writeJSON(w, http.StatusOK, entries)
}

func (h *Handler) ListBlockedRequests(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	entries, err := h.store.ListBlockedRequests(limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if entries == nil {
		entries = []*metadata.BlockedRequest{}
	}
	writeJSON(w, http.StatusOK, entries)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func loginPage(title, action, heading string) string {
	errorBlock := `<p class="error" id="err" style="display:none">Invalid username or password.</p>`
	// Show error if ?error=1 in URL — handled client-side via tiny inline script
	return `<!doctype html>
<html lang="en">
<head>
<meta charset="UTF-8"/>
<meta name="viewport" content="width=device-width,initial-scale=1"/>
<title>` + title + `</title>
<style>
*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
body{min-height:100vh;display:flex;align-items:center;justify-content:center;
  background:#0f172a;font-family:system-ui,sans-serif;color:#e2e8f0}
.card{background:#1e293b;border:1px solid #334155;border-radius:12px;
  padding:40px 36px;width:100%;max-width:380px;box-shadow:0 20px 60px #0008}
h1{font-size:1.4rem;font-weight:700;margin-bottom:6px;color:#f1f5f9}
.sub{font-size:.85rem;color:#94a3b8;margin-bottom:28px}
label{display:block;font-size:.8rem;font-weight:500;color:#94a3b8;margin-bottom:6px;margin-top:16px}
input{width:100%;padding:10px 12px;background:#0f172a;border:1px solid #334155;
  border-radius:8px;color:#f1f5f9;font-size:.95rem;outline:none;transition:border-color .15s}
input:focus{border-color:#e10098}
.error{background:#450a0a;border:1px solid #7f1d1d;color:#fca5a5;
  padding:10px 12px;border-radius:8px;font-size:.85rem;margin-top:16px}
button{width:100%;margin-top:24px;padding:11px;background:#e10098;color:#fff;
  border:none;border-radius:8px;font-size:.95rem;font-weight:600;cursor:pointer;transition:opacity .15s}
button:hover{opacity:.88}
.logo{text-align:center;font-size:1.1rem;font-weight:800;color:#e10098;
  letter-spacing:.05em;margin-bottom:24px}
</style>
</head>
<body>
<div class="card">
  <div class="logo">⚡ kastql</div>
  <h1>` + heading + `</h1>
  <p class="sub">Sign in to continue</p>
  ` + errorBlock + `
  <form method="POST" action="` + action + `">
    <label for="u">Username</label>
    <input id="u" name="username" type="text" autocomplete="username" autofocus required/>
    <label for="p">Password</label>
    <input id="p" name="password" type="password" autocomplete="current-password" required/>
    <button type="submit">Sign in</button>
  </form>
</div>
<script>
if(location.search.includes('error=1'))document.getElementById('err').style.display='block';
</script>
</body>
</html>`
}
