package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/novelo-ai/novelo-cli/internal/config"
	"github.com/novelo-ai/novelo-cli/internal/latentcut"
	"github.com/spf13/cobra"
)

// NewChatCmd returns the chat subcommand.
func NewChatCmd() *cobra.Command {
	var (
		message   string
		threadID  string
		newThread bool
	)

	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Chat with creative-video-agent to brainstorm short drama ideas",
		Long:  "Send a message to the creative-video-agent via latentCut-server. Supports multi-turn conversation via --thread-id. Designed for skill/script invocation.",
		Example: `  latentcut chat -m "我想写一个仙侠故事"
  latentcut chat -m "主角有什么特殊能力？" --thread-id thread-xxx
  latentcut chat -m "就这个方向，开始写吧" --json
  latentcut chat -m "换个话题" --new-thread`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChat(message, threadID, newThread)
		},
	}

	cmd.Flags().StringVarP(&message, "message", "m", "", "Message to send (required)")
	cmd.Flags().StringVar(&threadID, "thread-id", "", "Thread ID for conversation context")
	cmd.Flags().BoolVar(&newThread, "new-thread", false, "Force create a new thread (ignore cached thread)")
	_ = cmd.MarkFlagRequired("message")

	return cmd
}

func runChat(message, threadID string, newThread bool) error {
	// 1. Load config and validate auth
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.EffectiveToken() == "" {
		return fmt.Errorf("not logged in. Run: latentcut login")
	}

	// 2. Resolve thread ID
	if threadID == "" && !newThread {
		threadID = cfg.LastThreadID // reuse cached thread
	}
	if newThread {
		threadID = "" // will generate a new one below
	}
	// Generate a client-side threadId if none exists, to ensure the same
	// thread is reused across calls (server generates a new one each time if empty)
	if threadID == "" {
		threadID = fmt.Sprintf("cli-thread-%d", time.Now().UnixMilli())
	}

	// 3. Setup context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	client := latentcut.NewClient(cfg.LatentCutURL, cfg.EffectiveToken())

	// 4. Load conversation history
	history := latentcut.LoadHistory(threadID)

	if !jsonOut {
		fmt.Fprintf(os.Stderr, "Thread: %s\n", displayThreadID(threadID))
	}

	// 5. Call chat stream with history
	result, err := client.ChatStream(ctx, message, threadID, history.Messages, func(chunk string) {
		if !jsonOut {
			fmt.Print(chunk) // stream text to stdout in real-time
		}
	})
	if err != nil {
		if ctx.Err() != nil {
			return nil // user cancelled
		}
		return fmt.Errorf("chat: %w", err)
	}

	// 5. Output
	// 6. Save conversation turn to local history
	if result.Text != "" {
		_ = history.AddTurn(message, result.Text)
	}

	// Resolve effective thread ID for output
	outputThreadID := result.ThreadID
	if outputThreadID == "" {
		outputThreadID = threadID
	}

	if jsonOut {
		out := map[string]any{
			"text":     result.Text,
			"threadId": outputThreadID,
		}
		data, _ := json.Marshal(out)
		fmt.Println(string(data))
	} else {
		// Ensure newline after streamed text
		if len(result.Text) > 0 && result.Text[len(result.Text)-1] != '\n' {
			fmt.Println()
		}
		fmt.Fprintf(os.Stderr, "\n[thread: %s]\n", outputThreadID)
	}

	// 6. Cache thread ID for next invocation
	// Use the server-returned threadId if available, otherwise keep the client-generated one
	effectiveThreadID := result.ThreadID
	if effectiveThreadID == "" {
		effectiveThreadID = threadID
	}
	if effectiveThreadID != "" && effectiveThreadID != cfg.LastThreadID {
		cfg.LastThreadID = effectiveThreadID
		_ = cfg.Save() // best-effort save
	}

	return nil
}

func displayThreadID(id string) string {
	if id == "" {
		return "(new)"
	}
	return id
}
