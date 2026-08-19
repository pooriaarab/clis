// Copyright 2026 Pooria Arab and contributors. Licensed under Apache-2.0. See LICENSE.
// hand-authored novel command; preserve across regenerate (separate file)

package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type bestShowtimeRow struct {
	Time              time.Time `json:"-"`
	TimeLabel         string    `json:"time"`
	Movie             string    `json:"movie"`
	Formats           string    `json:"formats"`
	Theatre           string    `json:"theatre"`
	SeatsRemaining    int       `json:"seatsRemaining"`
	ShowtimeID        int       `json:"showtimeId"`
	SeatMapURL        string    `json:"seatMapUrl"`
	FormatQualityRank int       `json:"formatQualityRank"`
}

type bestShowtimesRankOptions struct {
	NearMinutes *int
	FutureFirst bool
	Now         time.Time
}

type bestShowtimesEnvelope struct {
	Results []bestShowtimesTheatre `json:"results"`
}

type bestShowtimesTheatre struct {
	Theatre string              `json:"theatre"`
	Dates   []bestShowtimesDate `json:"dates"`
}

type bestShowtimesDate struct {
	StartDate string               `json:"startDate"`
	Movies    []bestShowtimesMovie `json:"movies"`
}

type bestShowtimesMovie struct {
	Name        string                    `json:"name"`
	Experiences []bestShowtimesExperience `json:"experiences"`
}

type bestShowtimesExperience struct {
	ExperienceTypes []string               `json:"experienceTypes"`
	Sessions        []bestShowtimesSession `json:"sessions"`
}

type bestShowtimesSession struct {
	ShowStartDateTime string `json:"showStartDateTime"`
	VistaSessionID    int    `json:"vistaSessionId"`
	SeatsRemaining    int    `json:"seatsRemaining"`
	IsSoldOut         bool   `json:"isSoldOut"`
	IsInThePast       bool   `json:"isInThePast"`
	SeatMapURL        string `json:"seatMapUrl"`
}

func newShowtimesBestCmd(flags *rootFlags) *cobra.Command {
	var flagTheatreIDs string
	var flagDate string
	var flagNear string
	var flagAfter string
	var flagBefore string
	var flagMovie string
	var flagExperience string
	var flagLimit int
	var flagLanguage string

	cmd := &cobra.Command{
		Use:     "best",
		Short:   "Find the best upcoming showtimes across theatres",
		Example: "  cineplex-pp-cli showtimes best --theatre-id 1145,1001 --near 19:00",
		Annotations: map[string]string{
			"pp:endpoint":   "showtimes.best",
			"pp:method":     "GET",
			"pp:path":       "/prod/cpx/theatrical/api/v1/showtimes",
			"mcp:read-only": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if !hasChangedLocalFlags(cmd) && len(args) == 0 && !flags.dryRun {
				if flags.asJSON {
					if printErr := printJSONFiltered(cmd.OutOrStdout(), map[string]any{
						"error": "requires input",
						"usage": cmd.CommandPath() + " --help",
					}, flags); printErr != nil {
						return printErr
					}
					return usageErr(fmt.Errorf("%q requires input; run %q for usage", cmd.CommandPath(), cmd.CommandPath()+" --help"))
				}
				return cmd.Help()
			}

			theatreIDs := splitCSV(flagTheatreIDs)
			if len(theatreIDs) == 0 && !flags.dryRun {
				return usageErr(fmt.Errorf("--theatre-id is required"))
			}
			if dryRunOK(flags) {
				return nil
			}

			nearMinutes, err := parseOptionalClock(flagNear)
			if err != nil {
				return usageErr(fmt.Errorf("invalid --near: %w", err))
			}
			afterMinutes, err := parseOptionalClock(flagAfter)
			if err != nil {
				return usageErr(fmt.Errorf("invalid --after: %w", err))
			}
			beforeMinutes, err := parseOptionalClock(flagBefore)
			if err != nil {
				return usageErr(fmt.Errorf("invalid --before: %w", err))
			}

			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			rows := make([]bestShowtimeRow, 0)
			movieFilter := strings.ToLower(strings.TrimSpace(flagMovie))
			experienceFilter := splitCSV(flagExperience)
			for _, theatreID := range theatreIDs {
				params := map[string]string{
					"language":   formatCLIParamValue(flagLanguage),
					"locationId": formatCLIParamValue(theatreID),
					"date":       formatCLIParamValue(flagDate),
				}
				data, _, fetchErr := resolveReadWithStrategyAndResponsePath(ctx, c, flags, "auto", "showtimes", true, "/prod/cpx/theatrical/api/v1/showtimes", params, nil, "", cmd.ErrOrStderr())
				if fetchErr != nil {
					return classifyAPIError(fetchErr, flags)
				}
				fetched, decodeErr := flattenBestShowtimes(data, flagDate, theatreID, movieFilter, experienceFilter, afterMinutes, beforeMinutes)
				if decodeErr != nil {
					return decodeErr
				}
				rows = append(rows, fetched...)
			}

			rankOptions := bestShowtimesRankOptions{
				NearMinutes: nearMinutes,
				FutureFirst: flagDate == time.Now().Format("1/2/2006"),
				Now:         time.Now(),
			}
			rows = rankBestShowtimes(rows, rankOptions)
			if flagLimit > 0 && len(rows) > flagLimit {
				rows = rows[:flagLimit]
			}

			if wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printBestShowtimesTable(cmd, rows)
			}
			return printJSONFiltered(cmd.OutOrStdout(), rows, flags)
		},
	}

	cmd.Flags().StringVar(&flagTheatreIDs, "theatre-id", "", "Comma-separated Cineplex theatre location IDs")
	cmd.Flags().StringVar(&flagDate, "date", time.Now().Format("1/2/2006"), "Showtime date in M/D/YYYY format")
	cmd.Flags().StringVar(&flagNear, "near", "", "Target time in HH:MM; rank by closeness")
	cmd.Flags().StringVar(&flagAfter, "after", "", "Only showtimes at or after HH:MM")
	cmd.Flags().StringVar(&flagBefore, "before", "", "Only showtimes at or before HH:MM")
	cmd.Flags().StringVar(&flagMovie, "movie", "", "Substring match on movie name")
	cmd.Flags().StringVar(&flagExperience, "experience", "", "Comma-separated experience type filters")
	cmd.Flags().IntVar(&flagLimit, "limit", 20, "Maximum number of showtimes")
	cmd.Flags().StringVar(&flagLanguage, "language", "en", "Content language")

	return cmd
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func parseOptionalClock(value string) (*int, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	t, err := time.Parse("15:04", strings.TrimSpace(value))
	if err != nil {
		return nil, fmt.Errorf("must use HH:MM")
	}
	minutes := t.Hour()*60 + t.Minute()
	return &minutes, nil
}

