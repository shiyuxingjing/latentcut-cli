package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/novelo-ai/novelo-cli/internal/progress"
)

// WSClient manages a WebSocket connection for receiving progress events.
type WSClient struct {
	ServerURL string
	RunID     string
	APIKey    string
	OnMessage func(progress.WsProgressEvent)
	OnError   func(error)

	seen map[string]bool // dedup by timestamp:type
}

// NewWSClient creates a new WSClient.
func NewWSClient(serverURL, runID, apiKey string) *WSClient {
	return &WSClient{
		ServerURL: serverURL,
		RunID:     runID,
		APIKey:    apiKey,
		seen:      make(map[string]bool),
	}
}

// wsURL converts an HTTP/HTTPS server URL to a ws/wss URL for /ws/progress.
func (c *WSClient) wsURL() string {
	u := c.ServerURL
	u = strings.Replace(u, "http://", "ws://", 1)
	u = strings.Replace(u, "https://", "wss://", 1)
	parsed, err := url.Parse(u)
	if err != nil {
		return u + "/ws/progress"
	}
	parsed.Path = "/ws/progress"
	q := parsed.Query()
	q.Set("run_id", c.RunID)
	q.Set("api_key", c.APIKey)
	parsed.RawQuery = q.Encode()
	return parsed.String()
}

// Connect establishes the WebSocket connection with exponential backoff reconnection.
// It reads messages until the context is cancelled or a terminal event (complete/error) is received.
func (c *WSClient) Connect(ctx context.Context) error {
	const maxRetries = 5
	backoff := time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
		}

		done, err := c.readLoop(ctx)
		if err != nil && c.OnError != nil {
			c.OnError(fmt.Errorf("ws attempt %d: %w", attempt+1, err))
		}
		if done {
			return nil
		}
		if attempt == maxRetries {
			return fmt.Errorf("websocket failed after %d retries: %w", maxRetries, err)
		}
	}
	return nil
}

// readLoop connects and reads messages. Returns (done=true) when a terminal event is received.
func (c *WSClient) readLoop(ctx context.Context) (done bool, err error) {
	wsURL := c.wsURL()
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		return false, fmt.Errorf("dial %s: %w", wsURL, err)
	}
	defer conn.CloseNow()

	for {
		_, msg, err := conn.Read(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return true, nil
			}
			return false, err
		}

		var event progress.WsProgressEvent
		if err := json.Unmarshal(msg, &event); err != nil {
			continue
		}

		// Deduplicate replayed events by timestamp+type (same ms can have different event types)
		dedupeKey := fmt.Sprintf("%d:%s", event.Timestamp, event.Type)
		if c.seen[dedupeKey] {
			continue
		}
		c.seen[dedupeKey] = true

		if c.OnMessage != nil {
			c.OnMessage(event)
		}

		// Terminal events
		if event.Type == "complete" || (event.Type == "error" && !event.Recoverable) {
			conn.Close(websocket.StatusNormalClosure, "done")
			return true, nil
		}
	}
}

// Close is a no-op (connection is closed after readLoop exits).
func (c *WSClient) Close() error {
	return nil
}
