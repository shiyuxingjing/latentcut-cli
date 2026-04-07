package types

// PipelineRunRequest is the request body for POST /pipeline/run.
type PipelineRunRequest struct {
	InputText string `json:"input_text"`
	Style     string `json:"style,omitempty"`
}

// PipelineRunResponse is the response from POST /pipeline/run.
type PipelineRunResponse struct {
	RunID  string `json:"run_id"`
	Status string `json:"status"`
}

// ShotResult holds the video output for a single shot.
type ShotResult struct {
	ShotNumber int    `json:"shot_number"`
	VideoURL   string `json:"video_url"`
	LocalPath  string `json:"local_path,omitempty"`
	ExpiresAt  string `json:"expires_at,omitempty"` // ISO8601 expiry for CDN URLs
}

// RunFullData is the complete event data from /pipeline/run-full.
type RunFullData struct {
	Shots       []ShotResult `json:"shots"`
	ConcatVideo string       `json:"concat_video,omitempty"`
	PipelineID  string       `json:"pipeline_id,omitempty"`
}
