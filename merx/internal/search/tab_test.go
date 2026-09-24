package search

import (
	"os"
	"strings"
	"testing"
)

func TestUnwrap(t *testing.T) {
	// JS tab: pull the HTML fragment out of $("#innerTabContent").html('...').
	raw := readGolden(t, "testdata/docs-items.golden")
	got := Unwrap(string(raw))
	if strings.Contains(got, "innerTabContent") || !strings.HasPrefix(got, "<a href=") {
		t.Fatalf("unwrap %q", got)
	}
	// Already-HTML input is returned unchanged.
	plain := readGolden(t, "testdata/docs-plain.golden")
	if Unwrap(string(plain)) != string(plain) {
		t.Fatalf("passthrough %q", Unwrap(string(plain)))
	}
}

func TestParseDocList(t *testing.T) {
	// Two preview-dialog files; the "ignore me" link is not a preview.
	docs := mustParseDocs(t, "testdata/docs-items.golden")
	if len(docs) != 2 || docs[0] != (Doc{"77", "spec.pdf", "/public/solicitations/99/abstract/docs-items/77/attachment-preview-dialog"}) ||
		docs[1] != (Doc{"88", "addendum.docx", "/public/solicitations/99/abstract/docs-items/88/attachment-preview-dialog"}) {
		t.Fatalf("list %+v", docs)
	}

	// Preview dialog: the download hop uses attachment-preview-download.
	prev := mustParseDocs(t, "testdata/docs-preview.golden")
	if len(prev) != 1 || !strings.Contains(prev[0].URL, "/private/") || !strings.HasSuffix(prev[0].URL, "attachment-preview-download") {
		t.Fatalf("preview %+v", prev)
	}

	// Raw HTML: empty link text falls back to the id; a path is trimmed to the base name.
	plain := mustParseDocs(t, "testdata/docs-plain.golden")
	if len(plain) != 2 || plain[0] != (Doc{"55", "55", "/public/solicitations/99/abstract/docs-items/55/attachment-preview-dialog?x=1"}) ||
		plain[1] != (Doc{"66", "notes.pdf", "/public/solicitations/99/abstract/docs-items/66/attachment-preview-dialog"}) {
		t.Fatalf("plain %+v", plain)
	}

	// A categories tab has no attachment-preview-* links.
	empty := mustParseDocs(t, "testdata/categories.golden")
	if len(empty) != 0 {
		t.Fatalf("empty %+v", empty)
	}
}

func mustParseDocs(t *testing.T, path string) []Doc {
	t.Helper()
	docs, err := ParseDocList(strings.NewReader(string(readGolden(t, path))))
	if err != nil {
		t.Fatal(err)
	}
	return docs
}

func readGolden(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
