package latentcut

import "strings"

// APIResponse is the standard latentCut-server response wrapper.
type APIResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// LoginRequest is the body for POST /api/user/login.
type LoginRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

// LoginData is the data field from a successful login.
type LoginData struct {
	Token string `json:"token"`
	User  struct {
		ID    int    `json:"id"`
		Email string `json:"email"`
		UUID  string `json:"uuid"`
	} `json:"user"`
}

// CreateAPIKeyRequest is the body for POST /api/user/api-keys.
type CreateAPIKeyRequest struct {
	Name string `json:"name"`
}

// CreateAPIKeyData is the data field from a successful API key creation.
type CreateAPIKeyData struct {
	APIKey string `json:"api_key"`
	Name   string `json:"name"`
	ID     int    `json:"id"`
}

// CreateProjectRequest is the body for POST /api/projects.
type CreateProjectRequest struct {
	Title        string `json:"title"`
	NovelContent string `json:"novel_content"`
	Style        string `json:"style,omitempty"`
	StyleTitle   string `json:"style_title,omitempty"`
	ProjectMode  string `json:"project_mode"`
}

// CreateProjectData is the data field from project creation.
type CreateProjectData struct {
	TaskUUID      string `json:"task_uuid"`
	TaskType      string `json:"task_type"`
	TaskStatus    string `json:"task_status"`
	ProjectUUID   string `json:"project_uuid"`
	ProjectID     any    `json:"project_id"`
	Message       string `json:"message"`
	EstimatedTime any    `json:"estimated_time"`
}

// CanvasData holds the full project canvas structure.
// The server returns {cards: {uuid: cardData, ...}, connections: [...]}
type CanvasData struct {
	Cards       map[string]CardData `json:"cards"`
	Connections []any               `json:"connections"`
}

// CardData is a generic card in the canvas. Fields vary by card type.
type CardData struct {
	Raw           CardRaw `json:"_raw"`
	Title         string  `json:"title,omitempty"`
	Name          string  `json:"name,omitempty"`
	Number        int     `json:"number,omitempty"`
	Status        string  `json:"status,omitempty"`
	VideoStatus   string  `json:"videoStatus,omitempty"`
	VideoURL      string  `json:"videoUrl,omitempty"`
	ImageURL      string  `json:"imageUrl,omitempty"`
	ImageStatus   string  `json:"imageStatus,omitempty"`
	Image         string  `json:"image,omitempty"`
	CharacterName string  `json:"characterName,omitempty"`
	CharacterUUID string  `json:"characterUuid,omitempty"`
	Description   string  `json:"description,omitempty"`
	Location      string  `json:"location,omitempty"`
	ShotCount     int     `json:"shotCount,omitempty"`
	EpisodeID     string  `json:"episodeId,omitempty"`
}

// CardRaw contains the raw DB fields from the server.
type CardRaw struct {
	ID             int    `json:"id"`
	EpisodeUUID    string `json:"episode_uuid,omitempty"`
	ShotUUID       string `json:"shot_uuid,omitempty"`
	CharacterUUID  string `json:"character_uuid,omitempty"`
	LocationUUID   string `json:"locationtime_uuid,omitempty"`
	CharacterName  string `json:"character_name,omitempty"`
	Description    string `json:"description,omitempty"`
	Location       string `json:"location,omitempty"`
	TimeOfDay      string `json:"time_of_day,omitempty"`
	EpisodeID      int    `json:"episode_id,omitempty"`
	EpisodeNumber  int    `json:"episode_number,omitempty"`
	Title          string `json:"title,omitempty"`
	Name           string `json:"name,omitempty"`
	VideoStatus    string `json:"video_status,omitempty"`
	VideoURL       string `json:"video_url,omitempty"`
	ImageStatus    string `json:"image_status,omitempty"`
	ImageURL       string `json:"image_url,omitempty"`
	TTSStatus      string `json:"tts_status,omitempty"`
	SortOrder      int    `json:"sort_order,omitempty"`
}

// ParsedCanvas is a convenience struct extracted from CanvasData.
type ParsedCanvas struct {
	Episodes       []ParsedEpisode
	Shots          []ParsedShot
	CharacterUUIDs []string
	LocationUUIDs  []string
	Characters     int
	Locations      int
}

// ParsedEpisode is an episode extracted from canvas cards.
type ParsedEpisode struct {
	UUID        string
	Title       string
	Number      int
	VideoStatus string
	VideoURL    string
	Shots       []ParsedShot
}

