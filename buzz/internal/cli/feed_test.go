package cli

import "testing"

func TestParseFeedTypes(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    []string
		wantErr bool
	}{
		{name: "empty", raw: "", want: nil},
		{name: "single", raw: "mentions", want: []string{"mentions"}},
		{name: "csv trims spaces", raw: "mentions, needs_action,activity", want: []string{"mentions", "needs_action", "activity"}},
		{name: "invalid", raw: "mentions,bad", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFeedTypes(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseFeedTypes(%q) error = nil", tt.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseFeedTypes(%q) error = %v", tt.raw, err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("parseFeedTypes(%q) len = %d, want %d (%#v)", tt.raw, len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("parseFeedTypes(%q)[%d] = %q, want %q", tt.raw, i, got[i], tt.want[i])
				}
			}
		})
	}
}
