package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

type upstreamRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type upstreamResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []GQLError      `json:"errors,omitempty"`
}

func callUpstream(ctx context.Context, log *zap.Logger, url string, headers map[string]string, query string, variables map[string]any) (*upstreamResponse, error) {
	body, err := json.Marshal(upstreamRequest{Query: query, Variables: variables})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
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

	resp, err := httpClient.Do(req)
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

	if resp.StatusCode != http.StatusOK {
		// Surface the HTTP error clearly so it shows up in logs and in the
		// GraphQL response rather than as a JSON parse error.
		preview := string(raw)
		if len(preview) > 200 {
			preview = preview[:200] + "…"
		}
		log.Warn("upstream returned non-200",
			zap.String("url", url),
			zap.Int("status", resp.StatusCode),
			zap.String("body", preview),
		)
		return nil, fmt.Errorf("upstream %s returned HTTP %d: %s", url, resp.StatusCode, preview)
	}

	var result upstreamResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response from %s: %w", url, err)
	}
	return &result, nil
}

func headerKeys(h map[string]string) []string {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	return keys
}
