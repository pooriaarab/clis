package search

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestParseDetailFixtures(t *testing.T) {
	n := mustParseDetail(t, "testdata/portal.golden")
	if n.InternalID != "4090391205" || n.ReferenceNumber != "0000331977" || n.Buyer != "Defence Construction Canada - Pacific Region" ||
		n.ProjectType != "NPP - Notice of Proposed Procurement (Formal)" || n.ProjectNumber != "IE12347R_CN87913" || n.Title != "Request for Expression of Interest (RFEOI) for A-Jetty Replacement" ||
		n.SourceID != "FD.CC.B.C..862067.1143664.C111108" || n.Location != "Canada, British Columbia" || n.JobLocation != "CFB Esquimalt" || n.PurchaseType != "One Time Only- Delivery Date:2026/12/09" ||
		n.Published != "2026/09/17 08:56:08 AM PDT" || n.Closing != "2026/10/01 02:00:00 PM PDT" || n.BidIntent != "Not Available" || n.BidSubmission != "Electronic Bid Submission" || n.Pricing != "In Bid Questions" ||
		n.ContactName != "Danielle Flynn" || n.ContactPhone != "778-789-3070" || n.ContactEmail != "danielle.flynn@dcc-cdc.gc.ca" || n.Description == "" || len(n.AgreementTypes) != 2 {
		t.Fatalf("portal %+v", n)
	}
	raw, _ := os.ReadFile("testdata/portal.golden")
	s := strings.ReplaceAll(string(raw), `id="g_10"`, `id="tmp"`)
	s = strings.ReplaceAll(s, `id="g_14"`, `id="g_10"`)
	s = strings.ReplaceAll(s, `id="tmp"`, `id="g_14"`)
	for i := 28; i >= 11; i-- {
		if i != 14 {
			s = strings.ReplaceAll(s, `id="g_`+strconv.Itoa(i)+`"`, `id="g_`+strconv.Itoa(i+40)+`"`)
		}
	}
	shifted, err := ParseDetail(strings.NewReader(s))
	if err != nil || shifted.ReferenceNumber != n.ReferenceNumber || shifted.Title != n.Title || shifted.ContactEmail != n.ContactEmail {
		t.Fatalf("id shift scrambled fields %+v %v", shifted, err)
	}
	note := mustParseDetail(t, "testdata/notice.golden")
	if note.InternalID != "444165230573" || note.ReferenceNumber != "00005204274" || note.ProjectType != "RFQ - Request for Quote (Informal)" || note.ProjectNumber != "RRFB0129" ||
		note.ContactName != "Adolfo Vargas" || note.ContactEmail != "avargas@divertns.ca" || note.ContactPhone != "" || note.JobLocation != "" || note.Pricing != "" || len(note.AgreementTypes) != 0 {
		t.Fatalf("notice %+v", note)
	}
	cf, _ := os.ReadFile("testdata/categories.golden")
	merx, gsin, unspsc, err := ParseCategories(strings.NewReader(string(cf)))
	if err != nil || len(merx) != 1 || merx[0].Code != "C" || merx[0].Name != "Construction" || len(gsin) != 1 || gsin[0].Code != "5114BA" || len(unspsc) != 0 {
		t.Fatalf("cats %v %v %v %v", merx, gsin, unspsc, err)
	}
	if got := DetailCandidates("444165230573"); len(got) != 1 || !strings.Contains(got[0], "/view-notice/444165230573") || DetailCandidates("0000331977") != nil {
		t.Fatalf("cand %v", got)
	}
	u, err := PickSearchURL(Page{Records: []Record{{Solicitation: "0000331977", DetailURL: Host + "/x/0000331977"}}}, "0000331977")
	if err != nil || !strings.HasSuffix(u, "/0000331977") {
		t.Fatalf("pick %q %v", u, err)
	}
}

func mustParseDetail(t *testing.T, path string) Notice {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	n, err := ParseDetail(strings.NewReader(string(b)))
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestCategoriesFetchHostOnly(t *testing.T) {
	var evilN int
	evil := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { evilN++ }))
	defer evil.Close()
	cats, _ := os.ReadFile("testdata/categories.golden")
	merx := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Write(cats) }))
	defer merx.Close()
	get := func(u string) ([]byte, error) {
		d := merx.URL
		if !strings.HasPrefix(u, Host) {
			d = evil.URL
		}
		r, err := http.Get(d)
		if err != nil {
			return nil, err
		}
		defer r.Body.Close()
		return io.ReadAll(r.Body)
	}
	for _, tc := range [][3]string{
		{"https://169.254.169.254/x/abstract/categories", "", ""},
		{"//evil.example/abstract/categories", "", ""},
		{"/public/solicitations/1/abstract/categories", "/public/solicitations/1/abstract/categories", "C"},
	} {
		n, err := ParseDetail(strings.NewReader(`<div class="mets-field"><span class="mets-field-label">Title</span><div class="mets-field-body">T</div></div><a href="` + evil.URL + `/x/abstract/categories" data-ajax-url="` + tc[0] + `"></a>`))
		LoadCategories(get, &n)
		if err != nil || n.Title != "T" || n.CategoriesPath != tc[1] || evilN != 0 || (tc[2] == "C") != (len(n.MERX) == 1 && n.MERX[0].Code == "C") {
			t.Fatalf("%s %+v %v evil=%d", tc[0], n, err, evilN)
		}
	}
}
