package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"golang.org/x/sync/errgroup"
	"github.com/dakotahp/linkding-cleaner/internal/checker"
	"github.com/dakotahp/linkding-cleaner/internal/linkding"
	"github.com/dakotahp/linkding-cleaner/internal/reporter"
)

var version = "dev"

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil && err != flag.ErrHelp {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("linkding-cleaner", flag.ContinueOnError)
	fs.SetOutput(stdout)

	urlFlag := fs.String("url", "", "linkding base URL, e.g. https://links.example.com (required)")
	tokenFlag := fs.String("token", "", "linkding API token (overrides LINKDING_TOKEN env var)")
	concurrency := fs.Int("concurrency", 10, "number of parallel URL checks")
	timeout := fs.Duration("timeout", 10*time.Second, "per-request timeout for URL checks")
	showVersion := fs.Bool("version", false, "print version and exit")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *showVersion {
		fmt.Fprintln(stdout, version)
		return nil
	}

	if *urlFlag == "" {
		fs.Usage()
		return fmt.Errorf("--url is required")
	}

	token := *tokenFlag
	if token == "" {
		token = os.Getenv("LINKDING_TOKEN")
	}
	if token == "" {
		fs.Usage()
		return fmt.Errorf("--token flag or LINKDING_TOKEN environment variable is required")
	}

	client := linkding.New(*urlFlag, token)

	ctx := context.Background()
	bookmarks, err := client.FetchAllBookmarks(ctx)
	if err != nil {
		return fmt.Errorf("fetching bookmarks: %w", err)
	}

	fmt.Fprintf(stdout, "Checking %d bookmarks (concurrency=%d, timeout=%s)...\n",
		len(bookmarks), *concurrency, *timeout)

	g, gctx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, *concurrency)

	for _, bmark := range bookmarks {
		bmark := bmark
		g.Go(func() error {
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-gctx.Done():
				return gctx.Err()
			}

			checkCtx, cancel := context.WithTimeout(gctx, *timeout)
			defer cancel()

			result := checker.Check(checkCtx, bmark.URL)
			if result.Err != nil {
				fmt.Fprintf(stdout, "[ERR] %s: %v\n", bmark.URL, result.Err)
				return nil
			}

			reporter.Render(stdout, result.StatusCode, bmark.URL)

			if result.StatusCode == http.StatusNotFound {
				if archiveErr := client.Archive(ctx, bmark.ID); archiveErr != nil {
					fmt.Fprintf(stdout, "  [archive error] %s: %v\n", bmark.URL, archiveErr)
				} else {
					fmt.Fprintf(stdout, "  archived: %s\n", bmark.URL)
				}
			}
			return nil
		})
	}

	return g.Wait()
}
