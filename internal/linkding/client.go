package linkding

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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

// FetchAllBookmarks returns all bookmarks. Pagination is handled internally.
// Stub — full implementation in Task 3.
func (c *Client) FetchAllBookmarks(ctx context.Context) ([]Bookmark, error) {
	page, err := c.fetchPage(ctx, 0)
	if err != nil {
		return nil, err
	}
	return page.Results, nil
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
