package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dakotahp/linkding-cleaner/internal/checker"
	"github.com/dakotahp/linkding-cleaner/internal/linkding"
	"github.com/dakotahp/linkding-cleaner/internal/reporter"
	"github.com/mattn/go-isatty"
	"golang.org/x/sync/errgroup"
)

var version = "dev"

type checkResult struct {
	bookmark   linkding.Bookmark
	statusCode int
	err        error
	deadDomain bool
	archiveErr error
}

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
	dryRun := fs.Bool("dry-run", false, "check URLs and report what would be archived without making changes")
	archiveDeadDomains := fs.Bool("archive-dead-domains", false, "also archive bookmarks whose domain no longer exists in DNS")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *showVersion {
		fmt.Fprintln(stdout, version)
		return nil
	}

	if *concurrency < 1 {
		return fmt.Errorf("--concurrency must be at least 1")
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

	start := time.Now()

	// Start progress bar on TTY only.
	var prog *tea.Program
	var progDone chan struct{}
	if len(bookmarks) > 0 && isatty.IsTerminal(os.Stdout.Fd()) {
		prog = tea.NewProgram(newProgressModel(len(bookmarks)))
		progDone = make(chan struct{})
		go func() {
			prog.Run() //nolint:errcheck
			close(progDone)
		}()
	}

	results := make([]checkResult, len(bookmarks))
	var g errgroup.Group
	g.SetLimit(*concurrency)

	for i, bmark := range bookmarks {
		g.Go(func() error {
			checkCtx, cancel := context.WithTimeout(ctx, *timeout)
			defer cancel()

			r := checker.Check(checkCtx, bmark.URL)
			results[i] = checkResult{bookmark: bmark, statusCode: r.StatusCode, err: r.Err}
			if prog != nil {
				prog.Send(progressMsg{})
			}
			return nil
		})
	}

	_ = g.Wait()

	// Wait for progress bar to clear before printing results.
	if progDone != nil {
		<-progDone
	}

	if *archiveDeadDomains {
		dead := confirmDeadDomains(ctx, results, *concurrency, *timeout)
		if dead*2 > len(results) {
			return fmt.Errorf("%d of %d bookmarks have domains that do not resolve; "+
				"this looks like a DNS problem on your side, so nothing was archived", dead, len(results))
		}
	}

	if !*dryRun {
		archiveDead(ctx, client, results, *concurrency)
	}

	var wouldArchive []string
	for _, r := range results {
		switch {
		case r.deadDomain:
			fmt.Fprintf(stdout, "[DNS] %s: no such host\n", r.bookmark.URL)
		case r.err != nil:
			fmt.Fprintf(stdout, "[ERR] %s: %v\n", r.bookmark.URL, r.err)
		default:
			reporter.Render(stdout, r.statusCode, r.bookmark.URL)
		}
		if r.isDead() {
			switch {
			case *dryRun:
				wouldArchive = append(wouldArchive, r.bookmark.URL)
			case r.archiveErr != nil:
				fmt.Fprintf(stdout, "  [archive error] %s: %v\n", r.bookmark.URL, r.archiveErr)
			default:
				fmt.Fprintf(stdout, "  archived: %s\n", r.bookmark.URL)
			}
		}
	}

	if *dryRun {
		if len(wouldArchive) > 0 {
			fmt.Fprintf(stdout, "\nDry run: %d bookmark(s) would be archived:\n", len(wouldArchive))
			for _, url := range wouldArchive {
				fmt.Fprintf(stdout, "  %s\n", url)
			}
		} else {
			fmt.Fprintln(stdout, "\nDry run: no bookmarks would be archived.")
		}
	}

	fmt.Fprintf(stdout, "\nCompleted in %s\n", time.Since(start).Round(time.Millisecond))

	return nil
}

func (r checkResult) isDead() bool {
	if r.err != nil {
		return r.deadDomain
	}
	return r.statusCode == http.StatusNotFound || r.statusCode == http.StatusGone
}

// confirmDeadDomains checks each "no such host" failure a second time and
// marks the ones that fail again as dead. It returns how many it marked.
func confirmDeadDomains(ctx context.Context, results []checkResult, limit int, timeout time.Duration) int {
	var g errgroup.Group
	g.SetLimit(limit)
	var dead atomic.Int32
	for i := range results {
		r := &results[i]
		if !checker.IsDeadDomain(r.err) {
			continue
		}
		g.Go(func() error {
			checkCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			if checker.IsDeadDomain(checker.Check(checkCtx, r.bookmark.URL).Err) {
				r.deadDomain = true
				dead.Add(1)
			}
			return nil
		})
	}
	_ = g.Wait()
	return int(dead.Load())
}

func archiveDead(ctx context.Context, client *linkding.Client, results []checkResult, limit int) {
	var g errgroup.Group
	g.SetLimit(limit)
	for i := range results {
		r := &results[i]
		if !r.isDead() {
			continue
		}
		g.Go(func() error {
			r.archiveErr = client.Archive(ctx, r.bookmark.ID)
			return nil
		})
	}
	_ = g.Wait()
}
