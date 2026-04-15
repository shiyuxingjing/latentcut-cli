package cmd

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

// NewOpenCmd returns the open subcommand.
func NewOpenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "open <url>",
		Short: "Open a resource URL in the default browser",
		Long:  "Opens the given URL (image, video, or page) in your default browser for preview.",
		Example: `  latentcut open http://latentcut-resource.oss-cn-hangzhou.aliyuncs.com/xxx.png
  latentcut open https://shiyuxingjing.com`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOpen(args[0])
		},
	}
}

func runOpen(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}

	fmt.Printf("Opened: %s\n", url)
	return nil
}
