package progress

// WsProgressEvent is a WebSocket progress message from the server.
type WsProgressEvent struct {
	Type        string  `json:"type"`
	RunID       string  `json:"run_id"`
	Phase       int     `json:"phase,omitempty"`
	PhaseName   string  `json:"phase_name,omitempty"`
	CurrentStep string  `json:"current_step,omitempty"`
	Status      string  `json:"status,omitempty"`
	Progress    float64 `json:"progress,omitempty"`
	Data        any     `json:"data,omitempty"`
	Message     string  `json:"message,omitempty"`
	Recoverable bool    `json:"recoverable,omitempty"`
	Timestamp   int64   `json:"timestamp"`
}

// JSONLEvent is the structure emitted in --json mode.
type JSONLEvent struct {
	Type      string  `json:"type"`
	RunID     string  `json:"run_id"`
	Phase     int     `json:"phase"`
	PhaseName string  `json:"phase_name,omitempty"`
	Progress  float64 `json:"progress,omitempty"`
	Message   string  `json:"message,omitempty"`
	Timestamp int64   `json:"timestamp"`
}
