package csvx

import (
	"bufio"
	"encoding/csv"
	"io"
)

// Reader yields one record per Next call and never buffers the file.
type Reader struct {
	r   *csv.Reader
	hdr []string
	idx map[string]int
	row []string
}

// New wraps in as a record stream: the BOM is dropped before parsing,
// then quoted newlines and long fields are tolerated.
func New(in io.Reader) (*Reader, error) {
	br := bufio.NewReader(in)
	if bom, _ := br.Peek(3); string(bom) == "\uFEFF" {
		br.Discard(3)
	}
	r := csv.NewReader(br)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	hdr, err := r.Read()
	if err != nil {
		return nil, err
	}
	r.ReuseRecord = true
	idx := make(map[string]int, len(hdr))
	for i, h := range hdr {
		idx[h] = i
	}
	return &Reader{r: r, hdr: hdr, idx: idx}, nil
}

func (x *Reader) Header() []string { return x.hdr }

// Next advances to the next record, or returns io.EOF at the end.
func (x *Reader) Next() error {
	row, err := x.r.Read()
	x.row = row
	return err
}

// Get returns one field of the current record. Missing names are "".
func (x *Reader) Get(name string) string {
	i, ok := x.idx[name]
	if !ok || i >= len(x.row) {
		return ""
	}
	return x.row[i]
}
