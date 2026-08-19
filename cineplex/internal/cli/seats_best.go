// hand-authored novel command; preserve across regenerate

package cli

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

const seatLayoutPath = "https://apis.cineplex.com/prod/ticketing/api/v1/theatre/{theatreId}/showtime/{showtimeId}/seat-layout"

type seatLayout struct {
	TotalRows               int               `json:"totalRows"`
	TotalColumns            int               `json:"totalColumns"`
	MaxSeatSelectionAllowed int               `json:"maxSeatSelectionAllowed"`
	StandardSeats           seatLayoutSection `json:"standardSeats"`
	SeatMapURL              string            `json:"seatMapUrl,omitempty"`
}

type seatLayoutSection struct {
	ColumnCount int             `json:"columnCount"`
	RowCount    int             `json:"rowCount"`
	Rows        []seatLayoutRow `json:"rows"`
}

type seatLayoutRow struct {
	Number         int              `json:"number"`
	PhysicalNumber int              `json:"physicalNumber"`
	Label          string           `json:"label"`
	Seats          []seatLayoutSeat `json:"seats"`
}

type seatLayoutSeat struct {
	ID                   string `json:"id"`
	Column               int    `json:"column"`
	ColumnPhysicalNumber int    `json:"columnPhysicalNumber"`
	Label                string `json:"label"`
}

type seatRun struct {
	RowLabel    string
	Seats       []seatLayoutSeat
	Score       float64
	RowFraction float64
}

func (r seatRun) key() string {
	if len(r.Seats) == 0 {
		return r.RowLabel
	}
	return r.RowLabel + "-" + strconv.Itoa(r.Seats[0].ColumnPhysicalNumber) + "-" + strconv.Itoa(r.Seats[len(r.Seats)-1].ColumnPhysicalNumber)
}

func (r seatRun) labels() []string {
	labels := make([]string, 0, len(r.Seats))
	for _, seat := range r.Seats {
		labels = append(labels, seat.Label)
	}
	return labels
}

func (r seatRun) labelRange() string {
	labels := r.labels()
	if len(labels) == 0 {
		return ""
	}
	if len(labels) == 1 {
		return labels[0]
	}
	return labels[0] + "-" + labels[len(labels)-1]
}

func rankSeatRuns(rows []seatLayoutRow, count int, rowsBack float64) []seatRun {
	if count <= 0 {
		return nil
	}
	nonEmpty := make([]seatLayoutRow, 0, len(rows))
	allSeats := make([]seatLayoutSeat, 0)
	for _, row := range rows {
		if len(row.Seats) == 0 {
			continue
		}
		sortedSeats := append([]seatLayoutSeat(nil), row.Seats...)
		sort.SliceStable(sortedSeats, func(i, j int) bool {
			return sortedSeats[i].ColumnPhysicalNumber < sortedSeats[j].ColumnPhysicalNumber
		})
		row.Seats = sortedSeats
		nonEmpty = append(nonEmpty, row)
		allSeats = append(allSeats, sortedSeats...)
	}
	if len(nonEmpty) == 0 || len(allSeats) == 0 {
		return nil
	}

	minColumn, maxColumn := allSeats[0].ColumnPhysicalNumber, allSeats[0].ColumnPhysicalNumber
	for _, seat := range allSeats[1:] {
		minColumn = minInt(minColumn, seat.ColumnPhysicalNumber)
		maxColumn = maxInt(maxColumn, seat.ColumnPhysicalNumber)
	}
	centerColumn := float64(minColumn+maxColumn) / 2

	runs := make([]seatRun, 0)
	for rowIndex, row := range nonEmpty {
		rowFraction := 0.0
		if len(nonEmpty) > 1 {
			rowFraction = float64(rowIndex) / float64(len(nonEmpty)-1)
		}
		for start := 0; start+count <= len(row.Seats); start++ {
			candidate := row.Seats[start : start+count]
			contiguous := true
			for i := 1; i < len(candidate); i++ {
				if candidate[i].ColumnPhysicalNumber != candidate[i-1].ColumnPhysicalNumber+1 {
					contiguous = false
					break
				}
			}
			if !contiguous {
				continue
			}
			runCenter := float64(candidate[0].ColumnPhysicalNumber+candidate[len(candidate)-1].ColumnPhysicalNumber) / 2
			runs = append(runs, seatRun{
				RowLabel:    row.Label,
				Seats:       append([]seatLayoutSeat(nil), candidate...),
				Score:       math.Abs(runCenter-centerColumn)*0.7 + math.Abs(rowFraction-rowsBack)*22,
				RowFraction: rowFraction,
			})
		}
	}
	sort.SliceStable(runs, func(i, j int) bool { return runs[i].Score < runs[j].Score })
	return runs
}

