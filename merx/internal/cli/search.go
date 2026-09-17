package cli

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"merx-cli/internal/httpx"
	"merx-cli/internal/search"
)

func searchCmd() *cobra.Command {
	var status, keywords, category, location, publishDate, sortBy, sortDir string
	var page int
	cmd := &cobra.Command{Use: "search", Short: "Search public MERX solicitation lists", RunE: func(*cobra.Command, []string) error {
		q := search.Query{
			Status: status, Keywords: keywords, Category: category, Location: location,
			PublishDate: publishDate, SortBy: sortBy, SortDirection: sortDir, Page: page,
		}
		u, err := q.URL()
		if err != nil {
			return err
		}
		c, err := httpx.New()
		if err != nil {
			return err
		}
		req, err := httpx.NewRequest(http.MethodGet, u)
		if err != nil {
			return err
		}
		resp, err := c.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("GET %s: HTTP %d", u, resp.StatusCode)
		}
		pageOut, err := search.Parse(bytes.NewReader(body))
		if err != nil {
			return err
		}
		if w := search.CeilingWarning(pageOut.Total); w != "" {
			fmt.Fprintln(os.Stderr, w)
		}
		if flagJSON {
			return emit(struct {
				Status    string          `json:"status"`
				Total     int             `json:"total"`
				Page      int             `json:"page"`
				PageSize  int             `json:"page_size"`
				Reachable int             `json:"reachable"`
				Records   []search.Record `json:"records"`
			}{status, pageOut.Total, page, search.PageSize, search.MaxReachable, pageOut.Records})
		}
		fmt.Printf("%d results\n", pageOut.Total)
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNUMBER\tTITLE\tBUYER\tLOCATION\tPUBLISHED\tCLOSING\tDAYS")
		for _, r := range pageOut.Records {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				r.InternalID, r.Solicitation, r.Title, r.Buyer, r.Location, r.Published, r.Closing, r.DaysRemaining)
		}
		return w.Flush()
	}}
	cmd.Flags().StringVar(&status, "status", "open", "list: open, awarded, bid-results, closed")
	cmd.Flags().StringVar(&keywords, "keywords", "", "free-text keywords")
	cmd.Flags().StringVar(&category, "category", "", "comma-separated MERX category ids")
	cmd.Flags().StringVar(&location, "location", "", "comma-separated MERX location ids")
	cmd.Flags().StringVar(&publishDate, "publish-date", "", "HOURS_24 or WEEK_1")
	cmd.Flags().StringVar(&sortBy, "sort-by", "", "score, noticeTitle, region, publicationDate, closingDate")
	cmd.Flags().StringVar(&sortDir, "sort-direction", "", "ASC or DESC")
	cmd.Flags().IntVar(&page, "page", 1, "1-based page number (max 40)")
	return cmd
}
