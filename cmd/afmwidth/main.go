// Command afmwidth measures the rendered width of a string against an AFM
// font metrics file, without needing the actual font installed.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/zoemartinez8/afm-metrics/afm"
)

func main() {
	size := flag.Float64("size", 12, "font size in points, used to convert AFM units to a point width")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [-size pt] <font.afm> <text>\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	args := flag.Args()
	if len(args) != 2 {
		flag.Usage()
		os.Exit(2)
	}
	path, text := args[0], args[1]

	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()

	font, err := afm.Parse(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
		os.Exit(1)
	}

	units, err := font.StringWidth(text)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
		os.Exit(1)
	}

	points := float64(units) * *size / 1000
	fmt.Printf("%d units (%.2fpt at %gpt)\n", units, points, *size)
}
