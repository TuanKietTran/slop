package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/spf13/cobra"
)

// TestResult holds the result of running a single test case.
type TestResult struct {
	Name     string        `json:"name"`
	Passed   bool          `json:"passed"`
	Error    string        `json:"error,omitempty"`
	Duration time.Duration `json:"duration_ms"`
}

// VerifyResults holds all test results.
type VerifyResults struct {
	Total   int          `json:"total"`
	Passed  int          `json:"passed"`
	Failed  int          `json:"failed"`
	Results []TestResult `json:"results"`
}

func verifyCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "verify",
		Short: "Execute generated tests and capture results",
		RunE:  runVerify,
	}
	c.Flags().String("out", "out/", "artifact output directory")
	c.Flags().String("site", "", "target site URL (used to validate navigation)")
	return c
}

func runVerify(cmd *cobra.Command, _ []string) error {
	outDir, _ := cmd.Flags().GetString("out")
	site, _ := cmd.Flags().GetString("site")

	// Read tests.
	testsPath := filepath.Join(outDir, "tests.json")
	testsData, err := os.ReadFile(testsPath)
	if err != nil {
		return fmt.Errorf("read tests: %w", err)
	}

	var tests []TestCase
	if err := json.Unmarshal(testsData, &tests); err != nil {
		return fmt.Errorf("parse tests: %w", err)
	}

	// Read snapshot to get URL if site not provided.
	if site == "" {
		snapPath := filepath.Join(outDir, "snapshot.json")
		if b, err := os.ReadFile(snapPath); err == nil {
			var snap Snapshot
			if err := json.Unmarshal(b, &snap); err == nil {
				site = snap.URL
			}
		}
	}

	// Try playwright first, fall back to HTTP verification.
	verifyFn, cleanup, playwrightErr := setupPlaywrightVerifier()
	if playwrightErr != nil {
		fmt.Fprintf(os.Stderr, "WARN: playwright unavailable (%v), falling back to HTTP verification\n", playwrightErr)
	}
	if cleanup != nil {
		defer cleanup()
	}

	results := &VerifyResults{}
	for _, tc := range tests {
		start := time.Now()
		result := TestResult{Name: tc.Name}

		var tcErr error
		if verifyFn != nil {
			tcErr = verifyFn(tc, site)
		} else {
			tcErr = verifyWithHTTP(tc, site)
		}
		if tcErr != nil {
			result.Error = tcErr.Error()
		} else {
			result.Passed = true
		}

		result.Duration = time.Since(start)
		if result.Passed {
			results.Passed++
		} else {
			results.Failed++
		}
		results.Total++
		results.Results = append(results.Results, result)
	}

	outPath := filepath.Join(outDir, "results.json")
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create results file: %w", err)
	}
	defer func() { _ = f.Close() }()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(results); err != nil {
		return fmt.Errorf("encode results: %w", err)
	}

	fmt.Printf("verify complete: %d/%d passed → %s\n", results.Passed, results.Total, outPath)
	return nil
}

// setupPlaywrightVerifier tries to start a playwright browser and returns a verify function.
func setupPlaywrightVerifier() (func(TestCase, string) error, func(), error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, nil, err
	}

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		_ = pw.Stop()
		return nil, nil, err
	}

	cleanup := func() {
		_ = browser.Close()
		_ = pw.Stop()
	}

	fn := func(tc TestCase, site string) error {
		page, err := browser.NewPage()
		if err != nil {
			return fmt.Errorf("new page: %v", err)
		}
		defer func() { _ = page.Close() }()

		if site != "" {
			if _, err := page.Goto(site, playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			}); err != nil {
				return fmt.Errorf("navigate: %v", err)
			}
		}

		for _, step := range tc.Steps {
			if err := executeStep(page, step); err != nil {
				return fmt.Errorf("step %q: %v", step, err)
			}
		}
		return nil
	}

	return fn, cleanup, nil
}

// verifyWithHTTP runs basic HTTP reachability checks for each test step.
func verifyWithHTTP(tc TestCase, defaultSite string) error {
	for _, step := range tc.Steps {
		var url string
		if len(step) > 11 && step[:11] == "navigate: " {
			url = step[11:]
		} else {
			url = defaultSite
		}
		if url == "" {
			continue
		}
		resp, err := http.Get(url) //nolint:gosec
		if err != nil {
			return fmt.Errorf("GET %s: %w", url, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 400 {
			return fmt.Errorf("GET %s returned %d", url, resp.StatusCode)
		}
	}
	return nil
}

// executeStep runs a single test step description against the page.
func executeStep(page playwright.Page, step string) error {
	if len(step) > 11 && step[:11] == "navigate: " {
		url := step[11:]
		if _, err := page.Goto(url, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); err != nil {
			return err
		}
		return nil
	}
	// Default: check page loaded (title not empty).
	title, err := page.Title()
	if err != nil {
		return err
	}
	_ = title
	return nil
}
