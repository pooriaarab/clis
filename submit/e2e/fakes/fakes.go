// Package fakes serves local stand-ins for third-party pages. Tests point the
// binary at these servers, so no test touches the real network.
package fakes

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
)

// LaunchingNext serves a form shaped like the real submit page: every field
// the CLI requires plus a "What is A+B?" math check. Mode changes the shape:
//
//	"ok"      normal form, math "What is 2+3?", POST returns a thank-you page
//	"changed" form without the math field (redesign simulation)
type LaunchingNext struct {
	Server *httptest.Server
	Mode   string

	mu    sync.Mutex
	Gets  int
	Posts []url.Values
}

func NewLaunchingNext(mode string) *LaunchingNext {
	ln := &LaunchingNext{Mode: mode}
	mux := http.NewServeMux()
	mux.HandleFunc("/submit/", func(w http.ResponseWriter, r *http.Request) {
		ln.mu.Lock()
		defer ln.mu.Unlock()
		if r.Method == "POST" {
			_ = r.ParseForm()
			ln.Posts = append(ln.Posts, r.PostForm)
			fmt.Fprint(w, `<html><body><h1>Thank you!</h1><p>Your startup has been received and is under review.</p></body></html>`)
			return
		}
		ln.Gets++
		math := `<p><label>Quick Check: What is 2+3?</label><br/><input type="text" name="math" /></p>`
		if ln.Mode == "changed" {
			math = `<p><label>Prove you are human (new widget)</label></p>`
		}
		fmt.Fprintf(w, `<html><body><form action="" method="post">
<input type="text" name="startupname" />
<input type="text" name="startupurl" />
<input type="text" name="description" />
<textarea name="fulldescription"></textarea>
<textarea name="tags"></textarea>
<input type="radio" name="funding" value="1" />
<input type="radio" name="funding" value="0" />
<input type="radio" name="marketing_budget" value="$0" />
<input type="text" name="user" />
<input type="text" name="email" />
%s
<input type="submit" name="formSubmit" value="Submit Startup" />
</form></body></html>`, math)
	})
	ln.Server = httptest.NewServer(mux)
	return ln
}

// PostCount returns how many POSTs the fake received.
func (ln *LaunchingNext) PostCount() int {
	ln.mu.Lock()
	defer ln.mu.Unlock()
	return len(ln.Posts)
}

// FirstPost returns the first POST body, or nil.
func (ln *LaunchingNext) FirstPost() url.Values {
	ln.mu.Lock()
	defer ln.mu.Unlock()
	if len(ln.Posts) == 0 {
		return nil
	}
	return ln.Posts[0]
}

// Listing serves listing pages for `check`:
//
//	/follow    links to the site with a plain anchor
//	/nofollow  links to the site with rel="nofollow"
//	/missing   no link to the site
//	/redir     302 to /follow
type Listing struct {
	Server  *httptest.Server
	SiteURL string
}

func NewListing(siteURL string) *Listing {
	l := &Listing{SiteURL: siteURL}
	mux := http.NewServeMux()
	mux.HandleFunc("/follow", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<html><body><a href="%s">visit site</a></body></html>`, siteURL)
	})
	mux.HandleFunc("/nofollow", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<html><body><a href="%s" rel="nofollow noopener">visit site</a></body></html>`, siteURL)
	})
	mux.HandleFunc("/missing", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><a href="https://other.example.com">other</a></body></html>`)
	})
	mux.HandleFunc("/redir", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/follow", http.StatusFound)
	})
	l.Server = httptest.NewServer(mux)
	return l
}

// URL joins path onto the fake listing server.
func (l *Listing) URL(path string) string {
	return strings.TrimSuffix(l.Server.URL, "/") + path
}
