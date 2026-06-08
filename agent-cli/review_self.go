package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/spf13/cobra"
)

// ReviewResult holds the self-review output.
type ReviewResult struct {
	Summary  string   `json:"summary"`
	Issues   []string `json:"issues"`
	Score    int      `json:"score"`
	Approved bool     `json:"approved"`
}

func reviewSelfCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "review-self",
		Short: "First-pass self review of generated tests and results",
		RunE:  runReviewSelf,
	}
	c.Flags().String("out", "out/", "artifact output directory")
	c.Flags().String("skills-dir", "skills/", "skills directory")
	return c
}

func runReviewSelf(cmd *cobra.Command, _ []string) error {
	outDir, _ := cmd.Flags().GetString("out")
	skillsDir, _ := cmd.Flags().GetString("skills-dir")

	// Load artifacts.
	testsData, err := os.ReadFile(filepath.Join(outDir, "tests.json"))
	if err != nil {
		return fmt.Errorf("read tests: %w", err)
	}
	resultsData, err := os.ReadFile(filepath.Join(outDir, "results.json"))
	if err != nil {
		return fmt.Errorf("read results: %w", err)
	}

	// Load skill prompt.
	systemPrompt := defaultCodeReviewPrompt
	if b, err := os.ReadFile(filepath.Join(skillsDir, "code-review.md")); err == nil {
		systemPrompt = string(b)
	}

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	client := anthropic.NewClient()

	userContent := fmt.Sprintf("Tests:\n%s\n\nResults:\n%s", string(testsData), string(resultsData))

	msg, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.F(anthropic.ModelClaude3_5SonnetLatest),
		MaxTokens: anthropic.F(int64(2048)),
		System: anthropic.F([]anthropic.TextBlockParam{
			anthropic.NewTextBlock(systemPrompt),
		}),
		Messages: anthropic.F([]anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userContent)),
		}),
	})
	if err != nil {
		return fmt.Errorf("anthropic API call: %w", err)
	}

	var responseText string
	for _, block := range msg.Content {
		if block.Type == anthropic.ContentBlockTypeText {
			responseText += block.Text
		}
	}

	// Try to parse as ReviewResult JSON; otherwise create structured result.
	var review ReviewResult
	if err := json.Unmarshal([]byte(responseText), &review); err != nil {
		// Parse text into structured result.
		review = ReviewResult{
			Summary:  responseText,
			Issues:   extractIssues(responseText),
			Score:    75,
			Approved: !strings.Contains(strings.ToLower(responseText), "critical"),
		}
	}

	outPath := filepath.Join(outDir, "review.json")
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create review file: %w", err)
	}
	defer func() { _ = f.Close() }()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(review); err != nil {
		return fmt.Errorf("encode review: %w", err)
	}

	fmt.Printf("self-review complete (score: %d, approved: %v) → %s\n", review.Score, review.Approved, outPath)
	return nil
}

func extractIssues(text string) []string {
	var issues []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") || strings.HasPrefix(line, "•") {
			issues = append(issues, strings.TrimLeft(line, "-*• "))
		}
	}
	return issues
}

const defaultCodeReviewPrompt = `You are a senior QA engineer reviewing generated Playwright tests and their results.
Analyse the tests for correctness, coverage, and quality. Check results for failures.
Return a JSON object with: summary (string), issues (array of strings), score (0-100 integer), approved (boolean).
Return ONLY valid JSON, no markdown fences.`