func newSeatsBestCmd(flags *rootFlags) *cobra.Command {
	var count int
	var rowsBack float64

	cmd := &cobra.Command{
		Use:         "best <theatreId> <showtimeId>",
		Short:       "Find the best adjacent seats for a showtime",
		Annotations: map[string]string{"mcp:read-only": "true", "pp:endpoint": "seats.best", "pp:method": "GET", "pp:path": seatLayoutPath},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !hasChangedLocalFlags(cmd) && !flags.dryRun {
				return cmd.Help()
			}
			if len(args) < 2 {
				return usageErr(fmt.Errorf("missing required arguments\nUsage: %s <theatreId> <showtimeId>", cmd.CommandPath()))
			}
			if count <= 0 {
				return usageErr(fmt.Errorf("--count must be greater than zero"))
			}
			if dryRunOK(flags) {
				return nil
			}

			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			path := replacePathParam(replacePathParam(seatLayoutPath, "theatreId", args[0]), "showtimeId", args[1])
			data, _, err := resolveReadWithStrategyAndResponsePath(ctx, c, flags, "live", "seats", false, path, nil, nil, "", cmd.ErrOrStderr())
			if err != nil {
				return classifyAPIError(err, flags)
			}
			layout, err := decodeSeatLayout(data)
			if err != nil {
				return fmt.Errorf("decoding seat layout: %w", err)
			}
			runs := rankSeatRuns(layout.StandardSeats.Rows, count, rowsBack)
			if len(runs) == 0 {
				return fmt.Errorf("no contiguous run of %d seats found", count)
			}

			availabilityNote := ""
			// Authed ticketing uses the SCENE+ session COOKIE (confirmed by
			// capturing a real checkout), not a bearer token.
			if cookie := c.Config.SceneSessionCookie(); cookie != "" {
				availabilityPath := strings.Replace(path, "seat-layout", "seat-availability-for-cart", 1)
				availability, availabilityErr := c.GetWithHeaders(ctx, availabilityPath, nil, map[string]string{"Cookie": cookie})
				if availabilityErr == nil {
					if available := parseAvailableSeats(availability); len(available) > 0 {
						filtered := filterAvailableSeatRuns(runs, available)
						if len(filtered) > 0 {
							runs = filtered
						} else {
							availabilityNote = "no available run of that size; showing physical-best (some seats taken)"
						}
					} else {
						availabilityNote = "availability unavailable; showing physical-best"
					}
				} else {
					availabilityNote = "availability unavailable (session cookie may be expired); showing physical-best"
				}
			} else {
				availabilityNote = "live availability needs a SCENE+ session cookie: auth set-cookie <cookie>"
			}
			if availabilityNote != "" {
				fmt.Fprintln(cmd.ErrOrStderr(), availabilityNote)
			}

			seatMapURL := layout.SeatMapURL
			if seatMapURL == "" {
				seatMapURL = seatMapURLFor(args[0], args[1])
			}
			out := bestSeatOutput{
				Best:         seatRecommendationFromRun(runs[0], count, seatMapURL),
				Alternatives: recommendationsFromRuns(runs[1:], count, seatMapURL, 3),
				SeatMapURL:   seatMapURL,
				Availability: availabilityNote,
			}
			if wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printBestSeatText(cmd, out)
			}
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		},
	}
	cmd.Flags().IntVar(&count, "count", 2, "Number of adjacent seats")
	cmd.Flags().Float64Var(&rowsBack, "rows-back", 0.64, "Ideal row position from screen to back")
	return cmd
}

type bestSeatOutput struct {
	Best         seatRecommendation   `json:"best"`
	Alternatives []seatRecommendation `json:"alternatives"`
	SeatMapURL   string               `json:"seatMapUrl,omitempty"`
	Availability string               `json:"availability,omitempty"`
}

type seatRecommendation struct {
	Seats       []string `json:"seats"`
	Range       string   `json:"range"`
	Row         string   `json:"row"`
	Summary     string   `json:"summary"`
	SeatMapURL  string   `json:"seatMapUrl,omitempty"`
	Score       float64  `json:"score"`
	RowFraction float64  `json:"rowFraction"`
}

