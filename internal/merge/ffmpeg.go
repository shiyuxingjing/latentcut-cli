package merge

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/novelo-ai/novelo-cli/internal/types"
)

// DetectFFmpeg finds ffmpeg in PATH and returns its path.
func DetectFFmpeg() (string, error) {
	path, err := exec.LookPath("ffmpeg")
	if err != nil {
		return "", fmt.Errorf("ffmpeg not found in PATH\n\nInstall ffmpeg:\n  macOS:  brew install ffmpeg\n  Ubuntu: apt install ffmpeg\n  Windows: https://ffmpeg.org/download.html")
	}
	return path, nil
}

// DownloadVideos downloads shot videos in parallel into dir.
// Warns about expired URLs and skips them.
func DownloadVideos(ctx context.Context, shots []types.ShotResult, dir string) ([]string, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create shots dir: %w", err)
	}

	type result struct {
		index int
		path  string
		err   error
	}

	results := make([]result, len(shots))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4) // max 4 parallel downloads

	for i, shot := range shots {
		// Check expiry
		if shot.ExpiresAt != "" {
			exp, err := time.Parse(time.RFC3339, shot.ExpiresAt)
			if err == nil && time.Now().After(exp) {
				fmt.Fprintf(os.Stderr, "  warning: shot %d URL has expired, skipping\n", shot.ShotNumber)
				results[i] = result{index: i, err: fmt.Errorf("expired")}
				continue
			}
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, s types.ShotResult) {
			defer wg.Done()
			defer func() { <-sem }()

			filename := filepath.Join(dir, fmt.Sprintf("shot_%04d.mp4", s.ShotNumber))
			err := downloadFile(ctx, s.VideoURL, filename)
			results[idx] = result{index: idx, path: filename, err: err}
		}(i, shot)
	}
	wg.Wait()

	var paths []string
	var errs []error
	for _, r := range results {
		if r.err != nil {
			errs = append(errs, r.err)
			continue
		}
		paths = append(paths, r.path)
	}

	if len(errs) > 0 && len(paths) == 0 {
		return nil, fmt.Errorf("all downloads failed")
	}

	return paths, nil
}

func downloadFile(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download %s: status %d", url, resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

// MergeVideos uses ffmpeg concat demuxer to join inputFiles into outputPath.
func MergeVideos(ctx context.Context, ffmpegPath string, inputFiles []string, outputPath string) error {
	if len(inputFiles) == 0 {
		return fmt.Errorf("no input files to merge")
	}

	// Write filelist.txt
	listPath := filepath.Join(filepath.Dir(outputPath), "filelist.txt")
	f, err := os.Create(listPath)
	if err != nil {
		return fmt.Errorf("create filelist: %w", err)
	}
	for _, p := range inputFiles {
		abs, _ := filepath.Abs(p)
		fmt.Fprintf(f, "file '%s'\n", abs)
	}
	f.Close()
	defer os.Remove(listPath)

	cmd := exec.CommandContext(ctx, ffmpegPath,
		"-f", "concat",
		"-safe", "0",
		"-i", listPath,
		"-c", "copy",
		"-y",
		outputPath,
	)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg merge failed: %w", err)
	}
	return nil
}
