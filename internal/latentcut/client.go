package latentcut

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client handles HTTP communication with latentCut-server.
type Client struct {
	BaseURL string
	Token   string
	http    *http.Client
}

// NewClient creates a new latentCut-server client.
func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// doJSON sends a JSON request and decodes the APIResponse.
func (c *Client) doJSON(ctx context.Context, method, path string, body any) (*APIResponse, []byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bodyReader)
	if err != nil {
		return nil, nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("X-API-Key", c.Token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, rawBody, fmt.Errorf("unauthorized (401): token expired or invalid, run: novelo-cli login")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, rawBody, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(rawBody))
	}

	var apiResp APIResponse
	if err := json.Unmarshal(rawBody, &apiResp); err != nil {
		return nil, rawBody, fmt.Errorf("decode response: %w", err)
	}

	if apiResp.Code != 0 {
		return &apiResp, rawBody, fmt.Errorf("API error (code %d): %s", apiResp.Code, apiResp.Message)
	}

	return &apiResp, rawBody, nil
}

// decodeData extracts and decodes the data field from an APIResponse raw body.
func decodeData[T any](rawBody []byte) (*T, error) {
	var wrapper struct {
		Data T `json:"data"`
	}
	if err := json.Unmarshal(rawBody, &wrapper); err != nil {
		return nil, fmt.Errorf("decode data: %w", err)
	}
	return &wrapper.Data, nil
}

// Login authenticates and returns the JWT token.
func (c *Client) Login(ctx context.Context, account, password string) (*LoginData, error) {
	_, raw, err := c.doJSON(ctx, http.MethodPost, "/api/user/login", LoginRequest{
		Account:  account,
		Password: password,
	})
	if err != nil {
		return nil, err
	}
	return decodeData[LoginData](raw)
}

// CreateAPIKey creates a named API key using the given JWT token for auth.
// The JWT is sent as Authorization: Bearer since it is a short-lived login token.
// The returned api_key becomes the persistent credential for subsequent calls.
func (c *Client) CreateAPIKey(ctx context.Context, jwtToken, name string) (*CreateAPIKeyData, error) {
	body, err := json.Marshal(CreateAPIKeyRequest{Name: name})
	if err != nil {
		return nil, fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/user/api-keys", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwtToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("create API key failed (%d): %s", resp.StatusCode, string(rawBody))
	}

	return decodeData[CreateAPIKeyData](rawBody)
}

// CreateProject creates a new project and triggers AI parsing.
func (c *Client) CreateProject(ctx context.Context, title, novelContent, style string) (*CreateProjectData, error) {
	_, raw, err := c.doJSON(ctx, http.MethodPost, "/api/projects", CreateProjectRequest{
		Title:        title,
		NovelContent: novelContent,
		Style:        style,
		ProjectMode:  "short_drama",
	})
	if err != nil {
		return nil, err
	}
	return decodeData[CreateProjectData](raw)
}

// GetCanvasData retrieves the full project canvas structure.
func (c *Client) GetCanvasData(ctx context.Context, projectUUID string) (*CanvasData, error) {
	_, raw, err := c.doJSON(ctx, http.MethodGet, "/api/projects/"+projectUUID+"/canvas-data", nil)
	if err != nil {
		return nil, err
	}
	return decodeData[CanvasData](raw)
}

// PreviewShotBatch previews the batch generation plan.
func (c *Client) PreviewShotBatch(ctx context.Context, projectUUID string) (*ShotBatchPreview, error) {
	_, raw, err := c.doJSON(ctx, http.MethodGet, "/api/projects/"+projectUUID+"/generate/shot-batch/preview", nil)
	if err != nil {
		return nil, err
	}
	return decodeData[ShotBatchPreview](raw)
}

// CreateShotBatch triggers batch generation for all (or N) shots.
func (c *Client) CreateShotBatch(ctx context.Context, projectUUID, mode string, count int) (*ShotBatchData, error) {
	req := ShotBatchRequest{Mode: mode}
	if mode == "count" && count > 0 {
		req.Count = count
	}
	_, raw, err := c.doJSON(ctx, http.MethodPost, "/api/projects/"+projectUUID+"/generate/shot-batch", req)
	if err != nil {
		return nil, err
	}
	return decodeData[ShotBatchData](raw)
}

// GetPendingTasks retrieves pending generation tasks for a project.
func (c *Client) GetPendingTasks(ctx context.Context, projectUUID string) ([]PendingTask, error) {
	_, raw, err := c.doJSON(ctx, http.MethodGet, "/api/projects/"+projectUUID+"/pending-tasks", nil)
	if err != nil {
		return nil, err
	}
	result, err := decodeData[[]PendingTask](raw)
	if err != nil {
		return nil, err
	}
	return *result, nil
}

