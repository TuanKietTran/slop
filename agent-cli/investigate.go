package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/playwright-community/playwright-go"
	"github.com/spf13/cobra"
)

// Snapshot holds a crawled site snapshot.
type Snapshot struct {
	URL     string   `json:"url"`
	Title   string   `json:"title"`
	DOM     string   `json:"dom"`
	URLs    []string `json:"urls"`
	Links   []string `json:"links"`
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

	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("start playwright: %w", err)
	}
	defer func() { _ = pw.Stop() }()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("launch browser: %w", err)
	}
	defer func() { _ = browser.Close() }()

	page, err := browser.NewPage()
	if err != nil {
		return fmt.Errorf("new page: %w", err)
	}

	if _, err = page.Goto(site, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		return fmt.Errorf("navigate to %s: %w", site, err)
	}

	title, _ := page.Title()

	dom, err := page.Content()
	if err != nil {
		return fmt.Errorf("get page content: %w", err)
	}

	// Collect all href links.
	linkHandles, err := page.QuerySelectorAll("a[href]")
	if err != nil {
		return fmt.Errorf("query links: %w", err)
	}
	var links []string
	for _, h := range linkHandles {
		href, err := h.GetAttribute("href")
		if err == nil && href != "" {
			links = append(links, href)
		}
	}

	snap := &Snapshot{
		URL:   site,
		Title: title,
		DOM:   dom,
		URLs:  []string{site},
		Links: links,
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

	fmt.Printf("snapshot saved to %s (title: %q, links: %d)\n", outPath, title, len(links))
	return nil
}
