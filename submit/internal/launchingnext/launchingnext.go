// Package launchingnext is the ONE scriptable submission target: a plain form
// POST plus a static math check. Never add other auto-submit targets.
package launchingnext

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DefaultBase is the real site. SUBMIT_LAUNCHINGNEXT_BASE overrides it
// (tests only); the gap floor below only applies to the real site.
const DefaultBase = "https://www.launchingnext.com"

// SubmitPath is the form page, which also accepts the POST (action="").
const SubmitPath = "/submit/"

// MinGap is the minimum delay between real submissions.
const MinGap = 25 * time.Second

// BaseURL resolves the submit host. The second return is true when the
// test override is in effect.
func BaseURL(getenv func(string) string) (string, bool) {
	if b := strings.TrimSuffix(getenv("SUBMIT_LAUNCHINGNEXT_BASE"), "/"); b != "" {
		return b, true
	}
	return DefaultBase, false
}

// RequiredFields is the form shape verified on 2026-10-03. A submit aborts
// when any of these is missing: the form changed and the POST would be wrong.
var RequiredFields = []string{"startupname", "startupurl", "description", "fulldescription", "tags",
	"funding", "marketing_budget", "user", "email", "math", "formSubmit"}

var (
	formRe = regexp.MustCompile(`(?is)<form\b[^>]*>.*?</form>`)
	mathRe = regexp.MustCompile(`(?i)What is\s+(\d+)\s*([+\-x×*/÷]|–|—)\s*(\d+)\s*\?`)
)

// Form is a parsed submit form.
type Form struct {
	// Question is the math question text, e.g. "What is 2+3?".
	Question string
	// Answer is the computed math answer, sent as the math field.
	Answer string
}

// Fetch downloads the submit page and returns its HTML.
func Fetch(base string) (string, error) {
	client := &http.Client{Timeout: 25 * time.Second}
	req, err := http.NewRequest("GET", base+SubmitPath, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; submit-cli/1.0)")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch submit form %s: %w", base+SubmitPath, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("fetch submit form %s: status %d", base+SubmitPath, resp.StatusCode)
	}
	return string(body), nil
}

// Parse extracts the submit form and solves its math check. It fails when
// the form shape changed, so a site redesign never sends a blind POST.
func Parse(page string) (Form, error) {
	var f Form
	var candidates []string
	for _, m := range formRe.FindAllString(page, -1) {
		if strings.Contains(m, `name="startupname"`) || strings.Contains(m, `name='startupname'`) {
			candidates = append(candidates, m)
		}
	}
	if len(candidates) != 1 {
		return f, fmt.Errorf("form shape changed: found %d submit form(s), want 1; aborting", len(candidates))
	}
	form := candidates[0]
	var missing []string
	for _, name := range RequiredFields {
		if !strings.Contains(form, `name="`+name+`"`) && !strings.Contains(form, `name='`+name+`'`) {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return f, fmt.Errorf("form shape changed: missing field(s) %s; aborting", strings.Join(missing, ", "))
	}
	m := mathRe.FindStringSubmatch(form)
	if m == nil {
		return f, fmt.Errorf("form shape changed: math question not found; aborting")
	}
	a, _ := strconv.Atoi(m[1])
	b, _ := strconv.Atoi(m[3])
	var ans int
	switch m[2] {
	case "+":
		ans = a + b
	case "-", "–", "—":
		ans = a - b
	case "x", "X", "×", "*":
		ans = a * b
	case "/", "÷":
		if b == 0 {
			return f, fmt.Errorf("form shape changed: division by zero in math check; aborting")
		}
		ans = a / b
	default:
		return f, fmt.Errorf("form shape changed: unknown math operator %q; aborting", m[2])
	}
	f.Question = m[0]
	f.Answer = strconv.Itoa(ans)
	return f, nil
}

// Input is one site's submission values.
type Input struct {
	Name, URL, Headline, Full, Tags, Funding, Marketing, Founder, Email string
}

// FundingValue maps the sites YAML funding name to the form radio value.
func FundingValue(name string) string {
	switch name {
	case "side":
		return "1"
	case "bootstrapped":
		return "2"
	case "funded":
		return "3"
	default:
		return "0"
	}
}

// Body builds the exact POST body for one site. newsletter_optin is always
// left out: the CLI never opts the team inbox into marketing mail.
func Body(in Input, mathAnswer string) url.Values {
	v := url.Values{}
	v.Set("startupname", in.Name)
	v.Set("startupurl", in.URL)
	v.Set("description", in.Headline)
	v.Set("fulldescription", in.Full)
	v.Set("tags", in.Tags)
	v.Set("funding", FundingValue(in.Funding))
	v.Set("marketing_budget", in.Marketing)
	v.Set("user", in.Founder)
	v.Set("email", in.Email)
	v.Set("math", mathAnswer)
	v.Set("formSubmit", "Submit Startup")
	return v
}

// Post submits one body to the form endpoint.
func Post(base string, body url.Values) (int, string, error) {
	client := &http.Client{Timeout: 25 * time.Second}
	req, err := http.NewRequest("POST", base+SubmitPath, strings.NewReader(body.Encode()))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; submit-cli/1.0)")
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("post submission: %w", err)
	}
	defer resp.Body.Close()
	text, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return resp.StatusCode, "", err
	}
	return resp.StatusCode, string(text), nil
}

var successMarkers = []string{"thank you", "thanks for", "successfully submitted",
	"has been received", "have been received", "under review", "we will review"}

// LooksLikeSuccess is a heuristic: the response reads like an acceptance and
// does not redisplay the submit form (which signals field errors).
func LooksLikeSuccess(page string) bool {
	lower := strings.ToLower(page)
	if strings.Contains(lower, `name="startupname"`) {
		return false
	}
	for _, m := range successMarkers {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}
