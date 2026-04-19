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

// GetProjectProgress fetches the flat progress snapshot for a project.
//
// It first tries GET /api/projects/:uuid/progress (added in server >= 2026-04).
// On older servers that return 404 for the new route, it falls back to
// composing a snapshot from existing endpoints (project list + pending tasks).
// This keeps the CLI usable against both current and future servers without
// requiring a coordinated release.
func (c *Client) GetProjectProgress(ctx context.Context, projectUUID string) (*ProjectProgress, error) {
	_, raw, err := c.doJSON(ctx, http.MethodGet, "/api/projects/"+projectUUID+"/progress", nil)
	if err == nil {
		return decodeData[ProjectProgress](raw)
	}
	// Fallback path for servers without the new endpoint.
	// We detect "route not found" either as HTTP 404 or as server
	// payloads that include "404", "not found" or "ROUTE_NOT_FOUND".
	msg := strings.ToLower(err.Error())
	isRouteMissing := strings.Contains(msg, "404") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "route_not_found")
	if !isRouteMissing {
		return nil, err
	}
	return c.composeProjectProgress(ctx, projectUUID)
}

// composeProjectProgress fabricates a ProjectProgress from legacy endpoints
// (list projects + pending tasks). Used when the dedicated /progress route
// is missing. Slower (2 requests instead of 1) but functionally equivalent.
func (c *Client) composeProjectProgress(ctx context.Context, projectUUID string) (*ProjectProgress, error) {
	projects, err := c.ListProjects(ctx)
	if err != nil {
		return nil, fmt.Errorf("list projects for progress: %w", err)
	}
	var p map[string]any
	for _, item := range projects {
		if getStr(item, "project_uuid") == projectUUID {
			p = item
			break
		}
	}
	if p == nil {
		return nil, fmt.Errorf("project %s not found", projectUUID)
	}

	tasks, err := c.GetPendingTasks(ctx, projectUUID)
	if err != nil {
		// Non-fatal: we can still report the snapshot without task info.
		tasks = nil
	}

	byType := map[string]int{}
	byStatus := map[string]int{}
	for _, t := range tasks {
		byType[t.TaskType]++
		byStatus[t.TaskStatus]++
	}

	metadata, _ := p["metadata"].(map[string]any)
	assetParse := buildPhaseState(metadata, "asset_parse")
	shotParse := buildPhaseState(metadata, "shot_parse")

	// Derive current_phase with the same rules as the server endpoint so
	// the fallback output shape stays consistent.
	status := getStr(p, "status")
	currentPhase := "unknown"
	var phaseStep string
	var phaseProgress *float64
	switch status {
	case "completed":
		currentPhase = "completed"
	case "failed":
		currentPhase = "failed"
	default:
		if assetParse != nil && assetParse.Status != "completed" {
			currentPhase = "asset_parse"
			phaseStep = assetParse.CurrentStep
			phaseProgress = assetParse.Progress
		} else if shotParse != nil && shotParse.Status != "" && shotParse.Status != "completed" {
			currentPhase = "shot_parse"
			phaseStep = shotParse.CurrentStep
			phaseProgress = shotParse.Progress
		} else if len(tasks) > 0 {
			top := ""
			best := 0
			for t, n := range byType {
				if n > best {
					top, best = t, n
				}
			}
			if top != "" {
				currentPhase = "generating:" + top
			} else {
				currentPhase = "generating"
			}
		} else {
			currentPhase = "idle"
		}
	}

	snapshot := &ProjectProgress{
		ProjectUUID:     projectUUID,
		Title:           getStr(p, "title"),
		Status:          status,
		OverallProgress: getFloat(p, "progress"),
		CurrentPhase:    currentPhase,
		PhaseStep:       phaseStep,
		PhaseProgress:   phaseProgress,
		Episodes: ProjectProgressEpisodes{
			Total:  int(getFloat(p, "episode_count")),
			Parsed: nil,
		},
		Shots: ProjectProgressShots{
			Total: int(getFloat(p, "shot_count")),
		},
		AssetParse: assetParse,
		ShotParse:  shotParse,
		PendingTasks: PendingTasksSummary{
			Total:    len(tasks),
			ByType:   byType,
			ByStatus: byStatus,
		},
		UpdatedAt: getStr(p, "updated_at"),
	}

	if assetParse != nil && assetParse.CompletedEpisodes != nil {
		v := *assetParse.CompletedEpisodes
		snapshot.Episodes.Parsed = &v
	}

	return snapshot, nil
}