// ParsedShot is a shot extracted from canvas cards.
type ParsedShot struct {
	UUID        string
	EpisodeID   int
	VideoStatus string
	VideoURL    string
	SortOrder   int
}

// ParseCanvas extracts episodes, shots, characters, and locations from CanvasData.
func (c *CanvasData) ParseCanvas() ParsedCanvas {
	var result ParsedCanvas
	episodeMap := make(map[string]*ParsedEpisode)
	seenChars := make(map[string]bool)
	seenLocs := make(map[string]bool)

	// First pass: collect all cards by type
	// Card key prefixes: episode-, shot-, char-, loc-, kf-, dlg-, novel-
	for key, card := range c.Cards {
		switch {
		case strings.HasPrefix(key, "episode-"):
			ep := ParsedEpisode{
				UUID:        key,
				Title:       card.Title,
				Number:      card.Number,
				VideoStatus: card.VideoStatus,
				VideoURL:    card.VideoURL,
			}
			if ep.Title == "" {
				ep.Title = card.Raw.Title
			}
			if ep.VideoStatus == "" {
				ep.VideoStatus = card.Raw.VideoStatus
			}
			if ep.VideoURL == "" {
				ep.VideoURL = card.Raw.VideoURL
			}
			result.Episodes = append(result.Episodes, ep)
			episodeMap[key] = &result.Episodes[len(result.Episodes)-1]
		case strings.HasPrefix(key, "shot-"):
			shot := ParsedShot{
				UUID:        key,
				EpisodeID:   card.Raw.EpisodeID,
				VideoStatus: card.Raw.VideoStatus,
				VideoURL:    card.Raw.VideoURL,
				SortOrder:   card.Raw.SortOrder,
			}
			result.Shots = append(result.Shots, shot)
		case strings.HasPrefix(key, "char-"):
			uuid := card.Raw.CharacterUUID
			if uuid != "" && !seenChars[uuid] {
				seenChars[uuid] = true
				result.CharacterUUIDs = append(result.CharacterUUIDs, uuid)
				result.Characters++
			}
		case strings.HasPrefix(key, "loc-"):
			uuid := card.Raw.LocationUUID
			if uuid != "" && !seenLocs[uuid] {
				seenLocs[uuid] = true
				result.LocationUUIDs = append(result.LocationUUIDs, uuid)
				result.Locations++
			}
		}
	}

	// Second pass: assign shots to episodes
	for i := range result.Shots {
		shot := &result.Shots[i]
		for j := range result.Episodes {
			ep := &result.Episodes[j]
			if ep.UUID != "" && shot.EpisodeID == result.Episodes[j].Number {
				// Match by episode_id (DB id) - need to match by raw ID
			}
		}
	}

	// Match shots to episodes via episode raw ID
	epByDBID := make(map[int]*ParsedEpisode)
	for key, card := range c.Cards {
		if len(key) > 8 && key[:8] == "episode-" {
			if ep, ok := episodeMap[key]; ok {
				epByDBID[card.Raw.ID] = ep
			}
		}
	}
	for _, shot := range result.Shots {
		if ep, ok := epByDBID[shot.EpisodeID]; ok {
			ep.Shots = append(ep.Shots, shot)
		}
	}

	return result
}

// ShotBatchPreview is the response from shot-batch/preview.
type ShotBatchPreview struct {
	TotalCredits     int             `json:"totalCredits"`
	Credits          int             `json:"credits"`
	CostConfig       map[string]int  `json:"costConfig"`
	Candidates       []ShotCandidate `json:"candidates"`
	Options          []any           `json:"options"`
	DefaultOptionKey *string         `json:"defaultOptionKey"`
}

// ShotCandidate is a shot eligible for batch generation.
type ShotCandidate struct {
	ShotUUID string `json:"shotUuid"`
	Needs    []string `json:"needs"`
}

// ShotBatchRequest is the body for POST shot-batch.
type ShotBatchRequest struct {
	Mode  string `json:"mode"`
	Count int    `json:"count,omitempty"`
}

// ShotBatchData is the response from shot-batch creation.
type ShotBatchData struct {
	BatchID       string   `json:"batchId"`
	AcceptedCount int      `json:"acceptedCount"`
	EstimatedCost int      `json:"estimatedCost"`
	ShotUUIDs     []string `json:"shotUuids"`
}

// GenerateEpisodeVideoRequest is the body for episode-video generation.
type GenerateEpisodeVideoRequest struct {
	EpisodeUUID string `json:"episodeUuid"`
}

