package reporter_test

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/dakotahp/linkding-cleaner/internal/reporter"
)

func TestRender_404_IsRed(t *testing.T) {
	var buf bytes.Buffer
	reporter.Render(&buf, http.StatusNotFound, "https://example.com/gone")
	got := buf.String()
	if !strings.Contains(got, "\033[31m") {
		t.Errorf("expected red ANSI code (\\033[31m) in output, got: %q", got)
	}
	if !strings.Contains(got, "404") {
		t.Errorf("expected status code 404 in output, got: %q", got)
	}
	if !strings.Contains(got, "https://example.com/gone") {
		t.Errorf("expected URL in output, got: %q", got)
	}
}

func TestRender_403_IsYellow(t *testing.T) {
	var buf bytes.Buffer
	reporter.Render(&buf, http.StatusForbidden, "https://example.com/private")
	got := buf.String()
	if !strings.Contains(got, "\033[33m") {
		t.Errorf("expected yellow ANSI code (\\033[33m) in output, got: %q", got)
	}
	if !strings.Contains(got, "403") {
		t.Errorf("expected status code 403 in output, got: %q", got)
	}
}

func TestRender_200_NoColor(t *testing.T) {
	var buf bytes.Buffer
	reporter.Render(&buf, http.StatusOK, "https://example.com/ok")
	got := buf.String()
	if strings.Contains(got, "\033[") {
		t.Errorf("expected no ANSI escape codes for 200, got: %q", got)
	}
	if !strings.Contains(got, "200") {
		t.Errorf("expected status code 200 in output, got: %q", got)
	}
}

func TestRender_OtherStatus_NoColor(t *testing.T) {
	var buf bytes.Buffer
	reporter.Render(&buf, http.StatusInternalServerError, "https://example.com/error")
	got := buf.String()
	if strings.Contains(got, "\033[") {
		t.Errorf("expected no ANSI escape codes for 500, got: %q", got)
	}
	if !strings.Contains(got, "500") {
		t.Errorf("expected status code 500 in output, got: %q", got)
	}
}
