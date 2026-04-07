package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/novelo-ai/novelo-cli/internal/types"
)

func TestTriggerPipeline_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer testkey" {
			t.Errorf("expected Bearer testkey, got %s", r.Header.Get("Authorization"))
		}

		var req types.PipelineRunRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if req.InputText == "" {
			t.Error("expected non-empty input_text")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(types.PipelineRunResponse{RunID: "abc-123", Status: "started"})
	}))
	defer ts.Close()

	c := NewHTTPClient(ts.URL, "testkey")
	resp, err := c.TriggerPipeline(context.Background(), types.PipelineRunRequest{InputText: "hello"})
	if err != nil {
		t.Fatalf("TriggerPipeline error: %v", err)
	}
	if resp.RunID != "abc-123" {
		t.Errorf("expected run_id abc-123, got %s", resp.RunID)
	}
}

func TestTriggerPipeline_Unauthorized(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer ts.Close()

	c := NewHTTPClient(ts.URL, "badkey")
	_, err := c.TriggerPipeline(context.Background(), types.PipelineRunRequest{InputText: "hello"})
	if err == nil {
		t.Fatal("expected error for 401")
	}
}

func TestTriggerPipeline_TooManyRequests(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "too many", http.StatusTooManyRequests)
	}))
	defer ts.Close()

	c := NewHTTPClient(ts.URL, "testkey")
	_, err := c.TriggerPipeline(context.Background(), types.PipelineRunRequest{InputText: "hello"})
	if err == nil {
		t.Fatal("expected error for 429")
	}
}