// GenerateEpisodeVideo triggers episode video concatenation.
func (c *Client) GenerateEpisodeVideo(ctx context.Context, projectUUID, episodeUUID string) error {
	_, _, err := c.doJSON(ctx, http.MethodPost, "/api/projects/"+projectUUID+"/generate/episode-video", GenerateEpisodeVideoRequest{
		EpisodeUUID: episodeUUID,
	})
	return err
}

// GetTaskStatus checks a specific task's status.
func (c *Client) GetTaskStatus(ctx context.Context, taskUUID string) (*TaskStatus, error) {
	_, raw, err := c.doJSON(ctx, http.MethodGet, "/api/tasks/"+taskUUID+"/status", nil)
	if err != nil {
		return nil, err
	}
	return decodeData[TaskStatus](raw)
}

// GenerateCharacterImage triggers image generation for a character.
func (c *Client) GenerateCharacterImage(ctx context.Context, projectUUID, characterUUID string) error {
	_, _, err := c.doJSON(ctx, http.MethodPost, "/api/projects/"+projectUUID+"/generate/character-image", map[string]string{
		"characterUuid": characterUUID,
	})
	return err
}

// GenerateCharacterVoice triggers TTS voice generation for a character.
func (c *Client) GenerateCharacterVoice(ctx context.Context, projectUUID, characterUUID string) error {
	_, _, err := c.doJSON(ctx, http.MethodPost, "/api/projects/"+projectUUID+"/generate/character-voice", map[string]string{
		"characterUuid": characterUUID,
	})
	return err
}

// GenerateLocationImage triggers image generation for a location.
func (c *Client) GenerateLocationImage(ctx context.Context, projectUUID, locationUUID string) error {
	_, _, err := c.doJSON(ctx, http.MethodPost, "/api/projects/"+projectUUID+"/generate/locationtime-image", map[string]string{
		"locationtimeUuid": locationUUID,
	})
	return err
}

// ChatStreamRequest is the body for the creative-video-agent stream endpoint.
type ChatStreamRequest struct {
	Input      string        `json:"input"`
	ThreadID   string        `json:"threadId,omitempty"`
	ResourceID string        `json:"resourceId,omitempty"`
	Messages   []ChatMessage `json:"messages,omitempty"`
}

// ChatStreamResult holds the aggregated result from a chat stream.
type ChatStreamResult struct {
	Text     string `json:"text"`
	ThreadID string `json:"threadId"`
}

// ChatStream sends a message to creative-video-agent via SSE streaming.
// It calls onChunk for each text chunk received, and returns the full aggregated result.
// history contains previous conversation messages for context.
func (c *Client) ChatStream(ctx context.Context, message, threadID string, history []ChatMessage, onChunk func(text string)) (*ChatStreamResult, error) {
	reqBody := ChatStreamRequest{
		Input:    message,
		ThreadID: threadID,
		Messages: history,
	}

	bodyData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/ai/creative-video-agent/stream", bytes.NewReader(bodyData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if c.Token != "" {
		req.Header.Set("X-API-Key", c.Token)
	}

	httpClient := &http.Client{Timeout: 0} // no timeout for SSE
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chat stream connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized (401): run novelo-cli login")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("chat stream failed: status %d: %s", resp.StatusCode, string(body))
	}

	// Read streaming response — handles both plain text and SSE format
	result := &ChatStreamResult{ThreadID: threadID}
	buf := make([]byte, 4096)

	for {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}

		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			chunk := string(buf[:n])

			// Check if this is SSE format (starts with "data:" or "event:")
			if strings.HasPrefix(chunk, "data:") || strings.HasPrefix(chunk, "event:") {
				// Parse SSE: extract text from data lines
				for _, line := range strings.Split(chunk, "\n") {
					line = strings.TrimSpace(line)
					if !strings.HasPrefix(line, "data:") {
						continue
					}
					data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
					if data == "" || data == "[DONE]" {
						continue
					}
					var obj map[string]any
					if json.Unmarshal([]byte(data), &obj) == nil {
						for _, key := range []string{"text", "delta", "content", "token"} {
							if v, ok := obj[key]; ok {
								if s, ok := v.(string); ok && s != "" {
									result.Text += s
									if onChunk != nil {
										onChunk(s)
									}
									break
								}
							}
						}
						if tid, ok := obj["threadId"]; ok {
							if s, ok := tid.(string); ok && s != "" {
								result.ThreadID = s
							}
						}
					}
				}
			} else {
				// Plain text stream
				result.Text += chunk
				if onChunk != nil {
					onChunk(chunk)
				}
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			if ctx.Err() != nil {
				return result, ctx.Err()
			}
			return result, fmt.Errorf("stream read: %w", readErr)
		}
	}

	return result, nil
}

