// Package output prints results as text, tables or one JSON object.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
)

// Printer writes to the two standard streams. Stable output goes to Out.
type Printer struct {
	Out, Err io.Writer
	JSON     bool
}

// Object prints v as indented JSON.
func (p *Printer) Object(v any) error {
	enc := json.NewEncoder(p.Out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// Linef prints one text line to stdout.
func (p *Printer) Linef(format string, a ...any) {
	fmt.Fprintf(p.Out, format+"\n", a...)
}

// Warnf prints one line to stderr.
func (p *Printer) Warnf(format string, a ...any) {
	fmt.Fprintf(p.Err, format+"\n", a...)
}

// Table prints aligned rows. The first row is the header.
func (p *Printer) Table(rows [][]string) {
	w := tabwriter.NewWriter(p.Out, 0, 4, 2, ' ', 0)
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				fmt.Fprint(w, "\t")
			}
			fmt.Fprint(w, cell)
		}
		fmt.Fprintln(w)
	}
	w.Flush()
}
