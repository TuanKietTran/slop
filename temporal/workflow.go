package temporal

// TestWorkflow is the durable saga. One Activity per pipeline stage; each
// retries with backoff. State is queryable for the Nuxt UI / VSCode extension.

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// Task is the input to the TestWorkflow.
type Task struct {
	ID         string
	TargetSite string
	Skills     []string
	Model      string
}

// WorkflowState holds the current queryable state.
type WorkflowState struct {
	Stage  string `json:"stage"`
	Status string `json:"status"`
	Report string `json:"report,omitempty"`
}

func TestWorkflow(ctx workflow.Context, t Task) (string, error) {
	state := WorkflowState{Stage: "starting", Status: "running"}

	// Register query handler for live state introspection.
	if err := workflow.SetQueryHandler(ctx, "state", func() (WorkflowState, error) {
		return state, nil
	}); err != nil {
		return "", err
	}

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 15 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts:        3,
			InitialInterval:        5 * time.Second,
			BackoffCoefficient:     2.0,
			MaximumInterval:        2 * time.Minute,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Stage: provision devcontainer.
	state.Stage = "Provision"
	var devcontainer string
	if err := workflow.ExecuteActivity(ctx, "ProvisionDevcontainer", t).Get(ctx, &devcontainer); err != nil {
		state.Status = "failed"
		return "", err
	}

	// Pipeline stages.
	stages := []struct {
		name     string
		activity string
	}{
		{"Investigate", "InvestigateSite"},
		{"GenTests", "GenTestCases"},
		{"Verify", "RunVerify"},
		{"SelfReview", "SelfReviewCode"},
	}

	for _, s := range stages {
		state.Stage = s.name
		state.Status = "running"
		if err := workflow.ExecuteActivity(ctx, s.activity, t, devcontainer).Get(ctx, nil); err != nil {
			state.Status = "failed"
			return "", err
		}
	}

	// Submit to big review.
	state.Stage = "BigReview"
	state.Status = "running"
	var report string
	if err := workflow.ExecuteActivity(ctx, "SubmitToBigReview", t).Get(ctx, &report); err != nil {
		state.Status = "failed"
		return "", err
	}
	state.Report = report

	// Notify.
	state.Stage = "Notify"
	_ = workflow.ExecuteActivity(ctx, "Notify", t, report).Get(ctx, nil)

	state.Status = "completed"
	return report, nil
}
