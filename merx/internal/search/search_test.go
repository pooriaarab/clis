package search

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestParseOpenFixture(t *testing.T) {
	f, err := os.Open("testdata/open_list.golden")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	page, err := Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 3254 || len(page.Records) != 2 {
		t.Fatalf("total %d rows %d", page.Total, len(page.Records))
	}
	n := page.Records[0]
	if n.InternalID != "444165376614" || n.Solicitation != "" || n.PortalPrefix != "" || n.Slug != "" {
		t.Fatalf("notice ids %+v", n)
	}
	if !strings.HasSuffix(n.DetailURL, "/public/supplier/interception/view-notice/444165376614") {
		t.Fatalf("notice url %q", n.DetailURL)
	}
	if n.Title != "DD for Six Highway 11 Projects and PD for One Highway 141 Project" || n.Buyer != "Engineering Services - Northeast Region" {
		t.Fatalf("notice text %+v", n)
	}
	if n.Location != "Northeastern, Ont., CAN" || n.Published != "2026/09/17" || n.Closing != "2026/10/22" || n.DaysRemaining != "34 day(s) left" {
		t.Fatalf("notice dates %+v", n)
	}
	p := page.Records[1]
	if p.InternalID != "4071906311" || p.Solicitation != "0000330846" || p.PortalPrefix != "" || p.Slug != "RFP-for-Corner-Brook-Fleet-Centre" || p.Buyer != "Medavie Inc." {
		t.Fatalf("plain %+v", p)
	}
}

func TestCeilingAndURL(t *testing.T) {
	if CeilingWarning(1000) != "" {
		t.Fatal("1000 is reachable")
	}
	for _, n := range []int{3254, 559393} {
		w := CeilingWarning(n)
		if !strings.Contains(w, strconv.Itoa(n)) || !strings.Contains(w, "1000") {
			t.Fatalf("warning %q", w)
		}
	}
	u, err := (Query{Status: "open", Page: 2, Keywords: "road", SortBy: "closingDate", SortDirection: "asc"}).URL()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"/public/solicitations/open?", "pageNumber=2", "keywords=road", "sortBy=closingDate", "sortDirection=ASC"} {
		if !strings.Contains(u, want) {
			t.Fatalf("url %q missing %q", u, want)
		}
	}
	if strings.Contains(u, "pageSize") {
		t.Fatal("must not send pageSize")
	}
	for _, q := range []Query{
		{Status: "won"}, {Status: "open", Page: 41}, {Status: "open", Page: 0},
		{Status: "open", Page: 1, SortBy: "title"}, {Status: "open", Page: 1, PublishDate: "TODAY"},
	} {
		if _, err := q.URL(); err == nil {
			t.Fatalf("accepted %+v", q)
		}
	}
	if _, err := Parse(strings.NewReader(`<tr class="mets-table-row"></tr>`)); err == nil {
		t.Fatal("missing total")
	}
	closed, err := Parse(strings.NewReader(`<span class="simpleSolResultsNumResults">559,393 results</span>`))
	if err != nil || closed.Total != 559393 || CeilingWarning(closed.Total) == "" {
		t.Fatalf("closed %+v %v", closed, err)
	}
}

func TestParseStripsControlBytes(t *testing.T) {
	html := "<span class=\"simpleSolResultsNumResults\">1 results</span>" +
		"<table><tbody><tr class=\"mets-table-row\"><td>" +
		"<a id=\"searchResultSol_notice_1\" href=\"/public/supplier/interception/view-notice/1\" class=\"solicitation-link\">" +
		"<span class=\"rowTitle\">Evil\x1b[2JTitle</span>" +
		"<span class=\"buyer-name\">Buyer\x1b]0;pwned\x07Name</span>" +
		"</a></td></tr></tbody></table>"
	page, err := Parse(strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	r := page.Records[0]
	if strings.ContainsAny(r.Title, "\x1b\x07") || strings.ContainsAny(r.Buyer, "\x1b\x07") {
		t.Fatalf("control bytes leaked into record: %+v", r)
	}
	if r.Title != "Evil [2JTitle" || r.Buyer != "Buyer ]0;pwned Name" {
		t.Fatalf("unexpected sanitized text: %+v", r)
	}
}
