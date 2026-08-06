package cli

// buzz media / buzz upload — Blossom media storage (BUD-01 get, BUD-02
// upload). Auth is a self-signed kind:24242 event (KIND_BLOSSOM_AUTH,
// buzz-core/src/kind.rs), not NIP-98. Mirrors
// /Users/parab/code/buzz/crates/buzz-cli/src/client.rs (sign_blossom_get,
// sign_blossom_upload, media_url_from_input, upload_file, download_media)
// and crates/buzz-cli/src/commands/upload.rs (dispatch, dispatch_media).

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"buzz-cli/internal/nostr"
	"github.com/spf13/cobra"
)

func mediaCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "media", Short: "Media commands"}

	get := &cobra.Command{
		Use:   "get <input>",
		Short: "Download relay media with Blossom get auth",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			output, _ := cmd.Flags().GetString("output")
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			if resolved.RelayURL == "" {
				return inputError("relay URL is required")
			}
			mediaURL, err := mediaURLFromInput(resolved.RelayURL, args[0])
			if err != nil {
				return err
			}
			auth, err := signBlossomGet(keys, mediaURL)
			if err != nil {
				return otherWrap("sign blossom get auth", err)
			}
			relayClient, err := restClientFromResolved(resolved, keys)
			if err != nil {
				return err
			}
			body, status, err := relayClient.GetBytes(cmd.Context(), mediaURL, map[string]string{"Authorization": auth}, 120*time.Second)
			if err != nil {
				return ExitError{Code: ExitRelay, Message: "download media failed", Err: err}
			}
			if status < 200 || status >= 300 {
				return ExitError{Code: ExitRelay, Message: fmt.Sprintf("relay returned %d: %s", status, strings.TrimSpace(string(body)))}
			}
			if output == "" || output == "-" {
				_, err := opts.stdout().Write(body)
				return err
			}
			return os.WriteFile(output, body, 0o600)
		},
	}
	get.Flags().StringP("output", "o", "", "output path. Omit or use '-' to write raw bytes to stdout")

	cmd.AddCommand(get)
	return cmd
}

// signBlossomGet signs a BUD-01 `t=get` Blossom auth event (kind 24242)
// scoped to the media URL's server (host[:port]) and a 10-minute expiry.
func signBlossomGet(keys *nostr.KeyPair, mediaURL string) (string, error) {
	domain := relayServerTag(mediaURL)
	if domain == "" {
		return "", fmt.Errorf("invalid media URL: %s", mediaURL)
	}
	exp := time.Now().Unix() + 600
	tags := nostr.Tags{
		{"t", "get"},
		{"expiration", fmt.Sprint(exp)},
		{"server", domain},
	}
	event := nostr.NewUnsignedEvent(nostr.KindBlossomAuth, keys.PublicHex(), "Get media", tags, 0)
	if err := event.Sign(keys); err != nil {
		return "", err
	}
	return "Nostr " + base64URLNoPad(event.MustJSON()), nil
}

// signBlossomUpload signs a BUD-02 `t=upload` Blossom auth event
// (kind 24242) scoped to the file's sha256, mime-dependent expiry
// (600s images, 3600s video), and the relay's server tag.
func signBlossomUpload(keys *nostr.KeyPair, sha256Hex, mime, relayURL string) (string, error) {
	expiry := int64(600)
	if strings.HasPrefix(mime, "video/") {
		expiry = 3600
	}
	exp := time.Now().Unix() + expiry
	tags := nostr.Tags{
		{"t", "upload"},
		{"x", sha256Hex},
		{"expiration", fmt.Sprint(exp)},
	}
	if domain := relayServerTag(relayURL); domain != "" {
		tags = append(tags, nostr.Tag{"server", domain})
	}
	event := nostr.NewUnsignedEvent(nostr.KindBlossomAuth, keys.PublicHex(), "Upload file", tags, 0)
	if err := event.Sign(keys); err != nil {
		return "", err
	}
	return "Nostr " + base64URLNoPad(event.MustJSON()), nil
}

// relayServerTag derives the Blossom "server" tag value (normalized
// host[:port], default ports 80/443 collapsed) from a relay or media URL.
// Mirrors buzz_core::tenant::relay_url_authority / normalize_host.
func relayServerTag(rawURL string) string {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u.Host == "" {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	host = strings.TrimSuffix(host, ".")
	if port := u.Port(); port != "" && port != "80" && port != "443" {
		host = host + ":" + port
	}
	return host
}

// mediaURLFromInput resolves a `media get` argument (a full same-origin
// media URL, or a bare sha256[.ext] / sha256.thumb.jpg segment) to the
// relay's absolute /media/<sha256[.ext]> URL. Mirrors media_url_from_input.
func mediaURLFromInput(relayURL, input string) (string, error) {
	input = strings.TrimSpace(input)
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		parsed, err := url.Parse(input)
		if err != nil {
			return "", inputWrap("invalid media URL", err)
		}
		sha, ok := strings.CutPrefix(parsed.Path, "/media/")
		if !ok {
			return "", inputError("media URL must point at a /media/ path")
		}
		if !isSafeMediaPathSegment(sha) {
			return "", inputError("media path must be sha256, sha256.ext, or sha256.thumb.jpg")
		}
		relay, err := url.Parse(relayURL)
		if err != nil {
			return "", inputWrap("invalid relay URL", err)
		}
		if parsed.Scheme != relay.Scheme || parsed.Hostname() != relay.Hostname() || defaultPort(parsed) != defaultPort(relay) {
			return "", inputError("refusing to sign media GET for a non-relay origin")
		}
		return input, nil
	}
	if strings.Contains(input, "://") {
		return "", inputError("media URL must use http:// or https://")
	}
	sha := strings.TrimPrefix(input, "/media/")
	if sha == "" {
		return "", inputError("media input must be a URL or sha256[.ext]")
	}
	if !isSafeMediaPathSegment(sha) {
		return "", inputError("media input must be sha256, sha256.ext, or sha256.thumb.jpg")
	}
	return strings.TrimRight(relayURL, "/") + "/media/" + sha, nil
}

func defaultPort(u *url.URL) string {
	if p := u.Port(); p != "" {
		return p
	}
	switch u.Scheme {
	case "http", "ws":
		return "80"
	case "https", "wss":
		return "443"
	}
	return ""
}

func isSafeMediaPathSegment(shaExt string) bool {
	segments := strings.Split(shaExt, ".")
	switch len(segments) {
	case 1:
		return isLowerHexSHA256(segments[0])
	case 2:
		return isLowerHexSHA256(segments[0]) && isSafeMediaExt(segments[1])
	case 3:
		return isLowerHexSHA256(segments[0]) && segments[1] == "thumb" && segments[2] == "jpg"
	default:
		return false
	}
}

func isLowerHexSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

func isSafeMediaExt(value string) bool {
	if value == "" || len(value) > 8 {
		return false
	}
	for _, r := range value {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

func base64URLNoPad(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}
