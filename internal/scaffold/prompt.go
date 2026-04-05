package scaffold

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/prolm/prolm/internal/ui"
)

// prompter wraps a bufio.Scanner to read multiple prompted values from a
// single reader without the buffering-ahead problem of creating a new
// scanner per call.
type prompter struct {
	scanner *bufio.Scanner
}

func newPrompter(r io.Reader) *prompter {
	return &prompter{scanner: bufio.NewScanner(r)}
}

// ask prints a prompt and reads one line. If the user presses Enter
// without typing, defaultVal is returned.
func (p *prompter) ask(label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Fprintf(ui.Out, "  %s [%s]: ", label, defaultVal)
	} else {
		fmt.Fprintf(ui.Out, "  %s: ", label)
	}

	if p.scanner.Scan() {
		val := strings.TrimSpace(p.scanner.Text())
		if val != "" {
			return val
		}
	}
	return defaultVal
}
