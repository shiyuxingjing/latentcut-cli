package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/novelo-ai/novelo-cli/internal/types"
)

// HTTPClient handles HTTP communication with the Novelo server.
type HTTPClient struct {
	ServerURL string
	APIKey    string
	client    *http.Client
}

// NewHTTPClient creates a new HTTPClient.
func NewHTTPClient(serverURL, apiKey string) *HTTPClient {
	return &HTTPClient{
		ServerURL: serverURL,
		APIKey:    apiKey,
		client:    &http.Client{},
	}
}

// TriggerPipeline sends POST /pipeline/run and returns the run_id.
func (c *HTTPClient) TriggerPipeline(ctx context.Context, req types.PipelineRunRequest) (*types.PipelineRunResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.ServerURL+"/pipeline/run-full", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("invalid API key (401): check your api_key in ~/.novelo/config.yaml")
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("server at maximum concurrent runs (429): try again later")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("server returned %d", resp.StatusCode)
	}

	var result types.PipelineRunResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// HealthCheck calls GET /pipeline/health and returns the response body.
func (c *HTTPClient) HealthCheck(ctx context.Context) (map[string]any, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.ServerURL+"/pipeline/health", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("server unreachable at %s: %w", c.ServerURL, err)
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}
