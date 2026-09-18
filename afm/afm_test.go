package afm

import (
	"errors"
	"strings"
	"testing"
)

const sampleAFM = `StartFontMetrics 4.1
FontName TestFont
FullName Test Font
FamilyName Test
StartCharMetrics 3
C 65 ; WX 722 ; N A ;
C 66 ; WX 667 ; N B ;
C 67 ; WX 667 ; N C ;
EndCharMetrics
StartKernData
StartKernPairs 2
KPX A B -40
KPX A C -80
EndKernPairs
EndKernData
EndFontMetrics
`

func TestParseKerningPairs(t *testing.T) {
	font, err := Parse(strings.NewReader(sampleAFM))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	tests := []struct {
		first, second string
		want          int
	}{
		{"A", "B", -40},
		{"A", "C", -80},
	}
	for _, tt := range tests {
		got, ok := font.KerningFor(tt.first, tt.second)
		if !ok {
			t.Errorf("KerningFor(%q, %q): not found", tt.first, tt.second)
			continue
		}
		if got != tt.want {
			t.Errorf("KerningFor(%q, %q) = %d, want %d", tt.first, tt.second, got, tt.want)
		}
	}

	if _, ok := font.KerningFor("B", "A"); ok {
		t.Error("KerningFor(B, A) found a pair, but KPX pairs are directional")
	}
	if len(font.Kerns) != 2 {
		t.Errorf("len(font.Kerns) = %d, want 2", len(font.Kerns))
	}
}

func TestParseKerningPairsBadAmount(t *testing.T) {
	src := strings.Replace(sampleAFM, "KPX A B -40", "KPX A B abc", 1)

	_, err := Parse(strings.NewReader(src))
	if err == nil {
		t.Fatal("Parse: expected error for non-integer kerning amount, got nil")
	}
	var perr *ParseError
	if !errors.As(err, &perr) {
		t.Fatalf("Parse: error is not a *ParseError: %v", err)
	}
	if perr.Line != 12 {
		t.Errorf("ParseError.Line = %d, want 12", perr.Line)
	}
}

func TestParseKerningPairsMissingField(t *testing.T) {
	src := strings.Replace(sampleAFM, "KPX A B -40", "KPX A", 1)

	_, err := Parse(strings.NewReader(src))
	if err == nil {
		t.Fatal("Parse: expected error for KPX line missing fields, got nil")
	}
	var perr *ParseError
	if !errors.As(err, &perr) {
		t.Fatalf("Parse: error is not a *ParseError: %v", err)
	}
}
