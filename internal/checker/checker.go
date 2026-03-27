package checker

import (
	"context"
	"net/http"
)

// Result holds the outcome of a URL liveness check.
type Result struct {
	StatusCode int
	Err        error
}

// Check tests whether url is reachable. It sends a HEAD request first
// (cheaper, no body) and retries with GET if the server returns 405.
func Check(ctx context.Context, url string) Result {
	code, err := doRequest(ctx, http.MethodHead, url)
	if err != nil {
		return Result{StatusCode: code, Err: err}
	}
	if code == http.StatusMethodNotAllowed {
		code, err = doRequest(ctx, http.MethodGet, url)
	}
	return Result{StatusCode: code, Err: err}
}

func doRequest(ctx context.Context, method, url string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	return resp.StatusCode, nil
}
