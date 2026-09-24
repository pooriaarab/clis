package docs

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"merx-cli/internal/httpx"
)

func TestSkipMatchingChecksum(t *testing.T) {
	p, hits := []byte("already here"), 0
	dest := t.TempDir()
	os.WriteFile(filepath.Join(dest, "spec.pdf"), p, 0o644)
	writeManifest(dest, &Manifest{Documents: []File{{ID: "77", Filename: "spec.pdf", SHA256: sum(p)}}})
	got, err := Fetch(testClient(), docsServer(t, nil, false, nil, &hits, p, "application/pdf", "", nil).URL, "99", dest)
	if err != nil || hits != 0 || got.Documents[0].SHA256 != sum(p) {
		t.Fatalf("%d %+v %v", hits, got, err)
	}
}

func TestAcceptRecordsFilenameAndTime(t *testing.T) {
	posts, dest := 0, t.TempDir()
	got, err := Fetch(testClient(), docsServer(t, read(t, "../search/testdata/req-ack.golden"), true, &posts, nil, []byte("pdf"), "application/pdf", "", nil).URL, "99", dest)
	raw, _ := os.ReadFile(filepath.Join(dest, "spec.pdf"))
	saved, loadErr := loadManifest(dest)
	if err != nil || posts != 1 || got.Acceptances[0].Filename != "Confidentiality-NDA.pdf" || loadErr != nil || saved.Acceptances[0] != got.Acceptances[0] {
		t.Fatalf("%d %+v %v", posts, got, err)
	} else if _, e := time.Parse(time.RFC3339, got.Acceptances[0].AcceptedAt); e != nil || string(raw) != "pdf" {
		t.Fatalf("time/file %v %q", e, raw)
	}
}

func TestUnparseableAckDoesNotPost(t *testing.T) {
	posts, dest := 0, t.TempDir()
	got, err := Fetch(testClient(), docsServer(t, read(t, "../search/testdata/req-ack-bad.golden"), true, &posts, nil, nil, "application/pdf", "", nil).URL, "99", dest)
	raw, _ := os.ReadFile(filepath.Join(dest, manFile))
	if err != nil || posts != 0 || len(got.Acceptances) != 0 || got.SkippedGates[0].Status != gateSkip || !bytes.Contains(raw, []byte(gateSkip)) {
		t.Fatalf("%d %+v %v %s", posts, got, err, raw)
	}
}

func TestDownloadHop(t *testing.T) {
	for _, tc := range [][4]string{
		{"application/pdf", `attachment; filename="got.pdf"`, "PDF", "got.pdf"},
		{"text/javascript", "", "alert(1)", ""},
		{"application/pdf", `attachment; filename="../a/x.pdf"`, "PDF", "x.pdf"},
	} {
		var last string
		dest := t.TempDir()
		_, err := Fetch(testClient(), docsServer(t, nil, false, nil, nil, []byte(tc[2]), tc[0], tc[1], &last).URL, "99", dest)
		if tc[3] == "" {
			ents, _ := os.ReadDir(dest)
			if err == nil || !strings.Contains(err.Error(), "javascript") || len(ents) > 0 {
				t.Fatalf("%v %v", err, ents)
			}
			continue
		}
		raw, _ := os.ReadFile(filepath.Join(dest, tc[3]))
		if err != nil || last != "/private/solicitations/99/abstract/docs-items/77/attachment-preview-download" || string(raw) != tc[2] {
			t.Fatalf("%s %q %v", last, raw, err)
		}
		if _, e := os.Stat(filepath.Join(dest, "..", "a", "x.pdf")); tc[3] == "x.pdf" && e == nil {
			t.Fatal("escaped")
		}
	}
}

func docsServer(t *testing.T, ack []byte, gate bool, posts, hits *int, file []byte, ct, disp string, last *string) *httptest.Server {
	t.Helper()
	ok := false
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case r.Method == http.MethodPost:
			if posts != nil {
				*posts++
			}
			if r.ParseForm() != nil || r.Form.Get("_eventId_accept") == "" || r.Form.Get("_csrf") != "tok" {
				http.Error(w, "bad", 400)
				return
			}
			ok = true
			http.Redirect(w, r, "/public/solicitations/99/abstract/docs-items", 302)
		case strings.Contains(p, "/req-ack"):
			w.Write(ack)
		case strings.Contains(p, "attachment-preview-dialog"):
			w.Header().Set("Content-Type", "text/javascript;charset=UTF-8")
			w.Write(read(t, "testdata/preview-dialog.golden"))
		case strings.Contains(p, "attachment-preview-download"):
			if hits != nil {
				*hits++
			}
			if last != nil {
				*last = p
			}
			w.Header().Set("Content-Type", ct)
			if disp != "" {
				w.Header().Set("Content-Disposition", disp)
			}
			w.Write(file)
		case strings.HasSuffix(p, "/docs-items") && r.Header.Get("X-Requested-With") == "XMLHttpRequest" && !(gate && !ok):
			w.Header().Set("Content-Type", "text/javascript")
			w.Write([]byte(`$("#innerTabContent").html('<a href="/public/solicitations/99/abstract/docs-items/77/attachment-preview-dialog">spec.pdf</a>');`))
		case strings.HasSuffix(p, "/docs-items"):
			http.Redirect(w, r, "/private/supplier/solicitations/99/req-ack", 302)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(s.Close)
	return s
}

func testClient() *httpx.Client {
	return &httpx.Client{HTTP: &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, Delay: time.Millisecond}
}

func read(t *testing.T, p string) []byte {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
