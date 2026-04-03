package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/viper"
)

// Out is the writer for Info, Success, and Hint output.
// Defaults to os.Stdout. Tests may replace this with a *bytes.Buffer.
var Out io.Writer = os.Stdout

// Err is the writer for Warn and Error output.
// Defaults to os.Stderr. Tests may replace this with a *bytes.Buffer.
var Err io.Writer = os.Stderr

// jsonMsg is the structure emitted when --json is active.
type jsonMsg struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

// noColor reports whether colour output should be suppressed.
// Checks both the Viper flag and the raw NO_COLOR env var so that
// unit tests which bypass Cobra/Viper still behave correctly.
func noColor() bool {
	return viper.GetBool("no-color") || os.Getenv("NO_COLOR") != ""
}

// jsonMode reports whether --json structured output is active.
func jsonMode() bool {
	return viper.GetBool("json")
}

// emitJSON writes a structured JSON message to Out.
// json.Marshal cannot fail for jsonMsg (string-only fields), so the blank
// identifier makes the intentional discard explicit rather than accidental.
func emitJSON(level, msg string) {
	b, _ := json.Marshal(jsonMsg{Level: level, Message: msg})
	fmt.Fprintln(Out, string(b))
}

// printMsg is the single internal helper used by all public functions.
//
//   - JSON mode: writes a jsonMsg to Out (always stdout — single parseable stream).
//   - No-colour mode: writes plain prefix+message to w without ANSI codes.
//   - Colour mode: writes the coloured prefix via c then the plain message.
//     c.EnableColor() is called explicitly so that fatih/color's TTY
//     auto-detection does not suppress colour when w is not a real TTY
//     (e.g. when tests redirect Out to a bytes.Buffer).
func printMsg(w io.Writer, level, prefix string, c *color.Color, format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	if jsonMode() {
		emitJSON(level, msg)
		return
	}
	if noColor() {
		fmt.Fprintf(w, "%s%s\n", prefix, msg)
		return
	}
	c.EnableColor()
	c.Fprint(w, prefix)
	fmt.Fprintf(w, "%s\n", msg)
}

// Info prints an informational message (cyan) to Out.
func Info(format string, a ...any) {
	printMsg(Out, "info", "  --> ", color.New(color.FgCyan), format, a...)
}

// Success prints a success message (green, bold) to Out.
func Success(format string, a ...any) {
	printMsg(Out, "success", "  ✓  ", color.New(color.FgGreen, color.Bold), format, a...)
}

// Warn prints a warning message (yellow, bold) to Err.
func Warn(format string, a ...any) {
	printMsg(Err, "warn", "  !  ", color.New(color.FgYellow, color.Bold), format, a...)
}

// Error prints an error message (red, bold) to Err.
func Error(format string, a ...any) {
	printMsg(Err, "error", "  ✗  ", color.New(color.FgRed, color.Bold), format, a...)
}

// Hint prints a muted hint or suggestion (faint) to Out.
func Hint(format string, a ...any) {
	printMsg(Out, "hint", "  ?  ", color.New(color.Faint), format, a...)
}
