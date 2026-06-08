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

// TestCase holds a generated Playwright test case.
type TestCase struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Steps       []string `json:"steps"`
	Code        string `json:"code"`
	Selector    string `json:"selector,omitempty"`
}

func genTestsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "gen-tests",
		Short: "Generate test cases from snapshot via Anthropic API",
		RunE:  runGenTests,
	}
	c.Flags().String("out", "out/", "artifact output directory")
	c.Flags().String("skills", "test-gen", "comma-separated skill names")
	c.Flags().String("skills-dir", "skills/", "skills directory")
	return c
}

func runGenTests(cmd *cobra.Command, _ []string) error {
	outDir, _ := cmd.Flags().GetString("out")
	skillNames, _ := cmd.Flags().GetString("skills")
	skillsDir, _ := cmd.Flags().GetString("skills-dir")

	// Read snapshot.
	snapPath := filepath.Join(outDir, "snapshot.json")
	snapData, err := os.ReadFile(snapPath)
	if err != nil {
		return fmt.Errorf("read snapshot: %w", err)
	}

	// Load skills prompt.
	systemPrompt := defaultTestGenPrompt
	for _, skill := range strings.Split(skillNames, ",") {
		skill = strings.TrimSpace(skill)
		if skill == "" {
			continue
		}
		promptPath := filepath.Join(skillsDir, skill+".md")
		if b, err := os.ReadFile(promptPath); err == nil {
			systemPrompt = string(b)
			break
		}
	}

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	client := anthropic.NewClient()

	msg, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude_Sonnet_4_5,
		MaxTokens: 4096,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(string(snapData))),
		},
	})
	if err != nil {
		return fmt.Errorf("anthropic API call: %w", err)
	}

	var responseText string
	for _, block := range msg.Content {
		if block.Type == "text" {
			responseText += block.Text
		}
	}

	// Try to parse as JSON array; otherwise wrap in a single test case.
	var tests []TestCase
	if err := json.Unmarshal([]byte(responseText), &tests); err != nil {
		tests = []TestCase{{
			Name:        "generated-test",
			Description: "AI-generated test",
			Code:        responseText,
		}}
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	outPath := filepath.Join(outDir, "tests.json")
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create tests file: %w", err)
	}
	defer func() { _ = f.Close() }()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(tests); err != nil {
		return fmt.Errorf("encode tests: %w", err)
	}

	fmt.Printf("generated %d test cases → %s\n", len(tests), outPath)
	return nil
}

const defaultTestGenPrompt = `You are an expert QA engineer specialising in Playwright end-to-end testing.
Given a DOM snapshot of a web page, generate a JSON array of test cases.
Each test case must have: name, description, steps (array of strings), and code (a complete Playwright TypeScript test).
Return ONLY valid JSON, no markdown fences.`
