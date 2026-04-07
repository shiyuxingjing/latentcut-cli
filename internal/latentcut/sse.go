package latentcut

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// SSEEvent represents a parsed Server-Sent Event.
type SSEEvent struct {
	Name string
	Data string
	ID   string
}

// SSEHandler processes incoming SSE events. Return false to stop listening.
type SSEHandler func(event SSEEvent) bool

// SubscribeSSE connects to the latentCut-server SSE endpoint and streams events.
func (c *Client) SubscribeSSE(ctx context.Context, projectUUID, taskUUID string, handler SSEHandler) error {
	url := fmt.Sprintf("%s/api/projects/%s/events?token=%s", c.BaseURL, projectUUID, c.Token)
	if taskUUID != "" {
		url += "&task_uuid=" + taskUUID
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create SSE request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	httpClient := &http.Client{Timeout: 0} // no timeout for SSE
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("SSE connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SSE connect failed: status %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	// Increase scanner buffer for large SSE payloads
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	var current SSEEvent
	for scanner.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		line := scanner.Text()

		// Empty line = end of event
		if line == "" {
			if current.Data != "" || current.Name != "" {
				if !handler(current) {
					return nil
				}
				current = SSEEvent{}
			}
			continue
		}

		// Comment line (heartbeat)
		if strings.HasPrefix(line, ":") {
			continue
		}

		// Parse field
		if strings.HasPrefix(line, "event:") {
			current.Name = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if current.Data != "" {
				current.Data += "\n" + data
			} else {
				current.Data = data
			}
		} else if strings.HasPrefix(line, "id:") {
			current.ID = strings.TrimSpace(strings.TrimPrefix(line, "id:"))
		}
	}

	if err := scanner.Err(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("SSE read: %w", err)
	}

	return nil
}

// SubscribeSSEWithRetry wraps SubscribeSSE with reconnection logic.
func (c *Client) SubscribeSSEWithRetry(ctx context.Context, projectUUID, taskUUID string, handler SSEHandler, maxRetries int) error {
	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := c.SubscribeSSE(ctx, projectUUID, taskUUID, handler)
		if err == nil || ctx.Err() != nil {
			return err
		}
		if attempt < maxRetries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt+1) * 2 * time.Second):
			}
		}
	}
	return fmt.Errorf("SSE connection failed after %d retries", maxRetries)
}
