package checker

import (
	"regexp"
	"strconv"
	"strings"
)

// Severity of a parsed diagnostic line.
type Severity string

const (
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

// Diagnostic is a single warning or error extracted from swipl output.
//
// File/Line/Col are best-effort: swipl does not always include location
// information (e.g. "Warning: Goal (directive) failed: ..."). In that case
// File is empty and Line/Col are zero.
type Diagnostic struct {
	Severity Severity `json:"severity"`
	File     string   `json:"file,omitempty"`
	Line     int      `json:"line,omitempty"`
	Col      int      `json:"col,omitempty"`
	Message  string   `json:"message"`
}

// CheckResult is the structured outcome of a `prolm check` run.
type CheckResult struct {
	Diagnostics []Diagnostic `json:"diagnostics"`
	RawStdout   string       `json:"-"`
	RawStderr   string       `json:"-"`
}

// Warnings returns diagnostics whose severity is warning.
func (r *CheckResult) Warnings() []Diagnostic {
	return r.filter(SeverityWarning)
}

// Errors returns diagnostics whose severity is error.
func (r *CheckResult) Errors() []Diagnostic {
	return r.filter(SeverityError)
}

func (r *CheckResult) filter(s Severity) []Diagnostic {
	var out []Diagnostic
	for _, d := range r.Diagnostics {
		if d.Severity == s {
			out = append(out, d)
		}
	}
	return out
}

// diagRe matches the common swipl diagnostic shapes:
//
//	Warning: /path/file.pl:10:5: message
//	Warning: /path/file.pl:10: message
//	ERROR:   /path/file.pl:3:5: message
//	Warning: message-without-location
//
// Windows drive letters (C:\...) are not supported in Phase 1 (see ROADMAP 4.7).
var diagRe = regexp.MustCompile(`^(Warning|ERROR):\s*(?:(/[^\s:]+|[^\s:/][^\s:]*\.pl):(\d+)(?::(\d+))?:\s*)?(.*)$`)

// Parse converts swipl stdout+stderr into a structured CheckResult.
//
// The parser is deliberately tolerant: unrecognised lines are ignored and an
// empty input yields a zero-value result. Multi-line diagnostics (swipl prints
// continuation lines indented with whitespace) are coalesced into the preceding
// Diagnostic.Message.
func Parse(stdout, stderr string) *CheckResult {
	res := &CheckResult{RawStdout: stdout, RawStderr: stderr}

	combined := stdout
	if stderr != "" {
		if combined != "" && !strings.HasSuffix(combined, "\n") {
			combined += "\n"
		}
		combined += stderr
	}

	var current *Diagnostic
	for _, line := range strings.Split(combined, "\n") {
		trimmed := strings.TrimRight(line, "\r")
		if trimmed == "" {
			current = nil
			continue
		}
		m := diagRe.FindStringSubmatch(trimmed)
		if m == nil {
			// Indented continuation line — append to the current diagnostic.
			if current != nil && (strings.HasPrefix(trimmed, "\t") || strings.HasPrefix(trimmed, " ")) {
				cont := strings.TrimSpace(trimmed)
				if cont != "" {
					current.Message = strings.TrimSpace(current.Message + " " + cont)
				}
			} else {
				current = nil
			}
			continue
		}
		sev := SeverityWarning
		if m[1] == "ERROR" {
			sev = SeverityError
		}
		d := Diagnostic{
			Severity: sev,
			File:     m[2],
			Message:  strings.TrimSpace(m[5]),
		}
		if m[3] != "" {
			d.Line, _ = strconv.Atoi(m[3])
		}
		if m[4] != "" {
			d.Col, _ = strconv.Atoi(m[4])
		}
		res.Diagnostics = append(res.Diagnostics, d)
		current = &res.Diagnostics[len(res.Diagnostics)-1]
	}

	return res
}
