package reporter

import (
	"fmt"
	"io"
	"net/http"
)

const (
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorReset  = "\033[0m"
)

// Render writes a status line for the given URL to w.
// 404 is printed in red, 403 in yellow, all others in plain text.
func Render(w io.Writer, statusCode int, url string) {
	switch statusCode {
	case http.StatusNotFound:
		fmt.Fprintf(w, "%s[%d] %s%s\n", colorRed, statusCode, url, colorReset)
	case http.StatusForbidden:
		fmt.Fprintf(w, "%s[%d] %s%s\n", colorYellow, statusCode, url, colorReset)
	default:
		fmt.Fprintf(w, "[%d] %s\n", statusCode, url)
	}
}
