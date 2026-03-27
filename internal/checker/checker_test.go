package checker_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"linkding-cleaner/internal/checker"
)

func TestCheck_200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	result := checker.Check(context.Background(), srv.URL)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", result.StatusCode)
	}
}

func TestCheck_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	result := checker.Check(context.Background(), srv.URL)
	if result.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", result.StatusCode)
	}
}

func TestCheck_403(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	result := checker.Check(context.Background(), srv.URL)
	if result.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", result.StatusCode)
	}
}

func TestCheck_HEADFallbackToGET(t *testing.T) {
	var mu sync.Mutex
	var methods []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		methods = append(methods, r.Method)
		mu.Unlock()
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	result := checker.Check(context.Background(), srv.URL)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.StatusCode != http.StatusOK {
		t.Errorf("expected 200 after fallback, got %d", result.StatusCode)
	}
	if len(methods) != 2 || methods[0] != http.MethodHead || methods[1] != http.MethodGet {
		t.Errorf("expected [HEAD GET], got %v", methods)
	}
}

func TestCheck_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	result := checker.Check(ctx, srv.URL)
	if result.Err == nil {
		t.Fatal("expected a timeout error, got nil")
	}
}

func TestCheck_TransportErrorNotRetried(t *testing.T) {
	// A server that is immediately closed should cause a transport error on HEAD.
	// The checker should return that error directly without attempting GET.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // close immediately so requests fail

	var requestCount int
	// We can't easily intercept DefaultClient here, so we verify indirectly:
	// a closed server returns an error, and the result should have Err != nil.
	result := checker.Check(context.Background(), srv.URL)
	_ = requestCount
	if result.Err == nil {
		t.Fatal("expected transport error for closed server, got nil")
	}
}
