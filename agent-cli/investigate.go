package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/playwright-community/playwright-go"
	"github.com/spf13/cobra"
)

// Snapshot holds a crawled site snapshot.
type Snapshot struct {
	URL   string   `json:"url"`
	Title string   `json:"title"`
	DOM   string   `json:"dom"`
	URLs  []string `json:"urls"`
	Links []string `json:"links"`
}

func investigateCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "investigate",
		Short: "Crawl target site and save DOM snapshot",
		RunE:  runInvestigate,
	}
	c.Flags().String("site", "", "target site URL (required)")
	c.Flags().String("out", "out/", "artifact output directory")
	_ = c.MarkFlagRequired("site")
	return c
}

func runInvestigate(cmd *cobra.Command, _ []string) error {
	site, _ := cmd.Flags().GetString("site")
	outDir, _ := cmd.Flags().GetString("out")

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	snap, err := investigateWithPlaywright(site)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARN: playwright unavailable (%v), falling back to HTTP fetch\n", err)
		snap, err = investigateWithHTTP(site)
		if err != nil {
			return fmt.Errorf("investigate via HTTP: %w", err)
		}
	}

	outPath := filepath.Join(outDir, "snapshot.json")
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create snapshot file: %w", err)
	}
	defer func() { _ = f.Close() }()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(snap); err != nil {
		return fmt.Errorf("encode snapshot: %w", err)
	}

	fmt.Printf("snapshot saved to %s (title: %q, links: %d)\n", outPath, snap.Title, len(snap.Links))
	return nil
}

// investigateWithPlaywright uses a headless browser for full DOM capture.
func investigateWithPlaywright(site string) (*Snapshot, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("start playwright: %w", err)
	}
	defer func() { _ = pw.Stop() }()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("launch browser: %w", err)
	}
	defer func() { _ = browser.Close() }()

	page, err := browser.NewPage()
	if err != nil {
		return nil, fmt.Errorf("new page: %w", err)
	}

	if _, err = page.Goto(site, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		return nil, fmt.Errorf("navigate to %s: %w", site, err)
	}

	title, _ := page.Title()
	dom, err := page.Content()
	if err != nil {
		return nil, fmt.Errorf("get page content: %w", err)
	}

	linkHandles, err := page.QuerySelectorAll("a[href]")
	if err != nil {
		return nil, fmt.Errorf("query links: %w", err)
	}
	var links []string
	for _, h := range linkHandles {
		href, err := h.GetAttribute("href")
		if err == nil && href != "" {
			links = append(links, href)
		}
	}

	return &Snapshot{
		URL:   site,
		Title: title,
		DOM:   dom,
		URLs:  []string{site},
		Links: links,
	}, nil
}

// investigateWithHTTP fetches the page source via plain HTTP.
func investigateWithHTTP(site string) (*Snapshot, error) {
	resp, err := http.Get(site) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("http get %s: %w", site, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	dom := string(body)

	title := extractTitle(dom)
	links := extractLinks(dom)

	return &Snapshot{
		URL:   site,
		Title: title,
		DOM:   dom,
		URLs:  []string{site},
		Links: links,
	}, nil
}

// extractTitle pulls the <title> content from raw HTML.
func extractTitle(html string) string {
	lower := strings.ToLower(html)
	start := strings.Index(lower, "<title>")
	if start < 0 {
		return ""
	}
	start += len("<title>")
	end := strings.Index(lower[start:], "</title>")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(html[start : start+end])
}

// extractLinks pulls href values from <a> tags.
func extractLinks(html string) []string {
	var links []string
	lower := strings.ToLower(html)
	search := lower
	offset := 0
	for {
		idx := strings.Index(search, "href=\"")
		if idx < 0 {
			break
		}
		start := offset + idx + len("href=\"")
		rest := html[start:]
		end := strings.IndexByte(rest, '"')
		if end < 0 {
			break
		}
		href := rest[:end]
		if href != "" {
			links = append(links, href)
		}
		advance := idx + len("href=\"") + end + 1
		offset += advance
		search = lower[offset:]
	}
	return links
}