func flattenBestShowtimes(data []byte, requestedDate, theatreID, movieFilter string, experienceFilter []string, afterMinutes, beforeMinutes *int) ([]bestShowtimeRow, error) {
	theatres, err := decodeBestShowtimes(data)
	if err != nil {
		return nil, fmt.Errorf("decode showtimes response: %w", err)
	}

	rows := make([]bestShowtimeRow, 0)
	for _, theatre := range theatres {
		theatreName := theatre.Theatre
		if theatreName == "" {
			theatreName = theatreID
		}
		for _, date := range theatre.Dates {
			if date.StartDate != "" && !showtimesDateMatches(date.StartDate, requestedDate) {
				continue
			}
			for _, movie := range date.Movies {
				if movieFilter != "" && !strings.Contains(strings.ToLower(movie.Name), movieFilter) {
					continue
				}
				for _, experience := range movie.Experiences {
					if !matchesExperienceFilter(experience.ExperienceTypes, experienceFilter) {
						continue
					}
					formats := formatExperienceTypes(experience.ExperienceTypes)
					qualityRank := experienceQualityRank(experience.ExperienceTypes)
					for _, session := range experience.Sessions {
						if session.IsInThePast || session.IsSoldOut {
							continue
						}
						showtime, parseErr := time.Parse(time.RFC3339, session.ShowStartDateTime)
						if parseErr != nil {
							return nil, fmt.Errorf("decode showtime %q: %w", session.ShowStartDateTime, parseErr)
						}
						minutes := showtime.Hour()*60 + showtime.Minute()
						if afterMinutes != nil && minutes < *afterMinutes {
							continue
						}
						if beforeMinutes != nil && minutes > *beforeMinutes {
							continue
						}
						rows = append(rows, bestShowtimeRow{
							Time:              showtime,
							TimeLabel:         showtime.Format("3:04 PM"),
							Movie:             movie.Name,
							Formats:           formats,
							Theatre:           theatreName,
							SeatsRemaining:    session.SeatsRemaining,
							ShowtimeID:        session.VistaSessionID,
							SeatMapURL:        session.SeatMapURL,
							FormatQualityRank: qualityRank,
						})
					}
				}
			}
		}
	}
	return rows, nil
}

