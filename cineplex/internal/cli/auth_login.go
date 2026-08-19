// hand-authored novel command; preserve across regenerate

package cli

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha1"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"cineplex-pp-cli/internal/config"
	"github.com/spf13/cobra"

	_ "modernc.org/sqlite"
)

// newAuthLoginCmd imports the SCENE+ session cookie straight from a logged-in
// Chrome profile, so users don't have to hand-copy it. macOS only for now
// (Chrome's cookie store is decrypted with a Keychain key). It writes the same
// cookie `auth set-cookie` does. Fails loudly — it never stores a partial or
// wrong cookie silently.
func newAuthLoginCmd(flags *rootFlags) *cobra.Command {
	var profile string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Import your SCENE+ session cookie from a logged-in Chrome",
		Long: strings.Trim(`
Reads the cineplex.com cookies from your logged-in Chrome profile and stores
them as your SCENE+ session cookie, so authed ticketing commands (seat
availability, cart, reserve) work without hand-copying anything.

Requirements: macOS, Google Chrome, and an active cineplex.com login in the
chosen profile. Chrome must allow Keychain access when prompted. If your Chrome
uses app-bound cookie encryption (recent versions), this cannot decrypt them —
use 'auth set-cookie <cookie>' with a value copied from DevTools instead.
`, "\n"),
		Annotations: map[string]string{"mcp:local-write": "true", "pp:sensitive": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.OutOrStdout(), "would import cineplex cookies from Chrome profile:", profile)
				return nil
			}
			if runtime.GOOS != "darwin" {
				return fmt.Errorf("auth login (Chrome import) is macOS-only; on this OS use: %s auth set-cookie <cookie>", rootCommandName(cmd))
			}
			cookie, err := importChromeCineplexCookie(profile)
			if err != nil {
				return fmt.Errorf("%w\nfallback: copy the Cookie header from any apis.cineplex.com request in DevTools and run:\n  %s auth set-cookie <cookie>", err, rootCommandName(cmd))
			}
			cfg, err := config.Load(flags.configPath)
			if err != nil {
				return configErr(err)
			}
			if err := config.SaveSceneSessionCookie(cfg.Path, cookie); err != nil {
				return configErr(fmt.Errorf("saving session cookie: %w", err))
			}
			names := cookieNames(cookie)
			if flags.asJSON {
				return printJSONFiltered(cmd.OutOrStdout(), map[string]any{"saved": true, "config_path": cfg.Path, "cookies": names}, flags)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Imported %d cineplex cookie(s) from Chrome (%s) and saved to %s\n", len(names), profile, cfg.Path)
			return nil
		},
	}
	cmd.Flags().StringVar(&profile, "profile", "Default", "Chrome profile directory name")
	return cmd
}

func rootCommandName(cmd *cobra.Command) string {
	r := cmd.Root()
	if r != nil {
		return r.Name()
	}
	return "cineplex-pp-cli"
}

