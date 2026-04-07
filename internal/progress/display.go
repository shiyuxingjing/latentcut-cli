package progress

import (
	"fmt"
	"os"

	"github.com/schollz/progressbar/v3"
)

// PhaseState holds the state of a single workflow phase.
type PhaseState struct {
	Name     string
	Status   string // pending | active | done | error
	Progress int
}

// ProgressDisplay renders real-time terminal progress for the 8 pipeline phases.
type ProgressDisplay struct {
	phases     [8]PhaseState
	currentBar *progressbar.ProgressBar
	verbose    bool
}

var phaseNames = [8]string{"LLM Pipeline", "Episode Split", "Asset Extraction", "Shot Generation", "Visual Generation", "Start Frame Composition", "Video Generation", "Video Merge"}

// NewProgressDisplay creates a new ProgressDisplay.
func NewProgressDisplay(verbose bool) *ProgressDisplay {
	d := &ProgressDisplay{verbose: verbose}
	for i := range d.phases {
		d.phases[i] = PhaseState{
			Name:   phaseNames[i],
			Status: "pending",
		}
	}
	return d
}

// HandleEvent processes a WebSocket progress event and updates the display.
func (d *ProgressDisplay) HandleEvent(event WsProgressEvent) {
	switch event.Type {
	case "phase_start":
		if event.Phase >= 0 && event.Phase < 8 {
			d.RenderPhaseStart(event.Phase, event.PhaseName)
		}
	case "progress":
		if event.Phase >= 0 && event.Phase < 8 {
			d.RenderPhaseProgress(event.Phase, int(event.Progress))
		}
	case "phase_result":
		if event.Phase >= 0 && event.Phase < 8 {
			d.phases[event.Phase].Status = "done"
			if d.currentBar != nil {
				_ = d.currentBar.Finish()
				d.currentBar = nil
			}
			fmt.Fprintf(os.Stderr, "  [%s] done\n", d.phases[event.Phase].Name)
		}
	case "complete":
		d.RenderComplete("")
	case "error":
		d.RenderError(event.Message)
	}

	if d.verbose && event.CurrentStep != "" {
		fmt.Fprintf(os.Stderr, "  [verbose] step: %s\n", event.CurrentStep)
	}
}

// RenderPhaseStart prints the phase header and resets the progress bar.
func (d *ProgressDisplay) RenderPhaseStart(phase int, name string) {
	if phase < 0 || phase >= 4 {
		return
	}
	if d.currentBar != nil {
		_ = d.currentBar.Finish()
	}
	d.phases[phase].Status = "active"
	displayName := name
	if displayName == "" {
		displayName = phaseNames[phase]
	}
	d.phases[phase].Name = displayName
	fmt.Fprintf(os.Stderr, "\nPhase %d: %s\n", phase, displayName)
	d.currentBar = progressbar.NewOptions(100,
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionSetWidth(40),
		progressbar.OptionSetDescription("  Progress"),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionShowCount(),
	)
}

// RenderPhaseProgress updates the progress bar for a phase.
func (d *ProgressDisplay) RenderPhaseProgress(phase int, pct int) {
	if phase < 0 || phase >= 4 {
		return
	}
	d.phases[phase].Progress = pct
	if d.currentBar != nil {
		current := d.currentBar.State().CurrentPercent * 100
		delta := float64(pct) - current
		if delta > 0 {
			_ = d.currentBar.Add(int(delta))
		}
	}
}

// RenderComplete prints the completion message.
func (d *ProgressDisplay) RenderComplete(outputPath string) {
	if d.currentBar != nil {
		_ = d.currentBar.Finish()
		d.currentBar = nil
	}
	fmt.Fprintln(os.Stderr, "\nPipeline complete!")
	if outputPath != "" {
		fmt.Fprintf(os.Stderr, "Output: %s\n", outputPath)
	}
}

// RenderError prints an error message.
func (d *ProgressDisplay) RenderError(msg string) {
	if d.currentBar != nil {
		_ = d.currentBar.Finish()
		d.currentBar = nil
	}
	fmt.Fprintf(os.Stderr, "\nError: %s\n", msg)
}
