package checker

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/fatih/color"
)

// ReportOptions controls Report formatting. It intentionally mirrors
// testrunner.ReportOptions so the two commands feel alike.
type ReportOptions struct {
	Strict  bool
	JSON    bool
	NoColor bool
}

// Report writes a human-readable (or JSON) summary of res to w. Returns
// whether the run should be considered failing: any error diagnostic, or any
// warning when Strict is set.
func Report(res *CheckResult, w io.Writer, opts ReportOptions) bool {
	if opts.JSON {
		_ = json.NewEncoder(w).Encode(res)
		return failing(res, opts.Strict)
	}

	red := color.New(color.FgRed, color.Bold)
	yellow := color.New(color.FgYellow, color.Bold)
	green := color.New(color.FgGreen, color.Bold)
	if opts.NoColor {
		red.DisableColor()
		yellow.DisableColor()
		green.DisableColor()
	} else {
		red.EnableColor()
		yellow.EnableColor()
		green.EnableColor()
	}

	for _, d := range res.Diagnostics {
		switch d.Severity {
		case SeverityError:
			red.Fprint(w, "  ✗  ")
		default:
			yellow.Fprint(w, "  !  ")
		}
		if d.File != "" {
			if d.Line > 0 {
				fmt.Fprintf(w, "%s:%d: ", d.File, d.Line)
			} else {
				fmt.Fprintf(w, "%s: ", d.File)
			}
		}
		fmt.Fprintln(w, d.Message)
	}

	warns := len(res.Warnings())
	errs := len(res.Errors())
	summary := fmt.Sprintf("%d error(s), %d warning(s)", errs, warns)
	if failing(res, opts.Strict) {
		red.Fprint(w, "  ✗  ")
	} else {
		green.Fprint(w, "  ✓  ")
	}
	fmt.Fprintln(w, summary)
	return failing(res, opts.Strict)
}

func failing(res *CheckResult, strict bool) bool {
	if len(res.Errors()) > 0 {
		return true
	}
	if strict && len(res.Warnings()) > 0 {
		return true
	}
	return false
}
