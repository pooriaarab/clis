package walmart

import (
	"strings"
	"testing"
)

func TestParseCurl(t *testing.T) {
	raw := `curl 'https://www.walmart.ca/orchestra/cph/graphql/PurchaseHistoryV2/d15c6dbb75db24eebe6462584af35b618a320e531f9c051913b93fc59e40e94f?variables=x' \
  -H 'cookie: CID=a; SPID=b; auth=c' -H 'x-o-bu: WALMART-CA'`
	s, err := ParseCurl(raw)
	if err != nil {
		t.Fatal(err)
	}
	if s.Cookies["auth"] != "c" || len(s.Cookies) != 3 {
		t.Fatalf("parse mismatch: %+v", s)
	}
	if s.Hash != DefaultHistoryHash {
		t.Fatal("query hash not captured")
	}
	if _, err := ParseCurl("curl 'https://example.com/' -H 'cookie: CID=a; SPID=b; auth=c'"); err == nil {
		t.Fatal("expected rejection of non-walmart URL")
	}
	if _, err := ParseCurl("curl 'https://www.walmart.ca/' -H 'cookie: CID=a'"); err == nil {
		t.Fatal("expected rejection when auth cookie missing")
	}
}

func TestOrderGroupString(t *testing.T) {
	g := OrderGroup{DisplayID: "42", ItemCount: 3, DeliveryMessage: "Delivered"}
	s := g.String()
	if !strings.Contains(s, "42") || !strings.Contains(s, "3 items") {
		t.Fatal("bad summary: " + s)
	}
}
