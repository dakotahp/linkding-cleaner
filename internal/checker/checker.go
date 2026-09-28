package checker

import (
	"context"
	"errors"
	"net"
	"net/http"
)

// Some sites block Go's default User-Agent with a 403.
const userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36"

// Result holds the outcome of a URL liveness check.
type Result struct {
	StatusCode int
	Err        error
}

// Check tests whether url is reachable. It sends a HEAD request first
// (cheaper, no body) and retries with GET if HEAD returns an error status,
// because some servers answer HEAD with 404 or 405 for pages that exist.
func Check(ctx context.Context, url string) Result {
	code, err := doRequest(ctx, http.MethodHead, url)
	if err != nil {
		return Result{StatusCode: code, Err: err}
	}
	if code >= http.StatusBadRequest {
		code, err = doRequest(ctx, http.MethodGet, url)
	}
	return Result{StatusCode: code, Err: err}
}

func doRequest(ctx context.Context, method, url string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	return resp.StatusCode, nil
}

// IsDeadDomain reports whether err means the URL's host does not exist in DNS.
// Timeouts and other network errors do not count, because they are often temporary.
func IsDeadDomain(err error) bool {
	var dnsErr *net.DNSError
	return errors.As(err, &dnsErr) && dnsErr.IsNotFound
}
