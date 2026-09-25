package search

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestParsePrivateFixture(t *testing.T) {
	f, err := os.Open("testdata/private_list.golden")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	page, err := ParsePrivate(f)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 3332 || len(page.Records) != 3 {
		t.Fatalf("total %d rows %d", page.Total, len(page.Records))
	}
	n := page.Records[0]
	if n.InternalID != "4103112608" {
		t.Fatalf("internal id must be the href's last segment, got %q", n.InternalID)
	}
	if n.Title != "Division Office and Conference Centre Needs Assessment and Feasibility Study" || n.Buyer != "Interlake School Division" {
		t.Fatalf("title/buyer %+v", n)
	}
	if n.Location != "Interlake, MB, CAN" || n.Published != "2026/09/24" || n.Closing != "2026/10/09 01:00:00 PM EDT" {
		t.Fatalf("dates %+v", n)
	}
	if n.Description != "" || n.DetailURL != "https://www.merx.com/private/supplier/interception/open-solicitation/4103112608" {
		t.Fatalf("detail %+v", n)
	}
	if strings.Contains(n.DetailURL, "target=view") {
		t.Fatalf("query leaked into detail url %q", n.DetailURL)
	}
	hidden := page.Records[1]
	if hidden.InternalID != "444173036954" || strings.Contains(hidden.Title, "opens in a new window") {
		t.Fatalf("hidden title suffix leaked: %+v", hidden)
	}
	if hidden.Title != "Supply and Delivery of ZF AV133 Axles for Various Metrolinx Bus Facilities" || hidden.Buyer != "Metrolinx" || hidden.Location != "Southern Ontario, CAN" {
		t.Fatalf("notice row %+v", hidden)
	}
	d := page.Records[2]
	if d.InternalID != "444172969960" || d.Buyer != "City of Regina" || d.Location != "Saskatchewan, CAN" || d.Published != "2026/09/24" || d.Closing != "2026/10/15 03:00:00 PM EDT" {
		t.Fatalf("described row %+v", d)
	}
	if d.Description != "The City is seeking qualified contractors to provide equipment, operators and associated resources to undertake winter sidewalk maintenance services in support of accessibility objectives." {
		t.Fatalf("description %q", d.Description)
	}
	if d.DetailURL != "https://www.merx.com/private/supplier/interception/view-notice/444172969960" {
		t.Fatalf("notice url %q", d.DetailURL)
	}
}

func TestParsePrivateMissingTitleLink(t *testing.T) {
	const html = `<span class="mets-total-elements-display">1 - 1 of 1 results found</span>` +
		`<table><tr class="mets-table-row"><td><span class="buyerIdentification">Buyer</span></td></tr></table>`
	page, err := ParsePrivate(strings.NewReader(html))
	if err == nil || !strings.Contains(err.Error(), "solicitationsTitleLink") {
		t.Fatalf("missing title link must be an error, page %+v err %v", page, err)
	}
	if len(page.Records) != 0 {
		t.Fatalf("error returned records %+v", page.Records)
	}
}

func TestParsePrivateIDIsHrefSegment(t *testing.T) {
	const html = `<span class="mets-total-elements-display">1 - 1 of 1 results found</span>` +
		`<table><tr class="mets-table-row"><td><a id="title1" href="/private/supplier/interception/open-solicitation/4103112608?target=view" class="solicitationsTitleLink">T</a></td></tr></table>`
	page, err := ParsePrivate(strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	if page.Records[0].InternalID != "4103112608" {
		t.Fatalf("id %q", page.Records[0].InternalID)
	}
}

func TestPublicParseRejectsPrivateFixture(t *testing.T) {
	f, err := os.Open("testdata/private_list.golden")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := Parse(f); err == nil {
		t.Fatal("public Parse accepted an authenticated results row")
	}
}

func TestPublicRecordJSONOmitsDescription(t *testing.T) {
	raw, err := json.Marshal(Record{InternalID: "1", Title: "T"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "description") {
		t.Fatalf("public record JSON changed: %s", raw)
	}
	with, err := json.Marshal(Record{InternalID: "1", Description: "body"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(with), `"description":"body"`) {
		t.Fatalf("description dropped: %s", with)
	}
}
