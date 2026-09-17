package session

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

const (
	exe        = "e4s1"
	idpLogin   = `<html><body><form id="loginForm" action="/idp/SSO?execution=e4s1" method="post"><input type="hidden" name="serviceName" value="MERX"/><input name="j_username"/><input name="j_password"/><input type="hidden" name="_eventId_proceed" value="Proceed"/></form></body></html>`
	spAutoPost = `<html><body><form action="/idp/SSO" method="post"><input type="hidden" name="SAMLRequest" value="REQ123"/><input type="hidden" name="language" value="en"/></form></body></html>`
)

func TestFindForm(t *testing.T) {
	doc, err := html.Parse(strings.NewReader(spAutoPost))
	if err != nil {
		t.Fatal(err)
	}
	action, fields, ok := FindForm(doc, "SAMLRequest")
	if !ok || action != "/idp/SSO" || fields["SAMLRequest"] != "REQ123" || fields["language"] != "en" {
		t.Fatalf("form action=%q fields=%v ok=%v", action, fields, ok)
	}
}

func TestExecutionTokenFromE4s1(t *testing.T) {
	doc, err := html.Parse(strings.NewReader(idpLogin))
	if err != nil {
		t.Fatal(err)
	}
	action, fields, ok := FindForm(doc, "j_username", "j_password")
	if !ok || fields["serviceName"] != "MERX" || fields["_eventId_proceed"] == "" {
		t.Fatalf("form %v ok=%v", fields, ok)
	}
	tok, err := ExecutionToken(action)
	if err != nil || tok != exe {
		t.Fatalf("token %q err=%v", tok, err)
	}
	if _, err := ExecutionToken("/idp/SSO"); err == nil {
		t.Fatal("want error for missing execution token")
	}
}

func TestJarFilePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), JarFile)
	j, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	j.SetCookies(&url.URL{Scheme: "https", Host: "www.merx.com", Path: "/"}, []*http.Cookie{{Name: "JSESSIONID", Value: "x", Path: "/"}})
	if err := j.Save(); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("jar mode %o, want 600", fi.Mode().Perm())
	}
}

func TestJarRoundTripBothHosts(t *testing.T) {
	path := filepath.Join(t.TempDir(), JarFile)
	j, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	portal, err := url.Parse(Production.Portal + "/")
	if err != nil {
		t.Fatal(err)
	}
	idp, err := url.Parse(Production.IDP + "/")
	if err != nil {
		t.Fatal(err)
	}
	j.SetCookies(portal, []*http.Cookie{{Name: "JSESSIONID", Value: "portal", Path: "/"}})
	j.SetCookies(idp, []*http.Cookie{{Name: "JSESSIONID", Value: "idp", Path: "/"}})
	if err := j.Save(); err != nil {
		t.Fatal(err)
	}
	loaded, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	gotPortal := cookieValue(loaded.Cookies(portal), "JSESSIONID")
	gotIDP := cookieValue(loaded.Cookies(idp), "JSESSIONID")
	if gotPortal != "portal" || gotIDP != "idp" {
		t.Fatalf("portal=%q idp=%q", gotPortal, gotIDP)
	}
}

func cookieValue(cs []*http.Cookie, name string) string {
	for _, c := range cs {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}