func seatRecommendationFromRun(run seatRun, count int, seatMapURL string) seatRecommendation {
	return seatRecommendation{
		Seats:       run.labels(),
		Range:       run.labelRange(),
		Row:         run.RowLabel,
		Summary:     fmt.Sprintf("Best %d together: %s — center, ~%d%% back = sweet spot", count, run.labelRange(), int(math.Round(run.RowFraction*100))),
		SeatMapURL:  seatMapURL,
		Score:       run.Score,
		RowFraction: run.RowFraction,
	}
}

func recommendationsFromRuns(runs []seatRun, count int, seatMapURL string, limit int) []seatRecommendation {
	if len(runs) > limit {
		runs = runs[:limit]
	}
	out := make([]seatRecommendation, 0, len(runs))
	for _, run := range runs {
		out = append(out, seatRecommendationFromRun(run, count, seatMapURL))
	}
	return out
}

func printBestSeatText(cmd *cobra.Command, out bestSeatOutput) error {
	fmt.Fprintln(cmd.OutOrStdout(), out.Best.Summary)
	fmt.Fprintf(cmd.OutOrStdout(), "Row: %s\nSeats: %s\nSeat map: %s\n", out.Best.Row, strings.Join(out.Best.Seats, ", "), out.SeatMapURL)
	if len(out.Alternatives) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Alternatives:")
		for _, alternative := range out.Alternatives {
			fmt.Fprintf(cmd.OutOrStdout(), "- %s (%s)\n", alternative.Range, alternative.Row)
		}
	}
	return nil
}

func decodeSeatLayout(data json.RawMessage) (seatLayout, error) {
	var envelope struct {
		Results json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return seatLayout{}, err
	}
	payload := data
	if len(envelope.Results) > 0 && string(envelope.Results) != "null" {
		payload = envelope.Results
	}
	var layout seatLayout
	if err := json.Unmarshal(payload, &layout); err != nil {
		return seatLayout{}, err
	}
	return layout, nil
}

// parseAvailableSeats reads the CONFIRMED seat-availability-for-cart response
// shape {"seatAvailabilities":{"<seatId>":"Available"|"Occupied",...}, ...}
// (keys match the seat-layout seat.id, e.g. "1_17_30"), falling back to a
// generic probe for other/unknown shapes.
func parseAvailableSeats(data json.RawMessage) map[string]bool {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil
	}
	available := map[string]bool{}
	// Confirmed shape first.
	if root, ok := value.(map[string]any); ok {
		if sa, ok := root["seatAvailabilities"].(map[string]any); ok {
			for id, status := range sa {
				if s, ok := status.(string); ok {
					available[id] = strings.EqualFold(s, "Available")
				}
			}
			if len(available) > 0 {
				return available
			}
		}
	}
	// Fallback: generic probe (objects carrying id + status/available fields).
	probeAvailableSeats(value, available)
	return available
}

func probeAvailableSeats(value any, available map[string]bool) {
	switch node := value.(type) {
	case []any:
		for _, item := range node {
			probeAvailableSeats(item, available)
		}
	case map[string]any:
		identifier := firstString(node, "id", "seatId", "seat_id", "seatCode", "label")
		if identifier != "" {
			if state, ok := availabilityState(node); ok {
				available[identifier] = state
			}
		}
		for _, item := range node {
			probeAvailableSeats(item, available)
		}
	}
}

func availabilityState(node map[string]any) (bool, bool) {
	for _, key := range []string{"available", "isAvailable", "is_available", "selectable", "isSelectable", "open"} {
		if value, ok := node[key].(bool); ok {
			return value, true
		}
	}
	if status, ok := firstStringValue(node, "status", "state"); ok {
		switch strings.ToLower(status) {
		case "available", "open", "free", "selectable":
			return true, true
		case "unavailable", "occupied", "reserved", "blocked", "sold":
			return false, true
		}
	}
	return false, false
}

func filterAvailableSeatRuns(runs []seatRun, available map[string]bool) []seatRun {
	out := make([]seatRun, 0, len(runs))
	for _, run := range runs {
		ok := true
		for _, seat := range run.Seats {
			state, found := available[seat.ID]
			if !found {
				state, found = available[seat.Label]
			}
			if !found || !state {
				ok = false
				break
			}
		}
		if ok {
			out = append(out, run)
		}
	}
	return out
}

func firstString(node map[string]any, keys ...string) string {
	value, _ := firstStringValue(node, keys...)
	return value
}

func firstStringValue(node map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		if value, ok := node[key].(string); ok && strings.TrimSpace(value) != "" {
			return value, true
		}
	}
	return "", false
}

func seatMapURLFor(theatreID, showtimeID string) string {
	return "https://www.cineplex.com/ticketing/preview?theatreId=" + theatreID + "&showtimeId=" + showtimeID + "&dbox=false"
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
