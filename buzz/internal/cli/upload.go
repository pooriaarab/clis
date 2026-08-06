package cli

// buzz upload file — BUD-02 Blossom upload. Mirrors upload_file in
// /Users/parab/code/buzz/crates/buzz-cli/src/client.rs: sniff MIME from
// magic bytes, enforce the allowed-type/size limits, PUT to /upload with a
// fresh Blossom auth header, falling back to the legacy /media/upload path
// on 404/405.

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// allowedUploadMimes mirrors ALLOWED_MIMES in client.rs.
var allowedUploadMimes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
	"video/mp4":  true,
}

const (
	maxImageUploadBytes = 50 * 1024 * 1024
	maxVideoUploadBytes = 500 * 1024 * 1024
)

func uploadCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "upload", Short: "Upload commands"}

	file := &cobra.Command{
		Use:   "file",
		Short: "Upload a file to the relay's Blossom store",
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, err := requiredFlag(cmd, "file")
			if err != nil {
				return err
			}
			info, err := os.Stat(filePath)
			if err != nil {
				return otherWrap("cannot access "+filePath, err)
			}
			if !info.Mode().IsRegular() {
				return inputError(filePath + " is not a file")
			}
			data, err := os.ReadFile(filePath)
			if err != nil {
				return otherWrap("failed to read "+filePath, err)
			}

			// ponytail: net/http.DetectContentType implements the WHATWG
			// mimesniff algorithm (same magic-byte family as the Rust
			// `infer` crate); minor edge-case drift is acceptable here.
			mime := http.DetectContentType(data)
			if idx := strings.IndexByte(mime, ';'); idx >= 0 {
				mime = mime[:idx]
			}
			if !allowedUploadMimes[mime] {
				return inputError("unsupported file type: " + mime)
			}
			max := int64(maxImageUploadBytes)
			if strings.HasPrefix(mime, "video/") {
				max = maxVideoUploadBytes
			}
			if int64(len(data)) > max {
				return inputError("file too large: " + strconv.Itoa(len(data)) + " bytes (max " + strconv.FormatInt(max, 10) + ")")
			}

			sum := sha256.Sum256(data)
			shaHex := hex.EncodeToString(sum[:])

			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			if resolved.RelayURL == "" {
				return inputError("relay URL is required")
			}
			relayClient, err := restClientFromResolved(resolved, keys)
			if err != nil {
				return err
			}

			timeout := 120 * time.Second
			if strings.HasPrefix(mime, "video/") {
				timeout = 600 * time.Second
			}

			doPut := func(path string) (jsonRaw []byte, status int, err error) {
				auth, err := signBlossomUpload(keys, shaHex, mime, resolved.RelayURL)
				if err != nil {
					return nil, 0, otherWrap("sign blossom upload auth", err)
				}
				headers := map[string]string{
					"Authorization": auth,
					"Content-Type":  mime,
					"X-SHA-256":     shaHex,
				}
				raw, status, err := relayClient.PutRaw(cmd.Context(), path, data, headers, timeout)
				return raw, status, err
			}

			raw, status, err := doPut("/upload")
			if err != nil {
				return ExitError{Code: ExitRelay, Message: "upload failed", Err: err}
			}
			if status == http.StatusNotFound || status == http.StatusMethodNotAllowed {
				// Legacy relay: fall back to /media/upload.
				raw, status, err = doPut("/media/upload")
				if err != nil {
					return ExitError{Code: ExitRelay, Message: "upload failed", Err: err}
				}
			}
			if status < 200 || status >= 300 {
				return ExitError{Code: ExitRelay, Message: "relay returned " + strconv.Itoa(status) + ": " + strings.TrimSpace(string(raw))}
			}
			return opts.writeRawJSON(raw)
		},
	}
	file.Flags().String("file", "", "path to the file to upload")

	cmd.AddCommand(file)
	return cmd
}
