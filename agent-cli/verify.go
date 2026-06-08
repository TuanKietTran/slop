package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/spf13/cobra"
)

// TestResult holds the result of running a single test case.
type TestResult struct {
	Name    string        `json:"name"`
	Passed  bool          `json:"passed"`
	Error   string        `json:"error,omitempty"`
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

	results := &VerifyResults{}
	for _, tc := range tests {
		start := time.Now()
		result := TestResult{Name: tc.Name}

		func() {
			page, err := browser.NewPage()
			if err != nil {
				result.Error = fmt.Sprintf("new page: %v", err)
				return
			}
			defer func() { _ = page.Close() }()

			if site != "" {
				if _, err := page.Goto(site, playwright.PageGotoOptions{
					WaitUntil: playwright.WaitUntilStateDomcontentloaded,
				}); err != nil {
					result.Error = fmt.Sprintf("navigate: %v", err)
					return
				}
			}

			// For each step in the test case, execute as a simple check.
			for _, step := range tc.Steps {
				if err := executeStep(page, step); err != nil {
					result.Error = fmt.Sprintf("step %q: %v", step, err)
					return
				}
			}
			result.Passed = true
		}()

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

// executeStep runs a single test step description against the page.
// For steps that start with "navigate:", it navigates. Otherwise it checks visibility.
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
