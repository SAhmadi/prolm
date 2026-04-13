package ui

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureOut redirects Out to a buffer for the duration of the test.
func captureOut(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	orig := Out
	Out = buf
	t.Cleanup(func() { Out = orig })
	return buf
}

// captureErr redirects Err to a buffer for the duration of the test.
func captureErr(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	orig := Err
	Err = buf
	t.Cleanup(func() { Err = orig })
	return buf
}

// resetViper clears the no-color and json Viper flags after the test.
// Do NOT call t.Parallel() in tests that use this helper — Viper is global state.
func resetViper(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		viper.Set("no-color", false)
		viper.Set("json", false)
	})
}

// --- Group 1: Output routing ---

func TestInfo_WritesToOut(t *testing.T) {
	resetViper(t)
	viper.Set("no-color", true) // avoid ANSI in assertion
	out := captureOut(t)
	_ = captureErr(t) // ensure nothing leaks to Err

	Info("hello %s", "world")

	assert.Contains(t, out.String(), "hello world")
}

func TestSuccess_WritesToOut(t *testing.T) {
	resetViper(t)
	viper.Set("no-color", true)
	out := captureOut(t)
	_ = captureErr(t)

	Success("done")

	assert.Contains(t, out.String(), "done")
}

func TestHint_WritesToOut(t *testing.T) {
	resetViper(t)
	viper.Set("no-color", true)
	out := captureOut(t)
	_ = captureErr(t)

	Hint("try `prolm install`")

	assert.Contains(t, out.String(), "try `prolm install`")
}

func TestWarn_WritesToErr(t *testing.T) {
	resetViper(t)
	viper.Set("no-color", true)
	out := captureOut(t)
	errBuf := captureErr(t)

	Warn("uh oh")

	assert.Contains(t, errBuf.String(), "uh oh")
	assert.Empty(t, out.String(), "Warn must not write to Out")
}

func TestError_WritesToErr(t *testing.T) {
	resetViper(t)
	viper.Set("no-color", true)
	out := captureOut(t)
	errBuf := captureErr(t)

	Error("something failed")

	assert.Contains(t, errBuf.String(), "something failed")
	assert.Empty(t, out.String(), "Error must not write to Out")
}

// --- Group 2: No-colour mode ---

func TestNoColorFlag_SuppressesANSI(t *testing.T) {
	resetViper(t)
	viper.Set("no-color", true)
	out := captureOut(t)

	Info("msg")

	assert.NotContains(t, out.String(), "\x1b[", "ANSI escape must be absent when --no-color is set")
}

func TestNO_COLOR_EnvVar_SuppressesANSI(t *testing.T) {
	// Tests the env-var bypass path in noColor() that operates without Viper.
	t.Setenv("NO_COLOR", "1")
	resetViper(t)
	out := captureOut(t)

	Info("msg")

	assert.NotContains(t, out.String(), "\x1b[", "ANSI escape must be absent when NO_COLOR env is set")
}

func TestNoColor_OutputContainsMessage(t *testing.T) {
	resetViper(t)
	viper.Set("no-color", true)
	out := captureOut(t)

	Info("expected content")

	assert.Contains(t, out.String(), "expected content")
}

// --- Group 3: JSON mode ---

func TestJSONMode_Info(t *testing.T) {
	resetViper(t)
	viper.Set("json", true)
	out := captureOut(t)

	Info("installing clpfd")

	var msg jsonMsg
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(out.String())), &msg))
	assert.Equal(t, "info", msg.Level)
	assert.Equal(t, "installing clpfd", msg.Message)
}

func TestJSONMode_Success(t *testing.T) {
	resetViper(t)
	viper.Set("json", true)
	out := captureOut(t)

	Success("installed 3 packages")

	var msg jsonMsg
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(out.String())), &msg))
	assert.Equal(t, "success", msg.Level)
	assert.Equal(t, "installed 3 packages", msg.Message)
}

func TestJSONMode_Warn_WritesToOut(t *testing.T) {
	// In JSON mode all output goes to Out (single stream for tooling).
	resetViper(t)
	viper.Set("json", true)
	out := captureOut(t)
	errBuf := captureErr(t)

	Warn("yanked version skipped")

	var msg jsonMsg
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(out.String())), &msg))
	assert.Equal(t, "warn", msg.Level)
	assert.Equal(t, "yanked version skipped", msg.Message)
	assert.Empty(t, errBuf.String(), "JSON Warn must not write to Err")
}

func TestJSONMode_Error_WritesToOut(t *testing.T) {
	resetViper(t)
	viper.Set("json", true)
	out := captureOut(t)
	errBuf := captureErr(t)

	Error("checksum mismatch")

	var msg jsonMsg
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(out.String())), &msg))
	assert.Equal(t, "error", msg.Level)
	assert.Equal(t, "checksum mismatch", msg.Message)
	assert.Empty(t, errBuf.String(), "JSON Error must not write to Err")
}

func TestJSONMode_Hint(t *testing.T) {
	resetViper(t)
	viper.Set("json", true)
	out := captureOut(t)

	Hint("run `prolm install` first")

	var msg jsonMsg
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(out.String())), &msg))
	assert.Equal(t, "hint", msg.Level)
}