// PendingTask represents a pending generation task.
type PendingTask struct {
	TaskUUID   string `json:"task_uuid"`
	TaskType   string `json:"task_type"`
	TaskStatus string `json:"task_status"`
	TaskParams any    `json:"task_params"`
	TaskResult any    `json:"task_result"`
	Error      string `json:"error_message"`
}

// TaskStatus represents a task status response.
type TaskStatus struct {
	TaskUUID   string `json:"task_uuid"`
	TaskType   string `json:"task_type"`
	TaskStatus string `json:"task_status"`
	TaskResult any    `json:"task_result"`
	Error      string `json:"error_message"`
}

// SSE event types
const (
	EventDramaProgress = "drama_progress"
	EventDramaDone     = "drama_done"
	EventDramaFailed   = "drama_failed"
	EventShotVideoDone = "shot_video_done"
	EventEpisodeVideoDone = "episode_video_done"
)

// DramaProgressEvent is the SSE data for drama_progress events.
type DramaProgressEvent struct {
	TaskID      string  `json:"taskId"`
	Stage       string  `json:"stage"`
	Progress    float64 `json:"progress"`
	CurrentStep string  `json:"currentStep"`
}

// DramaDoneEvent is the SSE data for drama_done events.
type DramaDoneEvent struct {
	TaskID      string `json:"taskId"`
	ProjectUUID string `json:"projectUuid"`
}

// DramaFailedEvent is the SSE data for drama_failed events.
type DramaFailedEvent struct {
	TaskID        string  `json:"taskId"`
	Error         string  `json:"error"`
	RetryEligible bool    `json:"retryEligible"`
	Reason        string  `json:"reason"`
	Stage         string  `json:"stage"`
	CurrentStep   string  `json:"currentStep"`
	Progress      float64 `json:"progress"`
}

// ResourceDoneEvent is the SSE data for resource generation done events.
type ResourceDoneEvent struct {
	ProjectUUID string `json:"projectUuid"`
	EntityUUID  string `json:"entityUuid"`
	EntityType  string `json:"entityType"`
	FileURL     string `json:"fileUrl"`
}

// ProjectProgress is the flat progress snapshot returned by
// GET /api/projects/:uuid/progress. It is designed to be the single
// source of truth that agents/CLIs need to compute and display
// pipeline progress in one round trip.
type ProjectProgress struct {
	ProjectUUID     string                  `json:"project_uuid"`
	Title           string                  `json:"title"`
	Status          string                  `json:"status"`
	OverallProgress float64                 `json:"overall_progress"`
	CurrentPhase    string                  `json:"current_phase"`
	PhaseStep       string                  `json:"phase_step"`
	PhaseProgress   *float64                `json:"phase_progress"`
	Episodes        ProjectProgressEpisodes `json:"episodes"`
	Shots           ProjectProgressShots    `json:"shots"`
	AssetParse      *ProjectPhaseState      `json:"asset_parse,omitempty"`
	ShotParse       *ProjectPhaseState      `json:"shot_parse,omitempty"`
	PendingTasks    PendingTasksSummary     `json:"pending_tasks"`
	UpdatedAt       string                  `json:"updated_at"`
}

// ProjectProgressEpisodes is the episode roll-up inside a progress snapshot.
type ProjectProgressEpisodes struct {
	Total  int  `json:"total"`
	Parsed *int `json:"parsed"`
}

// ProjectProgressShots is the shot roll-up inside a progress snapshot.
type ProjectProgressShots struct {
	Total int `json:"total"`
}

// ProjectPhaseState mirrors metadata.asset_parse / metadata.shot_parse.
type ProjectPhaseState struct {
	Status            string   `json:"status"`
	Progress          *float64 `json:"progress"`
	CurrentStep       string   `json:"current_step"`
	CompletedEpisodes *int     `json:"completed_episodes,omitempty"`
	TotalEpisodes     *int     `json:"total_episodes,omitempty"`
	RetryEligible     *bool    `json:"retry_eligible,omitempty"`
	ErrorMessage      string   `json:"error_message,omitempty"`
	UpdatedAt         string   `json:"updated_at,omitempty"`
}

// PendingTasksSummary is the aggregated view of pending generation tasks.
type PendingTasksSummary struct {
	Total    int            `json:"total"`
	ByType   map[string]int `json:"by_type"`
	ByStatus map[string]int `json:"by_status"`
}
