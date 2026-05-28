package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	AdminCookieName = "kastql_admin_session"
	UserCookieName  = "kastql_user_session"
	cookieTTL       = 24 * time.Hour
)

var ErrInvalidSession = errors.New("invalid or expired session")

type SessionManager struct {
	secret []byte
}

func NewSessionManager(secret string) *SessionManager {
	return &SessionManager{secret: []byte(secret)}
}

// GenerateSecret creates a random 32-byte hex secret for dev use.
func GenerateSecret() string {
	b := make([]byte, 32)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// Issue writes a signed session cookie to the response.
func (m *SessionManager) Issue(w http.ResponseWriter, cookieName, username string) {
	expiry := time.Now().Add(cookieTTL).Unix()
	payload := fmt.Sprintf("%s|%d", username, expiry)

	sig := m.sign(cookieName, payload)
	value := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + sig

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    value,
		Path:     "/",
		Expires:  time.Now().Add(cookieTTL),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

// Validate reads and verifies a session cookie, returning the username on success.
func (m *SessionManager) Validate(r *http.Request, cookieName string) (string, error) {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return "", ErrInvalidSession
	}

	parts := strings.SplitN(c.Value, ".", 2)
	if len(parts) != 2 {
		return "", ErrInvalidSession
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", ErrInvalidSession
	}
	payload := string(payloadBytes)

	if !hmac.Equal([]byte(parts[1]), []byte(m.sign(cookieName, payload))) {
		return "", ErrInvalidSession
	}

	idx := strings.LastIndex(payload, "|")
	if idx < 0 {
		return "", ErrInvalidSession
	}
	expiry, err := strconv.ParseInt(payload[idx+1:], 10, 64)
	if err != nil || time.Now().Unix() > expiry {
		return "", ErrInvalidSession
	}

	return payload[:idx], nil
}

// Clear removes a session cookie.
func (m *SessionManager) Clear(w http.ResponseWriter, cookieName string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}

func (m *SessionManager) sign(cookieName, payload string) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(cookieName + ":" + payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
