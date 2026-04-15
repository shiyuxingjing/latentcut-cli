package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

// NewInstallCmd returns the install subcommand.
func NewInstallCmd() *cobra.Command {
	var installDir string

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install latentcut to system PATH",
		Long:  "Copy the current binary to /usr/local/bin (or specified directory) so it can be used from anywhere.",
		Example: `  latentcut install
  latentcut install --dir /usr/local/bin
  sudo latentcut install`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstall(installDir)
		},
	}

	defaultDir := "/usr/local/bin"
	if runtime.GOOS == "windows" {
		defaultDir = filepath.Join(os.Getenv("USERPROFILE"), "bin")
	}
	cmd.Flags().StringVar(&installDir, "dir", defaultDir, "Installation directory")

	return cmd
}

func runInstall(installDir string) error {
	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("resolve symlinks: %w", err)
	}

	destPath := filepath.Join(installDir, "latentcut")
	if runtime.GOOS == "windows" {
		destPath += ".exe"
	}

	// Check if already installed at the destination
	if execPath == destPath {
		fmt.Fprintln(os.Stderr, "Already installed at", destPath)
		return nil
	}

	// Try to create the directory
	if err := os.MkdirAll(installDir, 0755); err != nil {
		// May need sudo
		fmt.Fprintf(os.Stderr, "Cannot create %s (try: sudo latentcut install)\n", installDir)
		return fmt.Errorf("create install dir: %w", err)
	}

	// Copy the binary
	src, err := os.Open(execPath)
	if err != nil {
		return fmt.Errorf("open source binary: %w", err)
	}
	defer src.Close()

	dst, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		// May need sudo
		fmt.Fprintf(os.Stderr, "Cannot write to %s (try: sudo latentcut install)\n", destPath)
		return fmt.Errorf("create destination: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copy binary: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Installed to %s\n", destPath)

	// Verify it's in PATH
	found, _ := exec.LookPath("latentcut")
	if found != "" {
		fmt.Fprintf(os.Stderr, "Verified: latentcut is in PATH (%s)\n", found)
	} else {
		fmt.Fprintf(os.Stderr, "Warning: %s may not be in your PATH.\n", installDir)
		fmt.Fprintf(os.Stderr, "Add to PATH: export PATH=\"%s:$PATH\"\n", installDir)
	}

	return nil
}
