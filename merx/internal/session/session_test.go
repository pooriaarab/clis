package session

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/html"

	"merx-cli/internal/httpx"
)

const (
	exe        = "e4s1"
	idpLogin   = `<html><body><form id="loginForm" action="/idp/SSO?execution=e4s1" method="post"><input type="hidden" name="serviceName" value="MERX"/><input name="j_username"/><input name="j_password"/><input type="hidden" name="_eventId_proceed" value="Proceed"/></form></body></html>`
	spAutoPost = `<html><body><form action="/idp/SSO" method="post"><input type="hidden" name="SAMLRequest" value="REQ123"/><input type="hidden" name="language" value="en"/></form></body></html>`
	samlResp   = `<html><body><form action="/saml/SSO/alias/MERX" method="post"><input type="hidden" name="SAMLResponse" value="RESP123"/><input type="hidden" name="RelayState" value="R"/></form></body></html>`
	homeAnon   = `<html><body><script>_trackMemberDataGA( {"memberType":"Anonymous"} );</script></body></html>`
	homeAuthed = `<html><body><script>_trackMemberDataGA( {"memberType":"Member"} );</script></body></html>`
	badPage    = `<html><body><p>The username or password you entered is incorrect.</p></body></html>`
	inUsePage  = `<html><body><p>The account provided is currently in use. Only one session is permitted per account.</p></body></html>`
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

func TestSaveDropsExpiredCookies(t *testing.T) {
	path := filepath.Join(t.TempDir(), JarFile)
	j, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	j.items = append(j.items,
		storedCookie{Name: "JSESSIONID", Value: "live", Domain: "www.merx.com", Path: "/"},
		storedCookie{Name: "stale", Value: "gone", Domain: "www.merx.com", Path: "/", Expires: time.Now().Add(-time.Hour)},
	)
	if err := j.Save(); err != nil {
		t.Fatal(err)
	}
	loaded, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.items) != 1 || loaded.items[0].Name != "JSESSIONID" {
		t.Fatalf("items after save = %+v, want only the live cookie", loaded.items)
	}
}

func TestSetCookiesRejectsForeignDomain(t *testing.T) {
	path := filepath.Join(t.TempDir(), JarFile)
	j, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	portal, err := url.Parse(Production.Portal + "/")
	if err != nil {
		t.Fatal(err)
	}
	j.SetCookies(portal, []*http.Cookie{{Name: "JSESSIONID", Value: "x", Path: "/", Domain: "evil.example"}})
	if len(j.items) != 0 {
		t.Fatalf("items = %+v, want the foreign-domain cookie rejected", j.items)
	}
}

func TestSetCookiesMaxAgeOnlyExpires(t *testing.T) {
	path := filepath.Join(t.TempDir(), JarFile)
	j, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	portal, err := url.Parse(Production.Portal + "/")
	if err != nil {
		t.Fatal(err)
	}
	j.SetCookies(portal, []*http.Cookie{{Name: "JSESSIONID", Value: "x", Path: "/", MaxAge: 60}})
	if len(j.items) != 1 {
		t.Fatalf("items = %+v, want one stored cookie", j.items)
	}
	want := time.Now().Add(60 * time.Second)
	if got := j.items[0].Expires; got.IsZero() || got.Before(want.Add(-5*time.Second)) || got.After(want.Add(5*time.Second)) {
		t.Fatalf("expires = %v, want ~%v derived from MaxAge", got, want)
	}
}

