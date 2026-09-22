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

	// Parts lists the components of a composite glyph, from the line's PCC
	// fields, e.g. "Aacute" built from "A" plus "acute" at an offset. Empty
	// for a simple (non-composite) glyph.
	Parts []CompositePart
}

// CompositePart is one component of a composite glyph, from a PCC field
// such as "PCC acute 195 0": place the glyph named acute at a 195/0 unit
// offset from the composite's origin.
type CompositePart struct {
	Name   string
	DeltaX int
	DeltaY int
}

// KerningPair is one entry from the KernPairs section of an AFM file, e.g.
// "KPX A C -40": the amount to add to the advance width when the glyph
// named First is immediately followed by the glyph named Second.
type KerningPair struct {
	First  string
	Second string
	Amount int // in 1/1000 units of the font's em square
}

// Font holds the metrics parsed from a single AFM file.
type Font struct {
	Name       string
	FullName   string
	FamilyName string
	Chars      []CharMetric
	Kerns      []KerningPair

	byName    map[string]int
	byCode    map[int]int
	kernIndex map[[2]string]int
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

// KerningFor returns the kerning adjustment to apply when the glyph named
// first is immediately followed by the glyph named second, in 1/1000 em
// units. Kerning pairs are directional: a pair for ("A", "V") says nothing
// about ("V", "A").
func (f *Font) KerningFor(first, second string) (int, bool) {
	i, ok := f.kernIndex[[2]string{first, second}]
	if !ok {
		return 0, false
	}
	return f.Kerns[i].Amount, true
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
	font := &Font{byName: map[string]int{}, byCode: map[int]int{}, kernIndex: map[[2]string]int{}}

	scanner := bufio.NewScanner(r)
	lineNo := 0
	inMetrics := false
	inKerning := false

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
		case strings.HasPrefix(trimmed, "StartKernPairs"):
			inKerning = true
		case strings.HasPrefix(trimmed, "EndKernPairs"):
			inKerning = false
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
		case inKerning && strings.HasPrefix(trimmed, "KPX "):
			kp, err := parseKPXLine(line, lineNo)
			if err != nil {
				return nil, err
			}
			idx := len(font.Kerns)
			font.Kerns = append(font.Kerns, kp)
			font.kernIndex[[2]string{kp.First, kp.Second}] = idx
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
	haveCC := false
	wantParts := 0
	ccCol := 0

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
		case "CC":
			if len(rest) < 2 {
				return cm, newParseError(lineNo, f.col, line, "CC requires a base glyph name and a part count")
			}
			n, err := strconv.Atoi(rest[1])
			if err != nil {
				return cm, newParseError(lineNo, fieldValueCol(f, rest[1]), line,
					fmt.Sprintf("invalid composite part count %q: must be an integer", rest[1]))
			}
			haveCC = true
			wantParts = n
			ccCol = f.col
		case "PCC":
			if len(rest) < 3 {
				return cm, newParseError(lineNo, f.col, line, "PCC requires a part name and an x/y displacement")
			}
			dx, err := strconv.Atoi(rest[1])
			if err != nil {
				return cm, newParseError(lineNo, fieldValueCol(f, rest[1]), line,
					fmt.Sprintf("invalid x displacement %q: must be an integer", rest[1]))
			}
			dy, err := strconv.Atoi(rest[2])
			if err != nil {
				return cm, newParseError(lineNo, fieldValueCol(f, rest[2]), line,
					fmt.Sprintf("invalid y displacement %q: must be an integer", rest[2]))
			}
			cm.Parts = append(cm.Parts, CompositePart{Name: rest[0], DeltaX: dx, DeltaY: dy})
		}
	}

	if !haveWidth {
		return cm, newParseError(lineNo, baseCol, line, "character metric is missing a WX (width) field")
	}
	if !haveName {
		return cm, newParseError(lineNo, baseCol, line, "character metric is missing an N (name) field")
	}
	if haveCC && len(cm.Parts) != wantParts {
		return cm, newParseError(lineNo, ccCol, line,
			fmt.Sprintf("CC declares %d composite part(s) but %d PCC field(s) follow", wantParts, len(cm.Parts)))
	}
	return cm, nil
}

// splitWords breaks line into whitespace-separated tokens, along with the
// 1-based column where each one starts. Unlike splitFields, KernPairs lines
// have no ';' separators to key off of.
func splitWords(line string) []field {
	runes := []rune(line)
	var words []field
	i := 0
	for i < len(runes) {
		for i < len(runes) && (runes[i] == ' ' || runes[i] == '\t') {
			i++
		}
		if i >= len(runes) {
			break
		}
		start := i
		for i < len(runes) && runes[i] != ' ' && runes[i] != '\t' {
			i++
		}
		words = append(words, field{text: string(runes[start:i]), col: start + 1})
	}
	return words
}

// parseKPXLine parses a single "KPX <first> <second> <amount>" line from a
// KernPairs section.
func parseKPXLine(line string, lineNo int) (KerningPair, error) {
	words := splitWords(line)
	if len(words) < 4 {
		col := 1
		if len(words) > 0 {
			col = words[0].col
		}
		return KerningPair{}, newParseError(lineNo, col, line,
			"KPX requires two glyph names and a kerning amount")
	}

	amount, err := strconv.Atoi(words[3].text)
	if err != nil {
		return KerningPair{}, newParseError(lineNo, words[3].col, line,
			fmt.Sprintf("invalid kerning amount %q: must be an integer", words[3].text))
	}

	return KerningPair{First: words[1].text, Second: words[2].text, Amount: amount}, nil
}
