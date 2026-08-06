package cli

import "testing"

func TestNormalizeEmojiShortcode(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{name: "lowercases and trims colons", raw: " :Party-Parrot: ", want: "party-parrot"},
		{name: "allows underscore", raw: "Ship_It", want: "ship_it"},
		{name: "empty", raw: "::", wantErr: true},
		{name: "too long", raw: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklm", wantErr: true},
		{name: "invalid char", raw: "party.parrot", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeEmojiShortcode(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("normalizeEmojiShortcode(%q) error = nil", tt.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeEmojiShortcode(%q) error = %v", tt.raw, err)
			}
			if got != tt.want {
				t.Fatalf("normalizeEmojiShortcode(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}
