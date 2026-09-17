package search

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestParseDetailFixtures(t *testing.T) {
	n := mustParseDetail(t, "testdata/portal.golden")
	if n.InternalID != "4090391205" || n.ReferenceNumber != "0000331977" || n.Buyer != "Defence Construction Canada - Pacific Region" {
		t.Fatalf("portal ids %+v", n)
	}
	if n.ProjectType != "NPP - Notice of Proposed Procurement (Formal)" || n.ProjectNumber != "IE12347R_CN87913" || n.Title != "Request for Expression of Interest (RFEOI) for A-Jetty Replacement" {
		t.Fatalf("portal title %+v", n)
	}
	if n.SourceID != "FD.CC.B.C..862067.1143664.C111108" || n.Location != "Canada, British Columbia" || n.JobLocation != "CFB Esquimalt" || n.PurchaseType != "One Time Only- Delivery Date:2026/12/09" {
		t.Fatalf("portal place %+v", n)
	}
	if n.Published != "2026/09/17 08:56:08 AM PDT" || n.Closing != "2026/10/01 02:00:00 PM PDT" || n.BidIntent != "Not Available" || n.BidSubmission != "Electronic Bid Submission" || n.Pricing != "In Bid Questions" {
		t.Fatalf("portal dates %+v", n)
	}
	if n.ContactName != "Danielle Flynn" || n.ContactPhone != "778-789-3070" || n.ContactEmail != "danielle.flynn@dcc-cdc.gc.ca" || n.Description == "" || len(n.AgreementTypes) != 2 {
		t.Fatalf("portal extra %+v", n)
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
	if note.InternalID != "444165230573" || note.ReferenceNumber != "00005204274" || note.ProjectType != "RFQ - Request for Quote (Informal)" || note.ProjectNumber != "RRFB0129" {
		t.Fatalf("notice %+v", note)
	}
	if note.ContactName != "Adolfo Vargas" || note.ContactEmail != "avargas@divertns.ca" || note.ContactPhone != "" || note.JobLocation != "" || note.Pricing != "" || len(note.AgreementTypes) != 0 {
		t.Fatalf("notice must not invent fields %+v", note)
	}
	f, err := os.Open("testdata/categories.golden")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	merx, gsin, unspsc, err := ParseCategories(f)
	if err != nil || len(merx) != 1 || merx[0].Code != "C" || merx[0].Name != "Construction" || len(gsin) != 1 || gsin[0].Code != "5114BA" || len(unspsc) != 0 {
		t.Fatalf("cats %v %v %v %v", merx, gsin, unspsc, err)
	}
	if got := DetailCandidates("444165230573"); len(got) != 1 || !strings.Contains(got[0], "/view-notice/444165230573") {
		t.Fatalf("cand %v", got)
	}
	if DetailCandidates("0000331977") != nil {
		t.Fatal("public number has no pretty URL")
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
