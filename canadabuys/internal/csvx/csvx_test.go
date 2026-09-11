package csvx

import (
	"strings"
	"testing"
)

func TestStripsBOM(t *testing.T) {
	r, err := New(strings.NewReader("\uFEFF\"a\",\"b\"\n\"x\ny\",2\n"))
	if err != nil {
		t.Fatal(err)
	}
	if r.Header()[0] != "a" {
		t.Fatalf("header[0] = %q, want no BOM", r.Header()[0])
	}
	if err := r.Next(); err != nil || r.Get("a") != "x\ny" || r.Get("b") != "2" {
		t.Fatalf("quoted newline split the record: %v, %q %q", err, r.Get("a"), r.Get("b"))
	}
}
