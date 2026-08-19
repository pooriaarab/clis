package cli

import (
	"strconv"
	"testing"
)

func TestRankSeatRuns(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		rows     []seatLayoutRow
		count    int
		rowsBack float64
		want     []string
	}{
		{
			name: "prefers target depth and center",
			rows: []seatLayoutRow{
				{Label: "AA", Seats: seats("AA", 1, 2, 3, 4, 5)},
				{Label: "A", Seats: seats("A", 1, 2, 3, 4, 5)},
				{Label: "B", Seats: seats("B", 1, 2, 3, 4, 5)},
				{Label: "C", Seats: seats("C", 1, 2, 3, 4, 5)},
				{Label: "D", Seats: seats("D", 1, 2, 3, 4, 5)},
			},
			count:    2,
			rowsBack: 0.75,
			want:     []string{"C-2-3", "C-3-4", "C-1-2"},
		},
		{
			name: "skips gaps and ranks alternatives",
			rows: []seatLayoutRow{
				{Label: "A", Seats: seats("A", 1, 2, 4, 5)},
				{Label: "B", Seats: seats("B", 1, 2, 3, 4)},
			},
			count:    3,
			rowsBack: 1,
			want:     []string{"B-2-4"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := rankSeatRuns(tc.rows, tc.count, tc.rowsBack)
			for i, want := range tc.want {
				if i >= len(got) || got[i].key() != want {
					t.Fatalf("rankSeatRuns()[%d] = %q, want %q; all=%v", i, seatRunKey(got, i), want, got)
				}
			}
		})
	}
}

func seats(row string, columns ...int) []seatLayoutSeat {
	out := make([]seatLayoutSeat, 0, len(columns))
	for _, column := range columns {
		out = append(out, seatLayoutSeat{
			ID:                   row + "-" + strconv.Itoa(column),
			ColumnPhysicalNumber: column,
			Label:                row + strconv.Itoa(column),
		})
	}
	return out
}

func seatRunKey(runs []seatRun, index int) string {
	if index >= len(runs) {
		return "<missing>"
	}
	return runs[index].key()
}
