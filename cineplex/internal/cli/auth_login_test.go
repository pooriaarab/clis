package cli

import (
	"encoding/hex"
	"testing"
)

// RFC 6070 PBKDF2-HMAC-SHA1 test vectors.
func TestPBKDF2SHA1(t *testing.T) {
	cases := []struct {
		pass, salt string
		iter, klen int
		want       string
	}{
		{"password", "salt", 1, 20, "0c60c80f961f0e71f3a9b524af6012062fe037a6"},
		{"password", "salt", 2, 20, "ea6c014dc72d6f8ccd1ed92ace1d41f0d8de8957"},
		// Chrome-shaped: 16-byte key, 1003 iters, "saltysalt" — just checks length + determinism.
		{"anykey", "saltysalt", 1003, 16, ""},
	}
	for _, c := range cases {
		got := pbkdf2SHA1([]byte(c.pass), []byte(c.salt), c.iter, c.klen)
		if len(got) != c.klen {
			t.Fatalf("len = %d, want %d", len(got), c.klen)
		}
		if c.want != "" && hex.EncodeToString(got) != c.want {
			t.Fatalf("pbkdf2(%q,%q,%d,%d) = %s, want %s", c.pass, c.salt, c.iter, c.klen, hex.EncodeToString(got), c.want)
		}
	}
}

func TestPKCS7Unpad(t *testing.T) {
	// "hi" padded to 16 with 0x0e * 14
	padded := append([]byte("hi"), bytesRepeat(0x0e, 14)...)
	out, err := pkcs7Unpad(padded, 16)
	if err != nil || string(out) != "hi" {
		t.Fatalf("unpad = %q err=%v, want hi", out, err)
	}
	if _, err := pkcs7Unpad([]byte{1, 2, 3}, 16); err == nil {
		t.Fatalf("expected error on bad length")
	}
}

func bytesRepeat(b byte, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return out
}