// importChromeCineplexCookie reads + decrypts cineplex cookies from Chrome and
// returns a Cookie header string ("name=value; name2=value2").
func importChromeCineplexCookie(profile string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	cookiesDB := filepath.Join(home, "Library", "Application Support", "Google", "Chrome", profile, "Cookies")
	if _, err := os.Stat(cookiesDB); err != nil {
		return "", fmt.Errorf("Chrome cookies not found for profile %q (%s)", profile, cookiesDB)
	}

	key, err := chromeAESKey()
	if err != nil {
		return "", err
	}

	// Chrome holds the DB open with a lock; work on a copy.
	tmp, err := os.CreateTemp("", "cpx-cookies-*.db")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	tmp.Close()
	src, err := os.ReadFile(cookiesDB) // #nosec G304 -- fixed Chrome path.
	if err != nil {
		return "", fmt.Errorf("reading Chrome cookies (is Chrome open? try quitting it): %w", err)
	}
	if err := os.WriteFile(tmp.Name(), src, 0o600); err != nil {
		return "", err
	}

	db, err := sql.Open("sqlite", tmp.Name()+"?mode=ro&immutable=1")
	if err != nil {
		return "", err
	}
	defer db.Close()
	rows, err := db.Query(`SELECT host_key, name, encrypted_value FROM cookies WHERE host_key LIKE '%cineplex.com' ORDER BY host_key`)
	if err != nil {
		return "", fmt.Errorf("querying cookies: %w", err)
	}
	defer rows.Close()

	pairs := map[string]string{}
	appBound := false
	for rows.Next() {
		var host, name string
		var enc []byte
		if err := rows.Scan(&host, &name, &enc); err != nil {
			continue
		}
		val, err := decryptChromeCookie(enc, key)
		if err != nil {
			if strings.Contains(err.Error(), "v20") {
				appBound = true
			}
			continue
		}
		if val != "" {
			pairs[name] = val
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(pairs) == 0 {
		if appBound {
			return "", fmt.Errorf("Chrome cookies use app-bound encryption (v20) and can't be decrypted outside Chrome")
		}
		return "", fmt.Errorf("no cineplex cookies found — log in to cineplex.com in Chrome profile %q first", profile)
	}
	names := make([]string, 0, len(pairs))
	for n := range pairs {
		names = append(names, n)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(pairs))
	for _, n := range names {
		parts = append(parts, n+"="+pairs[n])
	}
	return strings.Join(parts, "; "), nil
}

// chromeAESKey derives the AES-128 key Chrome uses on macOS: PBKDF2-HMAC-SHA1
// of the "Chrome Safe Storage" Keychain password, salt "saltysalt", 1003 iters.
func chromeAESKey() ([]byte, error) {
	out, err := exec.Command("security", "find-generic-password", "-w", "-s", "Chrome Safe Storage").Output() // #nosec G204 -- fixed args.
	if err != nil {
		return nil, fmt.Errorf("reading Chrome Keychain key (approve the Keychain prompt): %w", err)
	}
	pass := strings.TrimRight(string(out), "\n")
	if pass == "" {
		return nil, fmt.Errorf("empty Chrome Safe Storage key")
	}
	return pbkdf2SHA1([]byte(pass), []byte("saltysalt"), 1003, 16), nil
}

// decryptChromeCookie decrypts a v10 AES-128-CBC cookie value. Rejects v20
// (app-bound) explicitly. Strips the 32-byte SHA256 domain prefix newer Chrome
// prepends to the plaintext when present.
func decryptChromeCookie(enc, key []byte) (string, error) {
	if len(enc) < 3 {
		return "", fmt.Errorf("short value")
	}
	prefix := string(enc[:3])
	if prefix == "v20" {
		return "", fmt.Errorf("v20 app-bound cookie")
	}
	if prefix != "v10" && prefix != "v11" {
		// Unencrypted (rare on macOS) — return as-is.
		return string(enc), nil
	}
	ct := enc[3:]
	if len(ct) == 0 || len(ct)%aes.BlockSize != 0 {
		return "", fmt.Errorf("bad ciphertext length")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	iv := make([]byte, aes.BlockSize)
	for i := range iv {
		iv[i] = ' '
	}
	pt := make([]byte, len(ct))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(pt, ct)
	pt, err = pkcs7Unpad(pt, aes.BlockSize)
	if err != nil {
		return "", err
	}
	// Newer Chrome prepends a 32-byte SHA256(domain) hash to the plaintext.
	if len(pt) > 32 && !isPrintable(pt[:32]) {
		pt = pt[32:]
	}
	return string(pt), nil
}

func isPrintable(b []byte) bool {
	for _, c := range b {
		if c != '\t' && (c < 0x20 || c > 0x7e) {
			return false
		}
	}
	return true
}

func pkcs7Unpad(b []byte, blockSize int) ([]byte, error) {
	if len(b) == 0 || len(b)%blockSize != 0 {
		return nil, fmt.Errorf("invalid padding")
	}
	n := int(b[len(b)-1])
	if n == 0 || n > blockSize || n > len(b) {
		return nil, fmt.Errorf("invalid padding")
	}
	for _, c := range b[len(b)-n:] {
		if int(c) != n {
			return nil, fmt.Errorf("invalid padding")
		}
	}
	return b[:len(b)-n], nil
}

// pbkdf2SHA1 is a minimal RFC 2898 PBKDF2 with HMAC-SHA1 (avoids adding x/crypto).
func pbkdf2SHA1(password, salt []byte, iter, keyLen int) []byte {
	prf := hmac.New(sha1.New, password)
	hashLen := prf.Size()
	numBlocks := (keyLen + hashLen - 1) / hashLen
	var dk []byte
	buf := make([]byte, 4)
	for block := 1; block <= numBlocks; block++ {
		prf.Reset()
		prf.Write(salt)
		buf[0] = byte(block >> 24)
		buf[1] = byte(block >> 16)
		buf[2] = byte(block >> 8)
		buf[3] = byte(block)
		prf.Write(buf)
		u := prf.Sum(nil)
		t := make([]byte, len(u))
		copy(t, u)
		for n := 2; n <= iter; n++ {
			prf.Reset()
			prf.Write(u)
			u = prf.Sum(nil)
			for i := range t {
				t[i] ^= u[i]
			}
		}
		dk = append(dk, t...)
	}
	return dk[:keyLen]
}

func cookieNames(cookie string) []string {
	var out []string
	for _, part := range strings.Split(cookie, ";") {
		if i := strings.IndexByte(part, '='); i > 0 {
			out = append(out, strings.TrimSpace(part[:i]))
		}
	}
	return out
}