// buildPhaseState extracts a phase-state sub-object from project metadata.
func buildPhaseState(metadata map[string]any, key string) *ProjectPhaseState {
	if metadata == nil {
		return nil
	}
	raw, ok := metadata[key].(map[string]any)
	if !ok {
		return nil
	}
	ps := &ProjectPhaseState{
		Status:       getStr(raw, "status"),
		CurrentStep:  getStr(raw, "current_step"),
		ErrorMessage: getStr(raw, "error_message"),
		UpdatedAt:    getStr(raw, "updated_at"),
	}
	if v, ok := raw["progress"]; ok {
		if f, ok := toFloat(v); ok {
			ps.Progress = &f
		}
	}
	if v, ok := raw["completed_episodes"]; ok {
		if f, ok := toFloat(v); ok {
			n := int(f)
			ps.CompletedEpisodes = &n
		}
	}
	if v, ok := raw["total_episodes"]; ok {
		if f, ok := toFloat(v); ok {
			n := int(f)
			ps.TotalEpisodes = &n
		}
	}
	if v, ok := raw["retryEligible"].(bool); ok {
		ps.RetryEligible = &v
	}
	return ps
}

func getStr(m map[string]any, k string) string {
	if v, ok := m[k]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getFloat(m map[string]any, k string) float64 {
	if v, ok := m[k]; ok {
		if f, ok := toFloat(v); ok {
			return f
		}
	}
	return 0
}

func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case json.Number:
		f, err := x.Float64()
		return f, err == nil
	}
	return 0, false
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

// GetAssistantSummary calls GET /api/projects/:uuid/assistant-summary
func (c *Client) GetAssistantSummary(ctx context.Context, projectUUID string) (*AssistantSummary, error) {
	_, raw, err := c.doJSON(ctx, http.MethodGet, "/api/projects/"+projectUUID+"/assistant-summary", nil)
	if err != nil {
		return nil, err
	}
	return decodeData[AssistantSummary](raw)
}

// GetGallery calls GET /api/projects/:uuid/gallery
func (c *Client) GetGallery(ctx context.Context, projectUUID string) (*Gallery, error) {
	_, raw, err := c.doJSON(ctx, http.MethodGet, "/api/projects/"+projectUUID+"/gallery", nil)
	if err != nil {
		return nil, err
	}
	return decodeData[Gallery](raw)
}

// GetResourceDetail calls GET /api/projects/:uuid/resources/:resourceUuid
func (c *Client) GetResourceDetail(ctx context.Context, projectUUID, resourceUUID string) (*GalleryResource, error) {
	_, raw, err := c.doJSON(ctx, http.MethodGet, "/api/projects/"+projectUUID+"/resources/"+resourceUUID, nil)
	if err != nil {
		return nil, err
	}
	return decodeData[GalleryResource](raw)
}

// PreviewWorkflow calls POST /api/projects/:uuid/workflows/preview
func (c *Client) PreviewWorkflow(ctx context.Context, projectUUID string, req WorkflowPreviewRequest) (*WorkflowPreview, error) {
	_, raw, err := c.doJSON(ctx, http.MethodPost, "/api/projects/"+projectUUID+"/workflows/preview", req)
	if err != nil {
		return nil, err
	}
	return decodeData[WorkflowPreview](raw)
}

// ExecuteWorkflow calls POST /api/projects/:uuid/workflows/execute
func (c *Client) ExecuteWorkflow(ctx context.Context, projectUUID, previewID string) (*WorkflowExecution, error) {
	_, raw, err := c.doJSON(ctx, http.MethodPost, "/api/projects/"+projectUUID+"/workflows/execute", map[string]string{
		"previewId": previewID,
	})
	if err != nil {
		return nil, err
	}
	return decodeData[WorkflowExecution](raw)
}
