# afm-metrics

A Go library and CLI for reading Adobe Font Metrics (`.afm`) files: the
plain-text sidecar files that ship with PostScript fonts and describe glyph
advance widths. If you're laying out text (line breaking, centering a title,
sizing a column) you often need to know how wide a string will render
*without* linking a full font rasterizer. AFM files give you that, and
they're just text.

## Library usage

```go
package main

import (
	"fmt"
	"os"

	"github.com/zoemartinez8/afm-metrics/afm"
)

func main() {
	f, err := os.Open("Helvetica.afm")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	font, err := afm.Parse(f)
	if err != nil {
		panic(err)
	}

	units, err := font.StringWidth("Hello, world")
	if err != nil {
		panic(err)
	}

	// AFM widths are in 1/1000 em units; multiply by point size and divide
	// by 1000 to get a rendered width in points.
	fmt.Printf("%d units, %.2fpt at 12pt\n", units, float64(units)*12/1000)
}
```

## CLI usage

```
$ go build -o afmwidth ./cmd/afmwidth
$ ./afmwidth -size 14 Helvetica.afm "Hello, world"
6552 units (91.73pt at 14pt)
```

## Error messages

Malformed CharMetrics lines are reported with a line number, a column
number, and the offending source line, the way a compiler would report a
syntax error, rather than a bare "invalid file":

```
$ ./afmwidth Broken.afm "A"
Broken.afm: line 4, column 8: invalid width "abc": must be an integer
    C 65 ; WX abc ; N A ;
           ^
```

## Scope

The parser currently reads `FontName`, `FullName`, `FamilyName`, the
`CharMetrics` section (character code, width, glyph name, and composite
glyphs via `CC`/`PCC`, exposed as `CharMetric.Parts`), and the `KernPairs`
section (`KPX` lines, by glyph name, via `Font.KerningFor`). The other
header fields aren't parsed yet.

`Font.StringWidth` treats each rune in the input as a character code
directly, which is correct for ASCII text against the base-14 fonts'
StandardEncoding but not a general Unicode encoding mapping.

## License

MIT, see [LICENSE](LICENSE).
