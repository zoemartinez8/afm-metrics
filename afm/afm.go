// Package afm reads Adobe Font Metrics (.afm) files: the plain-text sidecar
// files that describe glyph widths for a font without needing the font's
// outlines at all. That's enough to lay out text (line breaking, centering,
// column widths) without linking a rasterizer.
package afm

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// CharMetric is one glyph's entry from the CharMetrics section of an AFM
// file, e.g. "C 32 ; WX 278 ; N space ;".
type CharMetric struct {
	Code  int // font-internal character code, or -1 if unencoded
	Width int // advance width, in 1/1000 units of the font's em square
	Name  string
}

// Font holds the metrics parsed from a single AFM file.
type Font struct {
	Name       string
	FullName   string
	FamilyName string
	Chars      []CharMetric

	byName map[string]int
	byCode map[int]int
}

// WidthOf returns the advance width of the glyph with the given PostScript
// name, in 1/1000 em units.
func (f *Font) WidthOf(name string) (int, bool) {
	i, ok := f.byName[name]
	if !ok {
		return 0, false
	}
	return f.Chars[i].Width, true
}

// WidthOfCode returns the advance width of the glyph at the given character
// code, in 1/1000 em units.
func (f *Font) WidthOfCode(code int) (int, bool) {
	i, ok := f.byCode[code]
	if !ok {
		return 0, false
	}
	return f.Chars[i].Width, true
}

// StringWidth sums the advance widths of s, treating each rune as a
// character code in the font's encoding. This only gives correct results
// for text within the font's built-in encoding (StandardEncoding for most
// base-14 fonts); it is not general Unicode shaping.
func (f *Font) StringWidth(s string) (int, error) {
	total := 0
	for _, r := range s {
		w, ok := f.WidthOfCode(int(r))
		if !ok {
			return 0, fmt.Errorf("no metric for %q (code %d) in font %s", r, r, f.Name)
		}
		total += w
	}
	return total, nil
}

// Parse reads an AFM file from r and returns its font metrics. Errors from
// malformed CharMetrics lines are returned as *ParseError, with the line and
// column of the offending token.
func Parse(r io.Reader) (*Font, error) {
	font := &Font{byName: map[string]int{}, byCode: map[int]int{}}

	scanner := bufio.NewScanner(r)
	lineNo := 0
	inMetrics := false

	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		switch {
		case strings.HasPrefix(trimmed, "StartCharMetrics"):
			inMetrics = true
		case strings.HasPrefix(trimmed, "EndCharMetrics"):
			inMetrics = false
		case strings.HasPrefix(trimmed, "FontName"):
			font.Name = strings.TrimSpace(strings.TrimPrefix(trimmed, "FontName"))
		case strings.HasPrefix(trimmed, "FullName"):
			font.FullName = strings.TrimSpace(strings.TrimPrefix(trimmed, "FullName"))
		case strings.HasPrefix(trimmed, "FamilyName"):
			font.FamilyName = strings.TrimSpace(strings.TrimPrefix(trimmed, "FamilyName"))
		case inMetrics && strings.HasPrefix(trimmed, "C "):
			cm, err := parseCharMetricLine(line, lineNo)
			if err != nil {
				return nil, err
			}
			idx := len(font.Chars)
			font.Chars = append(font.Chars, cm)
			font.byName[cm.Name] = idx
			if cm.Code >= 0 {
				font.byCode[cm.Code] = idx
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading font metrics: %w", err)
	}
	if font.Name == "" {
		return nil, fmt.Errorf("no FontName found in input")
	}
	return font, nil
}

// field is a ';'-separated segment of a CharMetrics line, e.g. "WX 278",
// along with the 1-based column where it starts in the original line.
type field struct {
	text string
	col  int
}

func splitFields(line string) []field {
	runes := []rune(line)
	var fields []field
	start := 0
	for i := 0; i <= len(runes); i++ {
		if i < len(runes) && runes[i] != ';' {
			continue
		}
		raw := string(runes[start:i])
		leading := len(raw) - len(strings.TrimLeft(raw, " \t"))
		text := strings.TrimSpace(raw)
		if text != "" {
			fields = append(fields, field{text: text, col: start + leading + 1})
		}
		start = i + 1
	}
	return fields
}

// fieldValueCol finds the column of value within f, for pointing errors at
// the specific token that failed to parse rather than the field as a whole.
func fieldValueCol(f field, value string) int {
	if idx := strings.Index(f.text, value); idx >= 0 {
		return f.col + idx
	}
	return f.col
}

func parseCharMetricLine(line string, lineNo int) (CharMetric, error) {
	cm := CharMetric{Code: -1}
	haveWidth := false
	haveName := false

	fields := splitFields(line)
	baseCol := 1
	if len(fields) > 0 {
		baseCol = fields[0].col
	}

	for _, f := range fields {
		parts := strings.Fields(f.text)
		if len(parts) == 0 {
			continue
		}
		key, rest := parts[0], parts[1:]

		switch key {
		case "C":
			if len(rest) == 0 {
				return cm, newParseError(lineNo, f.col, line, "C requires a character code")
			}
			code, err := strconv.Atoi(rest[0])
			if err != nil {
				return cm, newParseError(lineNo, fieldValueCol(f, rest[0]), line,
					fmt.Sprintf("invalid character code %q: must be an integer", rest[0]))
			}
			cm.Code = code
		case "WX":
			if len(rest) == 0 {
				return cm, newParseError(lineNo, f.col, line, "WX requires a width value")
			}
			width, err := strconv.Atoi(rest[0])
			if err != nil {
				return cm, newParseError(lineNo, fieldValueCol(f, rest[0]), line,
					fmt.Sprintf("invalid width %q: must be an integer", rest[0]))
			}
			cm.Width = width
			haveWidth = true
		case "N":
			if len(rest) == 0 {
				return cm, newParseError(lineNo, f.col, line, "N requires a glyph name")
			}
			cm.Name = rest[0]
			haveName = true
		}
	}

	if !haveWidth {
		return cm, newParseError(lineNo, baseCol, line, "character metric is missing a WX (width) field")
	}
	if !haveName {
		return cm, newParseError(lineNo, baseCol, line, "character metric is missing an N (name) field")
	}
	return cm, nil
}
