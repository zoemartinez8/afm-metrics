package afm

import (
	"fmt"
	"strings"
)

// ParseError describes a problem found while reading an AFM file, pinned to
// the exact line and column of the offending token so a caller can report it
// the way a compiler would rather than just "invalid file".
type ParseError struct {
	Line   int
	Column int
	Source string // the raw text of the offending line
	Msg    string
}

func (e *ParseError) Error() string {
	col := e.Column
	if col < 1 {
		col = 1
	}
	pointer := strings.Repeat(" ", col-1) + "^"
	return fmt.Sprintf("line %d, column %d: %s\n    %s\n    %s",
		e.Line, col, e.Msg, e.Source, pointer)
}

func newParseError(line, col int, source, msg string) *ParseError {
	return &ParseError{Line: line, Column: col, Source: source, Msg: msg}
}
