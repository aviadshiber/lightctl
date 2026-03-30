package iostreams

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

// IOStreams provides standard I/O streams with TTY-aware helpers.
type IOStreams struct {
	Out    io.Writer
	ErrOut io.Writer
	quiet  bool
}

// New creates a default IOStreams writing to stdout/stderr.
func New() *IOStreams {
	return &IOStreams{
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}
}

// SetQuiet enables or disables quiet mode. When quiet, Printf is suppressed.
func (s *IOStreams) SetQuiet(q bool) {
	s.quiet = q
}

// IsTerminal returns true if stdout is a terminal.
func (s *IOStreams) IsTerminal() bool {
	f, ok := s.Out.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// ColorEnabled returns true when colored output is appropriate.
// Respects NO_COLOR (https://no-color.org/) and CLICOLOR conventions.
func (s *IOStreams) ColorEnabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("CLICOLOR") == "0" {
		return false
	}
	return s.IsTerminal()
}

// Printf writes formatted output, suppressed in quiet mode.
func (s *IOStreams) Printf(format string, args ...interface{}) {
	if s.quiet {
		return
	}
	fmt.Fprintf(s.Out, format, args...)
}

// Errorf writes formatted output to stderr, always printed.
func (s *IOStreams) Errorf(format string, args ...interface{}) {
	fmt.Fprintf(s.ErrOut, format, args...)
}

// Success prints a green check mark message.
func (s *IOStreams) Success(msg string) {
	if s.ColorEnabled() {
		fmt.Fprintf(s.ErrOut, "\033[32m✓\033[0m %s\n", msg)
	} else {
		fmt.Fprintf(s.ErrOut, "✓ %s\n", msg)
	}
}

// Warning prints a yellow exclamation message.
func (s *IOStreams) Warning(msg string) {
	if s.ColorEnabled() {
		fmt.Fprintf(s.ErrOut, "\033[33m!\033[0m %s\n", msg)
	} else {
		fmt.Fprintf(s.ErrOut, "! %s\n", msg)
	}
}

// Error prints a red X message.
func (s *IOStreams) Error(msg string) {
	if s.ColorEnabled() {
		fmt.Fprintf(s.ErrOut, "\033[31m✗\033[0m %s\n", msg)
	} else {
		fmt.Fprintf(s.ErrOut, "✗ %s\n", msg)
	}
}
