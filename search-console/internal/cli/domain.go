package cli

import (
	"fmt"
	"regexp"
	"strings"
)

var domainRE = regexp.MustCompile(`^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z][a-z0-9-]{1,62}$`)

// normalizeDomain lower-cases a bare domain name and drops a trailing dot. A URL,
// a path, a port or a wildcard is a usage error.
func normalizeDomain(s string) (string, error) {
	d := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(s)), ".")
	if !domainRE.MatchString(d) {
		return "", &ExitError{Code: ExitUsage, Err: fmt.Errorf("%q is not a bare domain name like example.com", s)}
	}
	return d, nil
}
