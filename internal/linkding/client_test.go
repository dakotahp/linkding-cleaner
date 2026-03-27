package linkding

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchPage_SinglePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/bookmarks/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("limit") != "100" {
			t.Errorf("expected limit=100, got %s", r.URL.Query().Get("limit"))
		}
		if r.URL.Query().Get("offset") != "0" {
			t.Errorf("expected offset=0, got %s", r.URL.Query().Get("offset"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(bookmarksResponse{
			Count: 2,
			Results: []Bookmark{
				{ID: 1, URL: "https://example.com", WebsiteTitle: "Example"},
				{ID: 2, URL: "https://golang.org", WebsiteTitle: "Go"},
			},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	got, err := c.fetchPage(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.Count != 2 {
		t.Errorf("expected count=2, got %d", got.Count)
	}
	if len(got.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(got.Results))
	}
	if got.Results[0].ID != 1 {
		t.Errorf("expected first bookmark ID=1, got %d", got.Results[0].ID)
	}
}

func TestFetchPage_AuthHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(bookmarksResponse{Count: 0, Results: []Bookmark{}})
	}))
	defer srv.Close()

	c := New(srv.URL, "my-secret-token")
	_, err := c.fetchPage(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Token my-secret-token" {
		t.Errorf("expected 'Token my-secret-token', got %q", gotAuth)
	}
}

func TestFetchPage_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := New(srv.URL, "bad-token")
	_, err := c.fetchPage(context.Background(), 0)
	if err == nil {
		t.Fatal("expected error for 401 response, got nil")
	}
}

func makeBookmarks(startID, count int) []Bookmark {
	b := make([]Bookmark, count)
	for i := range b {
		b[i] = Bookmark{
			ID:  startID + i,
			URL: fmt.Sprintf("https://example.com/%d", startID+i),
		}
	}
	return b
}

func TestFetchAllBookmarks_SinglePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(bookmarksResponse{
			Count:   2,
			Results: makeBookmarks(1, 2),
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	bookmarks, err := c.FetchAllBookmarks(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(bookmarks) != 2 {
		t.Errorf("expected 2 bookmarks, got %d", len(bookmarks))
	}
}

func TestFetchAllBookmarks_MultiPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		offset := r.URL.Query().Get("offset")
		w.Header().Set("Content-Type", "application/json")
		switch offset {
		case "", "0":
			_ = json.NewEncoder(w).Encode(bookmarksResponse{
				Count:   150,
				Results: makeBookmarks(1, 100),
			})
		case "100":
			_ = json.NewEncoder(w).Encode(bookmarksResponse{
				Count:   150,
				Results: makeBookmarks(101, 50),
			})
		default:
			t.Errorf("unexpected offset: %s", offset)
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	bookmarks, err := c.FetchAllBookmarks(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(bookmarks) != 150 {
		t.Errorf("expected 150 bookmarks, got %d", len(bookmarks))
	}
	// Verify page ordering: first 100 have IDs 1-100, next 50 have IDs 101-150
	if bookmarks[0].ID != 1 {
		t.Errorf("expected first bookmark ID=1, got %d", bookmarks[0].ID)
	}
	if bookmarks[100].ID != 101 {
		t.Errorf("expected bookmark[100] ID=101, got %d", bookmarks[100].ID)
	}
}

func TestFetchAllBookmarks_ExactlyOnePage(t *testing.T) {
	// count == pageSize should NOT trigger a second fetch
	var requestCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(bookmarksResponse{
			Count:   100,
			Results: makeBookmarks(1, 100),
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	bookmarks, err := c.FetchAllBookmarks(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(bookmarks) != 100 {
		t.Errorf("expected 100 bookmarks, got %d", len(bookmarks))
	}
	if requestCount != 1 {
		t.Errorf("expected exactly 1 request, got %d", requestCount)
	}
}

func TestArchive_SendsCorrectRequest(t *testing.T) {
	var gotMethod, gotPath, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	if err := c.Archive(context.Background(), 42); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("expected POST, got %s", gotMethod)
	}
	if gotPath != "/api/bookmarks/42/archive/" {
		t.Errorf("expected /api/bookmarks/42/archive/, got %s", gotPath)
	}
	if gotAuth != "Token test-token" {
		t.Errorf("expected 'Token test-token', got %s", gotAuth)
	}
}

func TestArchive_ReturnsErrorOnFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	if err := c.Archive(context.Background(), 1); err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}