func TestJSONMode_ValidJSON_SpecialChars(t *testing.T) {
	resetViper(t)
	viper.Set("json", true)
	out := captureOut(t)

	// Special characters that must be properly escaped in JSON.
	Info(`path "foo/bar" has \n newline and <tag>`)

	assert.True(t, json.Valid([]byte(strings.TrimSpace(out.String()))),
		"output must be valid JSON even with special characters")
}

// --- Group 4: Format strings ---

func TestInfo_FormatString(t *testing.T) {
	resetViper(t)
	viper.Set("no-color", true)
	out := captureOut(t)

	Info("count: %d", 42)

	assert.Contains(t, out.String(), "count: 42")
}

func TestError_FormatString(t *testing.T) {
	resetViper(t)
	viper.Set("no-color", true)
	_ = captureOut(t)
	errBuf := captureErr(t)

	Error("pack %q not found in %s", "clpfd", "registry")

	assert.Contains(t, errBuf.String(), `"clpfd"`)
	assert.Contains(t, errBuf.String(), "registry")
}

// --- Group 5: Table-driven smoke test ---

func TestAllPrinterFunctions_WriteMessage(t *testing.T) {
	// Fatal is omitted because it calls os.Exit(1).
	// Its Error path is covered by TestError_WritesToErr.
	tests := []struct {
		name     string
		fn       func(string, ...any)
		captureW func(*testing.T) *bytes.Buffer // which buffer to capture
		otherW   func(*testing.T) *bytes.Buffer // must remain empty
	}{
		{"Info", Info, captureOut, captureErr},
		{"Success", Success, captureOut, captureErr},
		{"Hint", Hint, captureOut, captureErr},
		{"Warn", Warn, captureErr, captureOut},
		{"Error", Error, captureErr, captureOut},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetViper(t)
			viper.Set("no-color", true)
			primary := tc.captureW(t)
			other := tc.otherW(t)

			tc.fn("test message for %s", tc.name)

			assert.Contains(t, primary.String(), "test message for "+tc.name,
				"%s should write message to its expected writer", tc.name)
			assert.Empty(t, other.String(),
				"%s must not write to the other writer", tc.name)
		})
	}
}

// --- Group 6: Spinner smoke test ---

func TestStartStopSpinner_DoesNotPanic(t *testing.T) {
	resetViper(t)
	viper.Set("no-color", true) // avoid terminal rendering in CI
	out := captureOut(t)

	StartSpinner("resolving dependencies")
	StopSpinner()

	// In no-color mode, StartSpinner prints a plain line; StopSpinner is a no-op.
	assert.Contains(t, out.String(), "resolving dependencies")
}

func TestStopSpinner_Idempotent(t *testing.T) {
	resetViper(t)
	viper.Set("no-color", true)
	_ = captureOut(t)

	// StopSpinner with nothing running must not panic.
	StopSpinner()
	StopSpinner()
}

func TestStartSpinner_JSONMode(t *testing.T) {
	resetViper(t)
	viper.Set("json", true)
	out := captureOut(t)

	StartSpinner("fetching clpfd")
	StopSpinner()

	var msg jsonMsg
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(out.String())), &msg))
	assert.Equal(t, "info", msg.Level)
	assert.Equal(t, "fetching clpfd", msg.Message)
}

// --- Group 7: NewDownloadBar ---

func TestNewDownloadBar_NoColor(t *testing.T) {
	resetViper(t)
	viper.Set("no-color", true)
	out := captureOut(t)

	bar := NewDownloadBar(1024, "downloading clpfd")

	require.NotNil(t, bar)
	assert.Contains(t, out.String(), "downloading clpfd")
	assert.NotContains(t, out.String(), "\x1b[", "no-color mode must not emit ANSI codes")
}

func TestNewDownloadBar_JSONMode(t *testing.T) {
	resetViper(t)
	viper.Set("json", true)
	out := captureOut(t)

	bar := NewDownloadBar(1024, "downloading clpfd")

	require.NotNil(t, bar)
	var msg jsonMsg
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(out.String())), &msg))
	assert.Equal(t, "info", msg.Level)
	assert.Equal(t, "downloading clpfd", msg.Message)
}

func TestNewDownloadBar_ColorMode_DoesNotPanic(t *testing.T) {
	// Exercises the progressbar config path without requiring a real TTY.
	resetViper(t)
	_ = captureOut(t)

	bar := NewDownloadBar(1024, "downloading clpfd")

	require.NotNil(t, bar)
}

func TestRewriteSubSecondElapsed_RewritesZeroSecondTokenToMilliseconds(t *testing.T) {
	got := rewriteSubSecondElapsed("progress [0s] done", 128*time.Millisecond)
	assert.Equal(t, "progress [128ms] done", got)
}

func TestRewriteSubSecondElapsed_LeavesLongDurationsUnchanged(t *testing.T) {
	got := rewriteSubSecondElapsed("progress [0s] done", 2*time.Second)
	assert.Equal(t, "progress [0s] done", got)
}
