package cli

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"merx-cli/internal/httpx"
	"merx-cli/internal/search"
)

func showCmd() *cobra.Command {
	return &cobra.Command{Use: "show <id>", Short: "Show one public MERX notice or solicitation", Args: cobra.ExactArgs(1), RunE: func(_ *cobra.Command, args []string) error {
		c, err := httpx.New()
		if err != nil {
			return err
		}
		n, err := loadNotice(c, args[0])
		if err != nil {
			return err
		}
		if flagJSON {
			return emit(n)
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "FIELD\tVALUE")
		for _, row := range [][2]string{
			{"internal_id", n.InternalID}, {"detail_url", n.DetailURL},
			{"reference_number", n.ReferenceNumber}, {"issuing_organization", n.Buyer},
			{"project_type", n.ProjectType}, {"project_number", n.ProjectNumber},
			{"title", n.Title}, {"source_id", n.SourceID},
			{"agreement_types", strings.Join(n.AgreementTypes, "; ")},
			{"location", n.Location}, {"job_location", n.JobLocation}, {"purchase_type", n.PurchaseType},
			{"publication_date", n.Published}, {"closing_date", n.Closing}, {"bid_intent", n.BidIntent},
			{"contact_name", n.ContactName}, {"contact_phone", n.ContactPhone}, {"contact_email", n.ContactEmail},
			{"bid_submission_type", n.BidSubmission}, {"pricing", n.Pricing}, {"description", n.Description},
			{"categories_merx", joinCategories(n.MERX)}, {"categories_gsin", joinCategories(n.GSIN)}, {"categories_unspsc", joinCategories(n.UNSPSC)},
		} {
			fmt.Fprintf(w, "%s\t%s\n", row[0], row[1])
		}
		return w.Flush()
	}}
}

func joinCategories(cats []search.Category) string {
	parts := make([]string, len(cats))
	for i, c := range cats {
		parts[i] = c.Code + ": " + c.Name
	}
	return strings.Join(parts, "; ")
}

func loadNotice(c *httpx.Client, id string) (search.Notice, error) {
	var last error
	for _, u := range search.DetailCandidates(id) {
		n, err := getNotice(c, u)
		if err != nil {
			last = err
			continue
		}
		return n, nil
	}
	for _, status := range []string{"open", "awarded", "bid-results", "closed"} {
		n, err := findByKeyword(c, status, id)
		if err != nil {
			last = err
			continue
		}
		return n, nil
	}
	return search.Notice{}, last
}

func findByKeyword(c *httpx.Client, status, id string) (search.Notice, error) {
	list, err := (search.Query{Status: status, Keywords: id, Page: 1}).URL()
	if err != nil {
		return search.Notice{}, err
	}
	body, _, err := getOK(c, list)
	if err != nil {
		return search.Notice{}, err
	}
	page, err := search.Parse(bytes.NewReader(body))
	if err != nil {
		return search.Notice{}, err
	}
	u, err := search.PickSearchURL(page, id)
	if err != nil {
		return search.Notice{}, err
	}
	return getNotice(c, u)
}

func getNotice(c *httpx.Client, u string) (search.Notice, error) {
	body, final, err := getOK(c, u)
	if err != nil {
		return search.Notice{}, err
	}
	n, err := search.ParseDetail(bytes.NewReader(body))
	if err != nil {
		return search.Notice{}, err
	}
	n.DetailURL = final
	search.LoadCategories(func(cu string) ([]byte, error) {
		raw, _, err := getOK(c, cu)
		return raw, err
	}, &n)
	return n, nil
}

func getOK(c *httpx.Client, u string) ([]byte, string, error) {
	req, err := httpx.NewRequest(http.MethodGet, u)
	if err != nil {
		return nil, "", err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("GET %s: HTTP %d", u, resp.StatusCode)
	}
	final := u
	if resp.Request != nil && resp.Request.URL != nil {
		final = resp.Request.URL.String()
	}
	return body, final, nil
}
