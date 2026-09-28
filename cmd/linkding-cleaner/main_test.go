package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
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

func TestRun_DryRun_DoesNotArchive(t *testing.T) {
	notFoundSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer notFoundSrv.Close()

	var mu sync.Mutex
	var archiveCalled bool
	linkdingSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			mu.Lock()
			archiveCalled = true
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(fakeResponse{
			Count:   1,
			Results: []fakeBookmark{{ID: 1, URL: notFoundSrv.URL}},
		})
	}))
	defer linkdingSrv.Close()

	var out bytes.Buffer
	err := run([]string{"--url", linkdingSrv.URL, "--token", "test-token", "--dry-run"}, &out)
	if err != nil {
		t.Fatalf("run() error: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if archiveCalled {
		t.Error("archive endpoint should not be called in dry-run mode")
	}
	if !strings.Contains(out.String(), "would be archived") {
		t.Errorf("expected dry-run summary in output, got: %s", out.String())
	}
}

func TestRun_DryRun_SummaryLists404URLs(t *testing.T) {
	notFoundSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer notFoundSrv.Close()

	okSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer okSrv.Close()

	linkdingSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(fakeResponse{
			Count: 2,
			Results: []fakeBookmark{
				{ID: 1, URL: notFoundSrv.URL},
				{ID: 2, URL: okSrv.URL},
			},
		})
	}))
	defer linkdingSrv.Close()

	var out bytes.Buffer
	err := run([]string{"--url", linkdingSrv.URL, "--token", "test-token", "--dry-run"}, &out)
	if err != nil {
		t.Fatalf("run() error: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "1 bookmark(s) would be archived") {
		t.Errorf("expected count in summary, got: %s", output)
	}
	if !strings.Contains(output, notFoundSrv.URL) {
		t.Errorf("expected 404 URL in summary, got: %s", output)
	}
	if strings.Contains(output, "archived: ") {
		t.Errorf("expected no 'archived:' lines in dry-run output, got: %s", output)
	}
}

func TestRun_DryRun_NoDeadLinks(t *testing.T) {
	okSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer okSrv.Close()

	linkdingSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(fakeResponse{
			Count:   1,
			Results: []fakeBookmark{{ID: 1, URL: okSrv.URL}},
		})
	}))
	defer linkdingSrv.Close()

	var out bytes.Buffer
	err := run([]string{"--url", linkdingSrv.URL, "--token", "test-token", "--dry-run"}, &out)
	if err != nil {
		t.Fatalf("run() error: %v", err)
	}
	if !strings.Contains(out.String(), "no bookmarks would be archived") {
		t.Errorf("expected 'no bookmarks would be archived', got: %s", out.String())
	}
}

func TestRun_ElapsedTimeAlwaysPrinted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(fakeResponse{Count: 0, Results: []fakeBookmark{}})
	}))
	defer srv.Close()

	var out bytes.Buffer
	if err := run([]string{"--url", srv.URL, "--token", "test-token"}, &out); err != nil {
		t.Fatalf("run() error: %v", err)
	}
	if !strings.Contains(out.String(), "Completed in") {
		t.Errorf("expected 'Completed in' in output, got: %s", out.String())
	}
}

func TestRun_ArchivesInParallel(t *testing.T) {
	notFoundSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer notFoundSrv.Close()

	var inFlight, maxInFlight atomic.Int32
	linkdingSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			n := inFlight.Add(1)
			defer inFlight.Add(-1)
			for {
				m := maxInFlight.Load()
				if n <= m || maxInFlight.CompareAndSwap(m, n) {
					break
				}
			}
			time.Sleep(50 * time.Millisecond)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		results := make([]fakeBookmark, 4)
		for i := range results {
			results[i] = fakeBookmark{ID: i + 1, URL: notFoundSrv.URL + "/" + strconv.Itoa(i+1)}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(fakeResponse{Count: len(results), Results: results})
	}))
	defer linkdingSrv.Close()

	var out bytes.Buffer
	err := run([]string{"--url", linkdingSrv.URL, "--token", "test-token", "--concurrency", "4"}, &out)
	if err != nil {
		t.Fatalf("run() error: %v\noutput: %s", err, out.String())
	}
	if got := maxInFlight.Load(); got < 2 {
		t.Errorf("expected archive calls to run in parallel, max in flight was %d", got)
	}
	if n := strings.Count(out.String(), "archived:"); n != 4 {
		t.Errorf("expected 4 archived lines, got %d\noutput: %s", n, out.String())
	}
}

func TestRun_RejectsConcurrencyBelowOne(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"--url", "http://example.com", "--token", "test-token", "--concurrency", "0"}, &out)
	if err == nil || !strings.Contains(err.Error(), "--concurrency") {
		t.Fatalf("expected --concurrency error, got %v", err)
	}
}

