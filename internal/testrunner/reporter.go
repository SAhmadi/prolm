package testrunner

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/fatih/color"
)

// ReportOptions controls Report formatting. It is intentionally decoupled
// from the global ui package so the reporter stays testable with bytes.Buffer.
type ReportOptions struct {
	Verbose bool
	JSON    bool
	NoColor bool
}

// Report writes a human-readable (or JSON) summary of res to w.
//
// In JSON mode the raw TestResult is encoded directly. In text mode a colour
// checkmark summary is printed plus one line per failing case. ANSI codes are
// suppressed when NoColor is set (mirrors the ui package's NO_COLOR semantics).
func Report(res *TestResult, w io.Writer, opts ReportOptions) {
	if opts.JSON {
		_ = json.NewEncoder(w).Encode(res)
		return
	}

	green := color.New(color.FgGreen, color.Bold)
	red := color.New(color.FgRed, color.Bold)
	if opts.NoColor {
		green.DisableColor()
		red.DisableColor()
	} else {
		green.EnableColor()
		red.EnableColor()
	}

	// Per-failure detail lines come first so the summary is the last thing the
	// user sees on screen (easier to read in CI logs).
	for _, c := range res.Cases {
		if c.Status == "pass" && !opts.Verbose {
			continue
		}
		if c.Status == "pass" {
			green.Fprint(w, "  ✓  ")
		} else {
			red.Fprint(w, "  ✗  ")
		}
		fmt.Fprintf(w, "%s:%s", c.Suite, c.Name)
		if c.Message != "" {
			fmt.Fprintf(w, " — %s", c.Message)
		}
		fmt.Fprintln(w)
	}

	summary := fmt.Sprintf("%d passed, %d failed", res.Passed, res.Failed+res.Errors)
	if res.Duration > 0 {
		summary += fmt.Sprintf(" (%s)", res.Duration.Round(1e6))
	}
	if res.FailedCount() == 0 {
		green.Fprint(w, "  ✓  ")
	} else {
		red.Fprint(w, "  ✗  ")
	}
	fmt.Fprintln(w, summary)
}
