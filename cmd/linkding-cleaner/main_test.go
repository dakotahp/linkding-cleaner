package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// fakeBookmark is used only for JSON encoding in test handlers.
type fakeBookmark struct {
	ID           int    `json:"id"`
	URL          string `json:"url"`
	WebsiteTitle string `json:"website_title"`
}

type fakeResponse struct {
	Count   int            `json:"count"`
	Results []fakeBookmark `json:"results"`
}

func TestRun_ArchivesNotFoundBookmarks(t *testing.T) {
	notFoundSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer notFoundSrv.Close()

	okSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer okSrv.Close()

	var mu sync.Mutex
	var archivedIDs []int

	linkdingSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			// Path: /api/bookmarks/{id}/archive/
			parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
			// parts: ["api", "bookmarks", "{id}", "archive"]
			id, _ := strconv.Atoi(parts[2])
			mu.Lock()
			archivedIDs = append(archivedIDs, id)
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(fakeResponse{
			Count: 2,
			Results: []fakeBookmark{
				{ID: 1, URL: notFoundSrv.URL, WebsiteTitle: "Gone"},
				{ID: 2, URL: okSrv.URL, WebsiteTitle: "OK"},
			},
		})
	}))
	defer linkdingSrv.Close()

	var out bytes.Buffer
	err := run([]string{"--url", linkdingSrv.URL, "--token", "test-token", "--concurrency", "2"}, &out)
	if err != nil {
		t.Fatalf("run() error: %v\noutput: %s", err, out.String())
	}

	mu.Lock()
	defer mu.Unlock()
	if len(archivedIDs) != 1 || archivedIDs[0] != 1 {
		t.Errorf("expected bookmark ID 1 archived, got: %v", archivedIDs)
	}
	if !strings.Contains(out.String(), notFoundSrv.URL) {
		t.Errorf("expected not-found URL in output")
	}
}

func TestRun_MultiPageFetchAndCheck(t *testing.T) {
	// Two target servers: first 100 bookmarks → 404, next 50 → 200
	notFoundSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer notFoundSrv.Close()

	okSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer okSrv.Close()

	var mu sync.Mutex
	var archivedCount int

	linkdingSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			mu.Lock()
			archivedCount++
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
			return
		}
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		w.Header().Set("Content-Type", "application/json")
		if offset == 0 {
			bmarks := make([]fakeBookmark, 100)
			for i := range bmarks {
				bmarks[i] = fakeBookmark{ID: i + 1, URL: notFoundSrv.URL}
			}
			_ = json.NewEncoder(w).Encode(fakeResponse{Count: 150, Results: bmarks})
		} else {
			bmarks := make([]fakeBookmark, 50)
			for i := range bmarks {
				bmarks[i] = fakeBookmark{ID: 101 + i, URL: okSrv.URL}
			}
			_ = json.NewEncoder(w).Encode(fakeResponse{Count: 150, Results: bmarks})
		}
	}))
	defer linkdingSrv.Close()

	var out bytes.Buffer
	err := run([]string{"--url", linkdingSrv.URL, "--token", "test-token", "--concurrency", "20"}, &out)
	if err != nil {
		t.Fatalf("run() error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if archivedCount != 100 {
		t.Errorf("expected 100 archives (all 404s), got %d", archivedCount)
	}
}

func TestRun_MissingURL(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"--token", "test-token"}, &out)
	if err == nil {
		t.Fatal("expected error for missing --url, got nil")
	}
}

func TestRun_MissingToken(t *testing.T) {
	t.Setenv("LINKDING_TOKEN", "")
	var out bytes.Buffer
	err := run([]string{"--url", "http://example.com"}, &out)
	if err == nil {
		t.Fatal("expected error for missing token, got nil")
	}
}

func TestRun_TokenFromEnv(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(fakeResponse{Count: 0, Results: []fakeBookmark{}})
	}))
	defer srv.Close()

	t.Setenv("LINKDING_TOKEN", "env-token")
	var out bytes.Buffer
	if err := run([]string{"--url", srv.URL}, &out); err != nil {
		t.Fatalf("run() error: %v", err)
	}
	if gotAuth != "Token env-token" {
		t.Errorf("expected 'Token env-token', got %q", gotAuth)
	}
}
