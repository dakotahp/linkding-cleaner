package linkding

import (
	"context"
	"encoding/json"
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
		json.NewEncoder(w).Encode(bookmarksResponse{
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
		json.NewEncoder(w).Encode(bookmarksResponse{Count: 0, Results: []Bookmark{}})
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
