package cli

import (
	"testing"

	"buzz-cli/internal/nostr"
)

func strPtr(s string) *string { return &s }

func TestChannelSummaryFromTags(t *testing.T) {
	tests := []struct {
		name string
		tags nostr.Tags
		want channelSummary
		ok   bool
	}{
		{
			name: "full metadata",
			tags: nostr.Tags{
				{"d", "channel-1"},
				{"name", "Composer"},
				{"t", "text"},
				{"private"},
				{"about", "about text"},
				{"topic", "Composer work"},
				{"purpose", "Track UI for the composer"},
				{"archived", "true"},
			},
			want: channelSummary{
				ChannelID:   "channel-1",
				Name:        "Composer",
				ChannelType: strPtr("text"),
				Visibility:  strPtr("private"),
				Archived:    true,
				About:       strPtr("about text"),
				Topic:       strPtr("Composer work"),
				Purpose:     strPtr("Track UI for the composer"),
			},
			ok: true,
		},
		{
			name: "missing d tag",
			tags: nostr.Tags{{"name", "Composer"}},
			ok:   false,
		},
		{
			name: "missing name tag",
			tags: nostr.Tags{{"d", "channel-1"}},
			ok:   false,
		},
		{
			name: "channel_type tag is NOT read (oracle quirk: reads bare \"t\")",
			tags: nostr.Tags{{"d", "channel-1"}, {"name", "General"}, {"channel_type", "voice"}},
			want: channelSummary{ChannelID: "channel-1", Name: "General"},
			ok:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := channelSummaryFromTags(tt.tags)
			if ok != tt.ok {
				t.Fatalf("channelSummaryFromTags() ok = %v, want %v", ok, tt.ok)
			}
			if !ok {
				return
			}
			if got.ChannelID != tt.want.ChannelID || got.Name != tt.want.Name || got.Archived != tt.want.Archived {
				t.Fatalf("channelSummaryFromTags() = %+v, want %+v", got, tt.want)
			}
			if !strPtrEqual(got.ChannelType, tt.want.ChannelType) ||
				!strPtrEqual(got.Visibility, tt.want.Visibility) ||
				!strPtrEqual(got.About, tt.want.About) ||
				!strPtrEqual(got.Topic, tt.want.Topic) ||
				!strPtrEqual(got.Purpose, tt.want.Purpose) {
				t.Fatalf("channelSummaryFromTags() pointer fields = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func strPtrEqual(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