func TestSetCookiesMaxAgeOverridesStaleExpires(t *testing.T) {
	path := filepath.Join(t.TempDir(), JarFile)
	j, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	portal, err := url.Parse(Production.Portal + "/")
	if err != nil {
		t.Fatal(err)
	}
	// RFC 6265 §5.3: Max-Age takes precedence over Expires. A stale
	// Expires paired with a live Max-Age must not delete the cookie.
	j.SetCookies(portal, []*http.Cookie{{
		Name: "JSESSIONID", Value: "x", Path: "/",
		MaxAge:  60,
		Expires: time.Now().Add(-time.Hour),
	}})
	if len(j.items) != 1 {
		t.Fatalf("items = %+v, want the cookie kept per Max-Age", j.items)
	}
	want := time.Now().Add(60 * time.Second)
	if got := j.items[0].Expires; got.IsZero() || got.Before(want.Add(-5*time.Second)) || got.After(want.Add(5*time.Second)) {
		t.Fatalf("expires = %v, want ~%v derived from MaxAge, not the stale Expires", got, want)
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

func fixtureServer(user, pass, outcome string, credPosts, acsPosts, logoutHits *int, lastACS *url.Values) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/public/authentication/login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "JSESSIONID", Value: "T.METS", Path: "/"})
		http.Redirect(w, r, "/saml/login", http.StatusFound)
	})
	mux.HandleFunc("/saml/login", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, spAutoPost)
	})
	mux.HandleFunc("/idp/SSO", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Query().Get("execution") == exe {
			*credPosts++
			r.ParseForm()
			okCred := r.Form.Get("j_username") == user && r.Form.Get("j_password") == pass &&
				r.Form.Get("serviceName") == "MERX" && r.Form.Get("_eventId_proceed") != ""
			switch {
			case outcome == "bad302":
				http.Redirect(w, r, "/idp/bad", http.StatusFound)
			case outcome == "inuse":
				io.WriteString(w, inUsePage)
			case okCred:
				io.WriteString(w, samlResp)
			default:
				io.WriteString(w, badPage)
			}
			return
		}
		io.WriteString(w, idpLogin)
	})
	mux.HandleFunc("/idp/bad", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, badPage)
	})
	mux.HandleFunc("/saml/SSO/alias/MERX", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		*acsPosts++
		*lastACS = r.Form
		if r.Form.Get("SAMLResponse") == "" {
			http.Error(w, "missing SAMLResponse", http.StatusBadRequest)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "MERXSESSION", Value: "1", Path: "/"})
	})
	mux.HandleFunc("/public/authentication/logout", func(w http.ResponseWriter, r *http.Request) {
		*logoutHits++
		http.Redirect(w, r, "/saml/logout", http.StatusFound)
	})
	mux.HandleFunc("/saml/logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "MERXSESSION", MaxAge: -1, Path: "/"})
		http.Redirect(w, r, "/", http.StatusFound)
	})
	mux.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "wrong logout path", http.StatusNotFound)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("MERXSESSION"); err == nil && c.Value == "1" {
			io.WriteString(w, homeAuthed)
			return
		}
		io.WriteString(w, homeAnon)
	})
	return httptest.NewServer(mux)
}

func testClient(j *Jar) *httpx.Client {
	c := &httpx.Client{HTTP: &http.Client{}, Delay: time.Millisecond}
	c.HTTP.Jar = j
	c.HTTP.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return c
}

func TestLoginOutcomes(t *testing.T) {
	const user, pass = "u", "p"
	cases := []struct {
		name, pass, outcome, errSub string
		wantACS                     int
	}{
		{"success posts SAMLResponse to ACS", pass, "ok", "", 1},
		{"302 then incorrect is bad credentials", "wrong", "bad302", "bad credentials", 0},
		{"currently in use is session conflict", pass, "inuse", "currently in use", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var credPosts, acsPosts, logoutHits int
			var lastACS url.Values
			srv := fixtureServer(user, pass, tc.outcome, &credPosts, &acsPosts, &logoutHits, &lastACS)
			t.Cleanup(srv.Close)
			ep := Endpoints{Portal: srv.URL, IDP: srv.URL}
			j, err := Open(filepath.Join(t.TempDir(), JarFile))
			if err != nil {
				t.Fatal(err)
			}
			err = Login(testClient(j), ep, user, tc.pass)
			if credPosts != 1 {
				t.Fatalf("credential attempts %d, want 1", credPosts)
			}
			if acsPosts != tc.wantACS {
				t.Fatalf("ACS posts %d, want %d", acsPosts, tc.wantACS)
			}
			if tc.errSub == "" {
				if err != nil {
					t.Fatal(err)
				}
				if lastACS.Get("SAMLResponse") != "RESP123" || lastACS.Get("RelayState") != "R" {
					t.Fatalf("ACS form %v", lastACS)
				}
				ok, err := Valid(testClient(j), ep.Portal)
				if err != nil || !ok {
					t.Fatalf("session valid=%v err=%v", ok, err)
				}
				if err := Logout(testClient(j), ep.Portal); err != nil || logoutHits != 1 {
					t.Fatalf("logout err=%v hits=%d", err, logoutHits)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.errSub) {
				t.Fatalf("error %v, want %q", err, tc.errSub)
			}
			if tc.outcome == "inuse" && !strings.Contains(err.Error(), "merx logout") {
				t.Fatalf("conflict error must mention merx logout: %v", err)
			}
		})
	}
}

func seedSession(t *testing.T, portal string, authed bool) *Jar {
	t.Helper()
	j, err := Open(filepath.Join(t.TempDir(), JarFile))
	if err != nil {
		t.Fatal(err)
	}
	if !authed {
		return j
	}
	u, err := url.Parse(portal + "/")
	if err != nil {
		t.Fatal(err)
	}
	j.SetCookies(u, []*http.Cookie{{Name: "MERXSESSION", Value: "1", Path: "/"}})
	return j
}

