package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

const defaultTimeoutMs = 30_000

var baseHTTPClient = &http.Client{} // no global timeout — per-request via context

type upstreamRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type upstreamResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []GQLError      `json:"errors,omitempty"`
}

// callUpstream sends a GraphQL request to url with optional retry and per-request timeout.
// retryCount = 0 → single attempt. timeoutMs = 0 → uses defaultTimeoutMs.
func callUpstream(
	ctx context.Context,
	log *zap.Logger,
	url string,
	headers map[string]string,
	query string,
	variables map[string]any,
	retryCount int,
	timeoutMs int,
) (*upstreamResponse, error) {
	if timeoutMs <= 0 {
		timeoutMs = defaultTimeoutMs
	}

	var lastErr error
	for attempt := 0; attempt <= retryCount; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 100ms, 200ms, 400ms…
			backoff := time.Duration(100*(1<<(attempt-1))) * time.Millisecond
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		resp, err := doUpstreamRequest(ctx, log, url, headers, query, variables, timeoutMs)
		if err == nil {
			return resp, nil
		}
		lastErr = err

		// Never retry on context cancellation or 4xx client errors.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		if isClientError(err) {
			return nil, err
		}

		log.Warn("upstream call failed, retrying",
			zap.String("url", url),
			zap.Int("attempt", attempt+1),
			zap.Int("max_retries", retryCount),
			zap.Error(err),
		)
	}
	return nil, lastErr
}

func doUpstreamRequest(
	ctx context.Context,
	log *zap.Logger,
	url string,
	headers map[string]string,
	query string,
	variables map[string]any,
	timeoutMs int,
) (*upstreamResponse, error) {
	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	body, err := json.Marshal(upstreamRequest{Query: query, Variables: variables})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	log.Debug("upstream request",
		zap.String("url", url),
		zap.Strings("headers", headerKeys(headers)),
	)

	resp, err := baseHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call %s: %w", url, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response from %s: %w", url, err)
	}

	log.Debug("upstream response",
		zap.String("url", url),
		zap.Int("status", resp.StatusCode),
		zap.Int("body_bytes", len(raw)),
	)

	if resp.StatusCode >= 500 {
		preview := string(raw)
		if len(preview) > 200 {
			preview = preview[:200] + "…"
		}
		return nil, &upstreamHTTPError{url: url, status: resp.StatusCode, body: preview}
	}

	if resp.StatusCode >= 400 {
		preview := string(raw)
		if len(preview) > 200 {
			preview = preview[:200] + "…"
		}
		return nil, &upstreamClientError{url: url, status: resp.StatusCode, body: preview}
	}

	var result upstreamResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response from %s: %w", url, err)
	}
	return &result, nil
}

// ── Error types ───────────────────────────────────────────────────────────────

type upstreamHTTPError struct{ url string; status int; body string }

func (e *upstreamHTTPError) Error() string {
	return fmt.Sprintf("upstream %s returned HTTP %d: %s", e.url, e.status, e.body)
}

type upstreamClientError struct{ url string; status int; body string }

func (e *upstreamClientError) Error() string {
	return fmt.Sprintf("upstream %s returned HTTP %d: %s", e.url, e.status, e.body)
}

func isClientError(err error) bool {
	var ce *upstreamClientError
	return errors.As(err, &ce)
}

func headerKeys(h map[string]string) []string {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	return keys
}