func decodeBestShowtimes(data []byte) ([]bestShowtimesTheatre, error) {
	var envelope bestShowtimesEnvelope
	if err := json.Unmarshal(data, &envelope); err == nil && envelope.Results != nil {
		return envelope.Results, nil
	}
	var theatres []bestShowtimesTheatre
	if err := json.Unmarshal(data, &theatres); err != nil {
		return nil, err
	}
	return theatres, nil
}

func showtimesDateMatches(value, requested string) bool {
	if value == requested {
		return true
	}
	for _, layout := range []string{"2006-01-02", time.RFC3339, "1/2/2006"} {
		parsed, err := time.Parse(layout, value)
		if err == nil && parsed.Format("1/2/2006") == requested {
			return true
		}
	}
	return false
}

func matchesExperienceFilter(formats, filter []string) bool {
	if len(filter) == 0 {
		return true
	}
	for _, requested := range filter {
		for _, format := range formats {
			if strings.EqualFold(strings.TrimSpace(requested), strings.TrimSpace(format)) {
				return true
			}
		}
	}
	return false
}

func formatExperienceTypes(formats []string) string {
	seen := make(map[string]struct{}, len(formats))
	unique := make([]string, 0, len(formats))
	for _, format := range formats {
		trimmed := strings.TrimSpace(format)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, trimmed)
	}
	if len(unique) == 0 {
		return "Regular"
	}
	return strings.Join(unique, ", ")
}

func experienceQualityRank(formats []string) int {
	qualities := []struct {
		name string
		rank int
	}{
		{name: "IMAX", rank: 8},
		{name: "UltraAVX", rank: 7},
		{name: "D-BOX", rank: 6},
		{name: "VIP", rank: 5},
		{name: "Dolby Atmos", rank: 4},
		{name: "ScreenX", rank: 3},
		{name: "3D", rank: 2},
		{name: "Regular", rank: 1},
	}
	best := 1
	for _, format := range formats {
		value := strings.ToLower(strings.TrimSpace(format))
		for _, quality := range qualities {
			if strings.Contains(value, strings.ToLower(quality.name)) && quality.rank > best {
				best = quality.rank
			}
		}
	}
	return best
}

func rankBestShowtimes(rows []bestShowtimeRow, options bestShowtimesRankOptions) []bestShowtimeRow {
	ranked := append([]bestShowtimeRow(nil), rows...)
	sort.SliceStable(ranked, func(i, j int) bool {
		left, right := ranked[i], ranked[j]
		if options.NearMinutes != nil {
			leftDistance := absInt(showtimeMinutes(left.Time) - *options.NearMinutes)
			rightDistance := absInt(showtimeMinutes(right.Time) - *options.NearMinutes)
			if leftDistance != rightDistance {
				return leftDistance < rightDistance
			}
			leftAfter := showtimeMinutes(left.Time) >= *options.NearMinutes
			rightAfter := showtimeMinutes(right.Time) >= *options.NearMinutes
			if leftAfter != rightAfter {
				return leftAfter
			}
		} else if options.FutureFirst {
			leftFuture := !left.Time.Before(options.Now)
			rightFuture := !right.Time.Before(options.Now)
			if leftFuture != rightFuture {
				return leftFuture
			}
		}
		if !left.Time.Equal(right.Time) {
			return left.Time.Before(right.Time)
		}
		if left.SeatsRemaining != right.SeatsRemaining {
			return left.SeatsRemaining > right.SeatsRemaining
		}
		if left.FormatQualityRank != right.FormatQualityRank {
			return left.FormatQualityRank > right.FormatQualityRank
		}
		return left.ShowtimeID < right.ShowtimeID
	})
	return ranked
}

func showtimeMinutes(value time.Time) int {
	return value.Hour()*60 + value.Minute()
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func printBestShowtimesTable(cmd *cobra.Command, rows []bestShowtimeRow) error {
	tw := newTabWriter(cmd.OutOrStdout())
	fmt.Fprintln(tw, "TIME\tMOVIE\tFORMATS\tTHEATRE\tSEATS\tSHOWTIME ID\tSEAT MAP URL\tFORMAT QUALITY RANK")
	for _, row := range rows {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%d\t%d\t%s\t%d\n", row.TimeLabel, row.Movie, row.Formats, row.Theatre, row.SeatsRemaining, row.ShowtimeID, row.SeatMapURL, row.FormatQualityRank)
	}
	return tw.Flush()
}
