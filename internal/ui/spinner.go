package ui

import (
	"fmt"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
)

var (
	spinMu  sync.Mutex
	spinner *progressbar.ProgressBar
)

// StartSpinner displays an indeterminate progress spinner with msg.
//
// Behaviour by mode:
//   - JSON: emits {"level":"info","message":"<msg>"} to Out; no animation.
//   - No-colour: prints a plain "  --> <msg>" line; no animation.
//   - Default: renders a Braille spinner that clears itself on StopSpinner.
//
// If a spinner is already running it is stopped before the new one starts.
// Safe to call from multiple goroutines.
func StartSpinner(msg string) {
	spinMu.Lock()
	defer spinMu.Unlock()

	stopSpinnerLocked()

	if jsonMode() {
		emitJSON("info", msg)
		return
	}
	if noColor() {
		fmt.Fprintf(Out, "  --> %s\n", msg)
		return
	}

	spinner = progressbar.NewOptions(-1,
		progressbar.OptionSetDescription(msg),
		progressbar.OptionSetWriter(Out),
		progressbar.OptionSpinnerType(14), // Braille: ⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏
		progressbar.OptionSetWidth(10),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionSetElapsedTime(false),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionThrottle(100*time.Millisecond),
	)
}

// StopSpinner stops and clears the active spinner, if any.
// Idempotent: safe to call when no spinner is running.
// Safe to call from multiple goroutines.
func StopSpinner() {
	spinMu.Lock()
	defer spinMu.Unlock()
	stopSpinnerLocked()
}

// stopSpinnerLocked stops the spinner. Caller must hold spinMu.
func stopSpinnerLocked() {
	if spinner == nil {
		return
	}
	_ = spinner.Finish()
	spinner = nil
}

// NewDownloadBar returns a progress bar configured for byte-counted downloads.
//
// Behaviour by mode:
//   - JSON: emits an info message to Out and returns a silent bar.
//   - No-colour: prints a plain line to Out and returns a silent bar.
//   - Default: returns a visible bar with byte counts and ETA.
//
// The caller should wrap the HTTP response body:
//
//	bar := ui.NewDownloadBar(resp.ContentLength, "Downloading clpfd")
//	_, err = io.Copy(dest, progressbar.NewReader(resp.Body, bar))
func NewDownloadBar(totalBytes int64, description string) *progressbar.ProgressBar {
	if jsonMode() {
		emitJSON("info", description)
		return progressbar.DefaultSilent(totalBytes)
	}
	if noColor() {
		fmt.Fprintf(Out, "  --> %s\n", description)
		return progressbar.DefaultSilent(totalBytes)
	}
	return progressbar.NewOptions64(totalBytes,
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWriter(Out),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(40),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionSetElapsedTime(true),
		progressbar.OptionSetPredictTime(true),
		progressbar.OptionClearOnFinish(),
	)
}
