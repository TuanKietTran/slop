package main

import (
	"log"
	"os"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	tpkg "github.com/tuankiettran/test-agents-control-plane/temporal"
)

const taskQueue = "test-agents-task-queue"

func main() {
	temporalHost := os.Getenv("TEMPORAL_HOST")
	if temporalHost == "" {
		temporalHost = "localhost:7233"
	}

	c, err := client.Dial(client.Options{HostPort: temporalHost})
	if err != nil {
		log.Fatalf("dial temporal: %v", err)
	}
	defer c.Close()

	acts, err := tpkg.NewActivities()
	if err != nil {
		log.Printf("WARN: activities init: %v (some features may be limited)", err)
		acts = &tpkg.Activities{IngressURL: "http://localhost:8080"}
	}

	w := worker.New(c, taskQueue, worker.Options{})
	w.RegisterWorkflow(tpkg.TestWorkflow)
	w.RegisterActivity(acts.ProvisionDevcontainer)
	w.RegisterActivity(acts.InvestigateSite)
	w.RegisterActivity(acts.GenTestCases)
	w.RegisterActivity(acts.RunVerify)
	w.RegisterActivity(acts.SelfReviewCode)
	w.RegisterActivity(acts.SubmitToBigReview)
	w.RegisterActivity(acts.Notify)

	log.Printf("temporal worker starting, task queue: %s", taskQueue)
	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalf("worker error: %v", err)
	}
}
