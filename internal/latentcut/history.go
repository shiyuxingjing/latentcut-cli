package latentcut

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ChatMessage represents a single message in conversation history.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ThreadHistory manages conversation history for a thread.
type ThreadHistory struct {
	ThreadID string        `json:"threadId"`
	Messages []ChatMessage `json:"messages"`
}

// historyDir returns the directory for storing thread histories.
func historyDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".novelo/threads"
	}
	return filepath.Join(home, ".novelo", "threads")
}

// historyPath returns the file path for a specific thread's history.
func historyPath(threadID string) string {
	return filepath.Join(historyDir(), threadID+".json")
}

// LoadHistory loads conversation history for a thread. Returns empty history if not found.
func LoadHistory(threadID string) *ThreadHistory {
	h := &ThreadHistory{ThreadID: threadID}

	data, err := os.ReadFile(historyPath(threadID))
	if err != nil {
		return h
	}

	_ = json.Unmarshal(data, h)
	return h
}

// AddTurn appends a user message and assistant response to the history and saves it.
func (h *ThreadHistory) AddTurn(userMsg, assistantMsg string) error {
	h.Messages = append(h.Messages,
		ChatMessage{Role: "user", Content: userMsg},
		ChatMessage{Role: "assistant", Content: assistantMsg},
	)

	// Keep last 12 turns (24 messages) to avoid unbounded growth
	const maxMessages = 24
	if len(h.Messages) > maxMessages {
		h.Messages = h.Messages[len(h.Messages)-maxMessages:]
	}

	return h.save()
}

func (h *ThreadHistory) save() error {
	dir := historyDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.Marshal(h)
	if err != nil {
		return err
	}

	return os.WriteFile(historyPath(h.ThreadID), data, 0600)
}