const deadDomainURL = "http://linkding-cleaner-test.invalid/"

func newLinkdingServer(t *testing.T, bookmarks []fakeBookmark) (string, func() []int) {
	t.Helper()
	var mu sync.Mutex
	var archived []int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
			id, _ := strconv.Atoi(parts[2])
			mu.Lock()
			archived = append(archived, id)
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(fakeResponse{Count: len(bookmarks), Results: bookmarks})
	}))
	t.Cleanup(srv.Close)
	return srv.URL, func() []int {
		mu.Lock()
		defer mu.Unlock()
		return slices.Sorted(slices.Values(archived))
	}
}

func newStatusServer(t *testing.T, status int) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestRun_ArchivesGoneBookmarks(t *testing.T) {
	baseURL, archived := newLinkdingServer(t, []fakeBookmark{
		{ID: 1, URL: newStatusServer(t, http.StatusGone)},
		{ID: 2, URL: newStatusServer(t, http.StatusOK)},
	})

	var out bytes.Buffer
	if err := run([]string{"--url", baseURL, "--token", "t"}, &out); err != nil {
		t.Fatalf("run() error: %v\noutput: %s", err, out.String())
	}
	if got := archived(); !slices.Equal(got, []int{1}) {
		t.Errorf("expected [1] archived, got %v", got)
	}
}

func TestRun_DeadDomainNotArchivedWithoutFlag(t *testing.T) {
	baseURL, archived := newLinkdingServer(t, []fakeBookmark{
		{ID: 1, URL: deadDomainURL},
		{ID: 2, URL: newStatusServer(t, http.StatusOK)},
		{ID: 3, URL: newStatusServer(t, http.StatusOK)},
	})

	var out bytes.Buffer
	if err := run([]string{"--url", baseURL, "--token", "t"}, &out); err != nil {
		t.Fatalf("run() error: %v\noutput: %s", err, out.String())
	}
	if got := archived(); len(got) != 0 {
		t.Errorf("expected nothing archived, got %v", got)
	}
}

func TestRun_DeadDomainArchivedWithFlag(t *testing.T) {
	baseURL, archived := newLinkdingServer(t, []fakeBookmark{
		{ID: 1, URL: deadDomainURL},
		{ID: 2, URL: newStatusServer(t, http.StatusOK)},
		{ID: 3, URL: newStatusServer(t, http.StatusOK)},
	})

	var out bytes.Buffer
	if err := run([]string{"--url", baseURL, "--token", "t", "--archive-dead-domains"}, &out); err != nil {
		t.Fatalf("run() error: %v\noutput: %s", err, out.String())
	}
	if got := archived(); !slices.Equal(got, []int{1}) {
		t.Errorf("expected [1] archived, got %v", got)
	}
	if !strings.Contains(out.String(), "[DNS] "+deadDomainURL) {
		t.Errorf("expected [DNS] line in output, got: %s", out.String())
	}
}

func TestRun_DeadDomainInDryRunSummary(t *testing.T) {
	baseURL, archived := newLinkdingServer(t, []fakeBookmark{
		{ID: 1, URL: deadDomainURL},
		{ID: 2, URL: newStatusServer(t, http.StatusOK)},
		{ID: 3, URL: newStatusServer(t, http.StatusOK)},
	})

	var out bytes.Buffer
	if err := run([]string{"--url", baseURL, "--token", "t", "--archive-dead-domains", "--dry-run"}, &out); err != nil {
		t.Fatalf("run() error: %v\noutput: %s", err, out.String())
	}
	if got := archived(); len(got) != 0 {
		t.Errorf("expected nothing archived in dry run, got %v", got)
	}
	if !strings.Contains(out.String(), "1 bookmark(s) would be archived") {
		t.Errorf("expected dead domain in dry-run summary, got: %s", out.String())
	}
}

func TestRun_MostlyDeadDomainsArchivesNothing(t *testing.T) {
	baseURL, archived := newLinkdingServer(t, []fakeBookmark{
		{ID: 1, URL: deadDomainURL + "a"},
		{ID: 2, URL: deadDomainURL + "b"},
		{ID: 3, URL: newStatusServer(t, http.StatusNotFound)},
	})

	var out bytes.Buffer
	err := run([]string{"--url", baseURL, "--token", "t", "--archive-dead-domains"}, &out)
	if err == nil || !strings.Contains(err.Error(), "nothing was archived") {
		t.Fatalf("expected abort error, got %v\noutput: %s", err, out.String())
	}
	if got := archived(); len(got) != 0 {
		t.Errorf("expected nothing archived, got %v", got)
	}
}
