package security

import (
	"bytes"
	"io"
	"net/http"
)

// RequestSizeLimitMiddleware rejects request bodies larger than MaxRequestBodyKB.
// It also reads the body into a buffer so graphqlHandler can read it twice
// (once for batch detection, once for decode).
func RequestSizeLimitMiddleware(mgr *Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cfg := mgr.Config()
			if cfg.MaxRequestBodyKB <= 0 || r.Method != http.MethodPost {
				next.ServeHTTP(w, r)
				return
			}

			maxBytes := int64(cfg.MaxRequestBodyKB) * 1024
			limited := io.LimitReader(r.Body, maxBytes+1)
			body, err := io.ReadAll(limited)
			if err != nil {
				http.Error(w, `{"errors":[{"message":"failed to read request body"}]}`,
					http.StatusBadRequest)
				return
			}
			if int64(len(body)) > maxBytes {
				ip := clientIP(r)
				mgr.LogBlocked("request_too_large", ip, r.URL.Path)
				http.Error(w, `{"errors":[{"message":"request body too large"}]}`,
					http.StatusRequestEntityTooLarge)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(body))
			next.ServeHTTP(w, r)
		})
	}
}

// CappedResponseWriter wraps ResponseWriter and stops writing after MaxResponseBodyKB.
type CappedResponseWriter struct {
	http.ResponseWriter
	MaxBytes int
	Written  int
	Capped   bool
}

func NewCappedWriter(w http.ResponseWriter, maxKB int) *CappedResponseWriter {
	return &CappedResponseWriter{ResponseWriter: w, MaxBytes: maxKB * 1024}
}

func (c *CappedResponseWriter) Write(p []byte) (int, error) {
	if c.Capped {
		return 0, nil
	}
	remaining := c.MaxBytes - c.Written
	if len(p) > remaining {
		c.Capped = true
		p = p[:remaining]
	}
	n, err := c.ResponseWriter.Write(p)
	c.Written += n
	return n, err
}
