package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"go.temporal.io/sdk/client"

	tpkg "github.com/tuankiettran/test-agents-control-plane/temporal"
)

const taskQueue = "test-agents-task-queue"

func main() {
	site := flag.String("site", "", "target site URL (required)")
	taskID := flag.String("id", "", "task ID (optional, auto-generated if empty)")
	model := flag.String("model", "claude-sonnet-4-6", "Anthropic model")
	skills := flag.String("skills", "test-gen,verify,code-review", "comma-separated skills")
	flag.Parse()

	if *site == "" {
		fmt.Fprintln(os.Stderr, "usage: client -site <url>")
		os.Exit(1)
	}

	temporalHost := os.Getenv("TEMPORAL_HOST")
	if temporalHost == "" {
		temporalHost = "localhost:7233"
	}

	c, err := client.Dial(client.Options{HostPort: temporalHost})
	if err != nil {
		log.Fatalf("dial temporal: %v", err)
	}
	defer c.Close()

	id := *taskID
	if id == "" {
		id = fmt.Sprintf("task-%d", time.Now().UnixNano())
	}

	t := tpkg.Task{
		ID:         id,
		TargetSite: *site,
		Model:      *model,
		Skills:     splitCSV(*skills),
	}

	opts := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("test-workflow-%s", id),
		TaskQueue: taskQueue,
	}

	run, err := c.ExecuteWorkflow(context.Background(), opts, tpkg.TestWorkflow, t)
	if err != nil {
		log.Fatalf("start workflow: %v", err)
	}

	log.Printf("started workflow: id=%s run=%s", opts.ID, run.GetRunID())

	var result string
	if err := run.Get(context.Background(), &result); err != nil {
		log.Fatalf("workflow failed: %v", err)
	}

	var pretty interface{}
	if err := json.Unmarshal([]byte(result), &pretty); err == nil {
		b, _ := json.MarshalIndent(pretty, "", "  ")
		fmt.Println(string(b))
	} else {
		fmt.Println(result)
	}
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	for _, p := range splitOn(s, ',') {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func splitOn(s string, sep rune) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == sep {
			out = append(out, cur)
			cur = ""
		} else {
			cur += string(r)
		}
	}
	out = append(out, cur)
	return out
}
