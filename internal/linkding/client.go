package linkding

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/sync/errgroup"
)

const pageSize = 100

// Bookmark represents a single linkding bookmark.
type Bookmark struct {
	ID           int      `json:"id"`
	URL          string   `json:"url"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	WebsiteTitle string   `json:"website_title"`
	TagNames     []string `json:"tag_names"`
}

type bookmarksResponse struct {
	Count   int        `json:"count"`
	Results []Bookmark `json:"results"`
}

// Client calls the linkding REST API.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// New creates a Client with a default HTTP client.
func New(baseURL, token string) *Client {
	return &Client{
		BaseURL:    baseURL,
		Token:      token,
		HTTPClient: &http.Client{},
	}
}

// FetchAllBookmarks returns all bookmarks from the linkding API,
// fetching additional pages in parallel if the total exceeds pageSize.
func (c *Client) FetchAllBookmarks(ctx context.Context) ([]Bookmark, error) {
	first, err := c.fetchPage(ctx, 0)
	if err != nil {
		return nil, err
	}

	totalPages := (first.Count + pageSize - 1) / pageSize
	if totalPages <= 1 {
		return first.Results, nil
	}

	// Pre-allocate slice so each goroutine writes to its own index — no mutex needed.
	pages := make([][]Bookmark, totalPages)
	pages[0] = first.Results

	g, gctx := errgroup.WithContext(ctx)
	for i := 1; i < totalPages; i++ {
		i := i // Go <1.22 loop var capture; harmless in 1.24
		g.Go(func() error {
			page, err := c.fetchPage(gctx, i*pageSize)
			if err != nil {
				return err
			}
			pages[i] = page.Results
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	var all []Bookmark
	for _, p := range pages {
		all = append(all, p...)
	}
	return all, nil
}

func (c *Client) fetchPage(ctx context.Context, offset int) (*bookmarksResponse, error) {
	url := fmt.Sprintf("%s/api/bookmarks/?limit=%d&offset=%d", c.BaseURL, pageSize, offset)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Token "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("linkding API returned status %d (offset %d)", resp.StatusCode, offset)
	}

	var result bookmarksResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