func TestLogoutTargetsPublicAuthenticationPath(t *testing.T) {
	var right, wrong int
	mux := http.NewServeMux()
	mux.HandleFunc("/public/authentication/logout", func(w http.ResponseWriter, r *http.Request) {
		right++
		http.SetCookie(w, &http.Cookie{Name: "MERXSESSION", MaxAge: -1, Path: "/"})
		http.Redirect(w, r, "/", http.StatusFound)
	})
	mux.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		wrong++
		http.Error(w, "old path", http.StatusNotFound)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("MERXSESSION"); err == nil && c.Value == "1" {
			io.WriteString(w, homeAuthed)
			return
		}
		io.WriteString(w, homeAnon)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	if err := Logout(testClient(seedSession(t, srv.URL, true)), srv.URL); err != nil {
		t.Fatal(err)
	}
	if right != 1 {
		t.Fatalf("hits on /public/authentication/logout = %d, want 1", right)
	}
	if wrong != 0 {
		t.Fatalf("hits on /logout = %d, want 0", wrong)
	}
}

func TestLogoutFollowsRedirectChainAndSAMLForms(t *testing.T) {
	var hops []string
	var samlReq, samlResp string
	mux := http.NewServeMux()
	mux.HandleFunc("/public/authentication/logout", func(w http.ResponseWriter, r *http.Request) {
		hops = append(hops, r.Method+" "+r.URL.Path)
		http.Redirect(w, r, "/saml/logout", http.StatusFound)
	})
	mux.HandleFunc("/saml/logout", func(w http.ResponseWriter, r *http.Request) {
		hops = append(hops, r.Method+" "+r.URL.Path)
		io.WriteString(w, `<html><body><form action="/idp/SLO" method="post"><input type="hidden" name="SAMLRequest" value="LOREQ"/><input type="hidden" name="RelayState" value="out"/></form></body></html>`)
	})
	mux.HandleFunc("/idp/SLO", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		samlReq = r.Form.Get("SAMLRequest")
		hops = append(hops, r.Method+" "+r.URL.Path)
		io.WriteString(w, `<html><body><form action="/saml/SingleLogout" method="post"><input type="hidden" name="SAMLResponse" value="LORESP"/><input type="hidden" name="RelayState" value="out"/></form></body></html>`)
	})
	mux.HandleFunc("/saml/SingleLogout", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		samlResp = r.Form.Get("SAMLResponse")
		hops = append(hops, r.Method+" "+r.URL.Path)
		http.SetCookie(w, &http.Cookie{Name: "MERXSESSION", MaxAge: -1, Path: "/"})
		http.Redirect(w, r, "/", http.StatusFound)
	})
	mux.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("old /logout path was requested")
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("MERXSESSION"); err == nil && c.Value == "1" {
			io.WriteString(w, homeAuthed)
			return
		}
		io.WriteString(w, homeAnon)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	if err := Logout(testClient(seedSession(t, srv.URL, true)), srv.URL); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"GET /public/authentication/logout",
		"GET /saml/logout",
		"POST /idp/SLO",
		"POST /saml/SingleLogout",
	}
	if strings.Join(hops, ",") != strings.Join(want, ",") {
		t.Fatalf("hops %v, want %v", hops, want)
	}
	if samlReq != "LOREQ" || samlResp != "LORESP" {
		t.Fatalf("SAMLRequest=%q SAMLResponse=%q", samlReq, samlResp)
	}
}

func TestLogoutWhenNotLoggedInIsNoop(t *testing.T) {
	var logoutHits int
	mux := http.NewServeMux()
	mux.HandleFunc("/public/authentication/logout", func(w http.ResponseWriter, r *http.Request) {
		logoutHits++
		http.Error(w, "should not be called when already anonymous", http.StatusInternalServerError)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, homeAnon)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	if err := Logout(testClient(seedSession(t, srv.URL, false)), srv.URL); err != nil {
		t.Fatalf("logout when not logged in must not error: %v", err)
	}
	if logoutHits != 0 {
		t.Fatalf("noop logout hit the logout path %d times", logoutHits)
	}
}

func TestLogoutDoesNotClaimSuccessWhenSessionRemains(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/public/authentication/logout", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/saml/logout", http.StatusFound)
	})
	mux.HandleFunc("/saml/logout", func(w http.ResponseWriter, r *http.Request) {
		// Redirects complete, but the session cookie is left in place.
		http.Redirect(w, r, "/", http.StatusFound)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, homeAuthed)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	err := Logout(testClient(seedSession(t, srv.URL, true)), srv.URL)
	if err == nil {
		t.Fatal("Logout returned success while the session still looks authenticated")
	}
	if strings.Contains(err.Error(), "Logged out.") {
		t.Fatalf("must not report plain success: %v", err)
	}
	if !strings.Contains(err.Error(), "authenticated") {
		t.Fatalf("error should say the session remains: %v", err)
	}
}
