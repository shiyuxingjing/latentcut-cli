package latentcut

// AssistantSummary is the response from GET /api/projects/:uuid/assistant-summary
type AssistantSummary struct {
	ProjectUUID         string               `json:"projectUuid"`
	Title               string               `json:"title"`
	Status              string               `json:"status"`
	ModeRecommendations []ModeRecommendation `json:"modeRecommendations"`
	Structure           SummaryStructure     `json:"structure"`
	AssetsReady         AssetsReady          `json:"assetsReady"`
	Progress            SummaryProgress      `json:"progress"`
	NextSuggestedTarget *SuggestedTarget     `json:"nextSuggestedTarget,omitempty"`
}

// ModeRecommendation is a recommended next action mode.
type ModeRecommendation struct {
	Mode   string `json:"mode"`
	Label  string `json:"label"`
	Reason string `json:"reason"`
}

// SummaryStructure holds project structure counts.
type SummaryStructure struct {
	Episodes   int `json:"episodes"`
	Shots      int `json:"shots"`
	Characters int `json:"characters"`
	Locations  int `json:"locations"`
}

// AssetsReady holds counts of ready assets.
type AssetsReady struct {
	CharacterImages int `json:"characterImages"`
	CharacterVoices int `json:"characterVoices"`
	LocationImages  int `json:"locationImages"`
}

// SummaryProgress holds overall progress info.
type SummaryProgress struct {
	Overall      int    `json:"overall"`
	Phase        string `json:"phase"`
	PendingTasks int    `json:"pendingTasks"`
}

// SuggestedTarget is the next recommended shot to work on.
type SuggestedTarget struct {
	EpisodeNumber int    `json:"episodeNumber"`
	ShotNumber    int    `json:"shotNumber"`
	ShotUUID      string `json:"shotUuid"`
}

// Gallery is the response from GET /api/projects/:uuid/gallery
type Gallery struct {
	Characters []GalleryCharacter `json:"characters"`
	Locations  []GalleryLocation  `json:"locations"`
	Shots      []GalleryShot      `json:"shots"`
	Resources  []GalleryResource  `json:"resources"`
}

// GalleryCharacter is a character in the gallery view.
type GalleryCharacter struct {
	UUID              string `json:"uuid"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	ImageStatus       string `json:"imageStatus"`
	ImageURL          string `json:"imageUrl"`
	ImageResourceUUID string `json:"imageResourceUuid,omitempty"`
	VoiceStatus       string `json:"voiceStatus"`
	VoiceURL          string `json:"voiceUrl,omitempty"`
	VoiceResourceUUID string `json:"voiceResourceUuid,omitempty"`
}

// GalleryLocation is a location in the gallery view.
type GalleryLocation struct {
	UUID              string `json:"uuid"`
	Name              string `json:"name"`
	ImageStatus       string `json:"imageStatus"`
	ImageURL          string `json:"imageUrl"`
	ImageResourceUUID string `json:"imageResourceUuid,omitempty"`
}

// GalleryShot is a shot in the gallery view.
type GalleryShot struct {
	UUID              string `json:"uuid"`
	EpisodeNumber     int    `json:"episodeNumber"`
	EpisodeTitle      string `json:"episodeTitle"`
	ShotNumber        int    `json:"shotNumber"`
	VideoStatus       string `json:"videoStatus"`
	VideoURL          string `json:"videoUrl"`
	VideoResourceUUID string `json:"videoResourceUuid,omitempty"`
}

// GalleryResource is a flattened latest resource record for direct lookup/open.
type GalleryResource struct {
	ResourceUUID string `json:"resourceUuid"`
	TargetUUID   string `json:"targetUuid"`
	TargetType   string `json:"targetType"`
	RelationType string `json:"relationType"`
	Label        string `json:"label"`
	Status       string `json:"status"`
	FileURL      string `json:"fileUrl"`
}

// WorkflowPreviewRequest is the body for POST /workflows/preview
type WorkflowPreviewRequest struct {
	Mode   string                 `json:"mode"`
	Target *WorkflowPreviewTarget `json:"target,omitempty"`
}

// WorkflowPreviewTarget specifies the target for a workflow preview.
type WorkflowPreviewTarget struct {
	ShotUUID    string `json:"shotUuid,omitempty"`
	EpisodeUUID string `json:"episodeUuid,omitempty"`
}

// WorkflowPreview is the response from POST /workflows/preview
type WorkflowPreview struct {
	PreviewID           string               `json:"previewId"`
	Mode                string               `json:"mode"`
	Scope               WorkflowScope        `json:"scope"`
	Outputs             []WorkflowOutput     `json:"outputs"`
	MissingDependencies []WorkflowDependency `json:"missingDependencies"`
	Cost                WorkflowCost         `json:"cost"`
	Timing              WorkflowTiming       `json:"timing"`
	Explain             WorkflowExplanation  `json:"explain"`
	CanExecute          bool                 `json:"canExecute"`
	BlockReason         *string              `json:"blockReason"`
	ExpiresAt           string               `json:"expiresAt"`
}

// WorkflowScope defines what's included in the workflow.
type WorkflowScope struct {
	ShotUUIDs    []string `json:"shotUuids"`
	EpisodeUUIDs []string `json:"episodeUuids"`
}

// WorkflowOutput describes a type and count of outputs.
type WorkflowOutput struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

// WorkflowDependency describes a missing dependency.
type WorkflowDependency struct {
	Type     string `json:"type"`
	UUID     string `json:"uuid,omitempty"`
	Name     string `json:"name,omitempty"`
	ShotUUID string `json:"shotUuid,omitempty"`
}

// WorkflowCost holds estimated cost information.
type WorkflowCost struct {
	EstimatedCredits int    `json:"estimatedCredits"`
	Currency         string `json:"currency"`
	Balance          any    `json:"balance"`
}

// WorkflowTiming holds estimated time information.
type WorkflowTiming struct {
	EstimatedSeconds int `json:"estimatedSeconds"`
}

// WorkflowExplanation provides human-readable explanations.
type WorkflowExplanation struct {
	Summary         string `json:"summary"`
	UserSafeSummary string `json:"userSafeSummary"`
}

// WorkflowExecution is the response from POST /workflows/execute
type WorkflowExecution struct {
	WorkflowRunID    string        `json:"workflowRunId"`
	Mode             string        `json:"mode"`
	Accepted         bool          `json:"accepted"`
	ProjectUUID      string        `json:"projectUuid"`
	Scope            WorkflowScope `json:"scope"`
	CreatedTasks     int           `json:"createdTasks"`
	EstimatedCredits int           `json:"estimatedCredits"`
}
