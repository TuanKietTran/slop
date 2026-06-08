package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nats-io/nats.go"
	"github.com/spf13/cobra"
)

// ReviewPayload is the bundle sent to the big-review queue.
type ReviewPayload struct {
	TaskID      string          `json:"task_id"`
	TargetSite  string          `json:"target_site"`
	Snapshot    json.RawMessage `json:"snapshot,omitempty"`
	Tests       json.RawMessage `json:"tests,omitempty"`
	Results     json.RawMessage `json:"results,omitempty"`
	SelfReview  json.RawMessage `json:"self_review,omitempty"`
}

func submitCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "submit",
		Short: "Bundle artifacts and publish to the big-review NATS queue",
		RunE:  runSubmit,
	}
	c.Flags().String("out", "out/", "artifact output directory")
	c.Flags().String("to", "review-queue", "NATS subject to publish to")
	c.Flags().String("task-id", "", "task ID (optional)")
	return c
}

func runSubmit(cmd *cobra.Command, _ []string) error {
	outDir, _ := cmd.Flags().GetString("out")
	subject, _ := cmd.Flags().GetString("to")
	taskID, _ := cmd.Flags().GetString("task-id")

	payload := ReviewPayload{TaskID: taskID}

	// Read all artifacts (non-fatal if missing).
	readJSON := func(name string) json.RawMessage {
		b, err := os.ReadFile(filepath.Join(outDir, name))
		if err != nil {
			return nil
		}
		return json.RawMessage(b)
	}

	payload.Snapshot = readJSON("snapshot.json")
	payload.Tests = readJSON("tests.json")
	payload.Results = readJSON("results.json")
	payload.SelfReview = readJSON("review.json")

	// Extract target site from snapshot if available.
	if payload.Snapshot != nil {
		var snap map[string]any
		if err := json.Unmarshal(payload.Snapshot, &snap); err == nil {
			if u, ok := snap["url"].(string); ok {
				payload.TargetSite = u
			}
		}
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}

	nc, err := nats.Connect(natsURL)
	if err != nil {
		return fmt.Errorf("connect to NATS at %s: %w", natsURL, err)
	}
	defer func() { _ = nc.Drain() }()

	if err := nc.Publish(subject, b); err != nil {
		return fmt.Errorf("publish to %s: %w", subject, err)
	}

	fmt.Printf("submitted %d bytes to NATS subject %q\n", len(b), subject)
	return nil
}
