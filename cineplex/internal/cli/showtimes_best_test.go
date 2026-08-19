package cli

import (
	"testing"
	"time"
)

func TestRankBestShowtimes(t *testing.T) {
	target := 20 * 60
	today := time.Date(2026, time.August, 18, 12, 0, 0, 0, time.Local)

	tests := []struct {
		name    string
		rows    []bestShowtimeRow
		options bestShowtimesRankOptions
		wantIDs []int
	}{
		{
			name: "near prefers soonest after on equal distance then seats",
			rows: []bestShowtimeRow{
				{ShowtimeID: 1, Time: today.Add(8*time.Hour + 30*time.Minute), SeatsRemaining: 20, FormatQualityRank: 8},
				{ShowtimeID: 2, Time: today.Add(7*time.Hour + 30*time.Minute), SeatsRemaining: 80, FormatQualityRank: 1},
				{ShowtimeID: 3, Time: today.Add(8*time.Hour + 30*time.Minute), SeatsRemaining: 80, FormatQualityRank: 1},
			},
			options: bestShowtimesRankOptions{NearMinutes: &target},
			wantIDs: []int{3, 1, 2},
		},
		{
			name: "chronological ranking prefers future today",
			rows: []bestShowtimeRow{
				{ShowtimeID: 4, Time: today.Add(-time.Hour), SeatsRemaining: 90, FormatQualityRank: 8},
				{ShowtimeID: 5, Time: today.Add(3 * time.Hour), SeatsRemaining: 10, FormatQualityRank: 1},
				{ShowtimeID: 6, Time: today.Add(2 * time.Hour), SeatsRemaining: 50, FormatQualityRank: 1},
			},
			options: bestShowtimesRankOptions{FutureFirst: true, Now: today},
			wantIDs: []int{6, 5, 4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rankBestShowtimes(tt.rows, tt.options)
			if len(got) != len(tt.wantIDs) {
				t.Fatalf("rankBestShowtimes() returned %d rows, want %d", len(got), len(tt.wantIDs))
			}
			for i, wantID := range tt.wantIDs {
				if got[i].ShowtimeID != wantID {
					t.Errorf("rankBestShowtimes()[%d].ShowtimeID = %d, want %d", i, got[i].ShowtimeID, wantID)
				}
			}
		})
	}
}