// Chat sends a synchronous message to creative-video-agent (non-streaming).
func (c *Client) Chat(ctx context.Context, message, threadID string) (*ChatStreamResult, error) {
	_, raw, err := c.doJSON(ctx, http.MethodPost, "/api/ai/creative-video-agent/chat", map[string]string{
		"message":  message,
		"threadId": threadID,
	})
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data struct {
			Text     string `json:"text"`
			ThreadID string `json:"threadId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("decode chat response: %w", err)
	}
	return &ChatStreamResult{
		Text:     resp.Data.Text,
		ThreadID: resp.Data.ThreadID,
	}, nil
}

// RedeemCode redeems a credit code.
func (c *Client) RedeemCode(ctx context.Context, code string) error {
	_, _, err := c.doJSON(ctx, http.MethodPost, "/api/credits/redeem-codes/redeem", map[string]string{
		"code": code,
	})
	return err
}

// GetCreditsBalance retrieves the user's credit balance.
func (c *Client) GetCreditsBalance(ctx context.Context) (map[string]any, error) {
	_, raw, err := c.doJSON(ctx, http.MethodGet, "/api/credits/balance", nil)
	if err != nil {
		return nil, err
	}
	result, err := decodeData[map[string]any](raw)
	if err != nil {
		return nil, err
	}
	return *result, nil
}

// GetProject retrieves full project details.
func (c *Client) GetProject(ctx context.Context, projectUUID string) (map[string]any, error) {
	_, raw, err := c.doJSON(ctx, http.MethodGet, "/api/projects/"+projectUUID, nil)
	if err != nil {
		return nil, err
	}
	result, err := decodeData[map[string]any](raw)
	if err != nil {
		return nil, err
	}
	return *result, nil
}

// ListProjects retrieves all projects for the current user.
func (c *Client) ListProjects(ctx context.Context) ([]map[string]any, error) {
	_, raw, err := c.doJSON(ctx, http.MethodGet, "/api/projects", nil)
	if err != nil {
		return nil, err
	}
	// Server returns {data: {list: [...]}} (paginated)
	type listWrapper struct {
		List []map[string]any `json:"list"`
	}
	result, err := decodeData[listWrapper](raw)
	if err != nil {
		return nil, err
	}
	return result.List, nil
}

// DeleteProject deletes a project.
func (c *Client) DeleteProject(ctx context.Context, projectUUID string) error {
	_, _, err := c.doJSON(ctx, http.MethodDelete, "/api/projects/"+projectUUID, nil)
	return err
}

// GenerateKeyframeImage triggers keyframe image generation.
func (c *Client) GenerateKeyframeImage(ctx context.Context, projectUUID, keyframeUUID string) error {
	_, _, err := c.doJSON(ctx, http.MethodPost, "/api/projects/"+projectUUID+"/generate/keyframe-image", map[string]string{
		"keyframeUuid": keyframeUUID,
	})
	return err
}

// GenerateDialogueAudio triggers dialogue audio generation.
func (c *Client) GenerateDialogueAudio(ctx context.Context, projectUUID, dialogueUUID string) error {
	_, _, err := c.doJSON(ctx, http.MethodPost, "/api/projects/"+projectUUID+"/generate/dialogue-audio", map[string]string{
		"dialogueUuid": dialogueUUID,
	})
	return err
}

// GenerateShotVideo triggers video generation for a shot.
func (c *Client) GenerateShotVideo(ctx context.Context, projectUUID, shotUUID string) error {
	_, _, err := c.doJSON(ctx, http.MethodPost, "/api/projects/"+projectUUID+"/generate/shot-video", map[string]string{
		"shotUuid": shotUUID,
	})
	return err
}

// GenerateShotStoryboard triggers storyboard generation for a shot.
func (c *Client) GenerateShotStoryboard(ctx context.Context, projectUUID, shotUUID string) error {
	_, _, err := c.doJSON(ctx, http.MethodPost, "/api/projects/"+projectUUID+"/generate/shot-storyboard", map[string]string{
		"shotUuid": shotUUID,
	})
	return err
}
