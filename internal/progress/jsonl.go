package progress

import (
	"encoding/json"
	"os"
)

// JSONLWriter writes progress events as JSONL to stdout.
type JSONLWriter struct{}

// NewJSONLWriter creates a new JSONLWriter.
func NewJSONLWriter() *JSONLWriter {
	return &JSONLWriter{}
}

// HandleEvent encodes a WsProgressEvent as a JSONLEvent and writes it to stdout.
func (w *JSONLWriter) HandleEvent(event WsProgressEvent) {
	ev := JSONLEvent{
		Type:      event.Type,
		RunID:     event.RunID,
		Phase:     event.Phase,
		PhaseName: event.PhaseName,
		Progress:  event.Progress,
		Message:   event.Message,
		Timestamp: event.Timestamp,
	}
	data, err := json.Marshal(ev)
	if err != nil {
		return
	}
	os.Stdout.Write(data)
	os.Stdout.Write([]byte("\n"))
}
