package temporal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"go.temporal.io/sdk/activity"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Activities holds shared dependencies for all activity implementations.
type Activities struct {
	K8sClient  kubernetes.Interface
	IngressURL string
}

// NewActivities creates an Activities instance from environment.
func NewActivities() (*Activities, error) {
	kubecfg := os.Getenv("KUBECONFIG")
	if kubecfg == "" {
		kubecfg = os.Getenv("HOME") + "/.kube/config"
	}
	cfg, err := clientcmd.BuildConfigFromFlags("", kubecfg)
	if err != nil {
		return nil, fmt.Errorf("build k8s config: %w", err)
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("build k8s client: %w", err)
	}
	ingressURL := os.Getenv("INGRESS_URL")
	if ingressURL == "" {
		ingressURL = "http://localhost:8080"
	}
	return &Activities{K8sClient: cs, IngressURL: ingressURL}, nil
}

// ProvisionDevcontainer creates a K8s Job to run the agent pipeline.
func (a *Activities) ProvisionDevcontainer(ctx context.Context, t Task) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Provisioning devcontainer", "task", t.ID)

	ttlSeconds := int32(3600)
	backoffLimit := int32(2)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("agent-%s", t.ID[:8]),
			Namespace: "test-agents",
			Labels:    map[string]string{"task-id": t.ID},
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: &ttlSeconds,
			BackoffLimit:            &backoffLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"task-id": t.ID},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "agent",
							Image: "harbor.local/agent-runner:latest",
							Command: []string{
								"sh", "-c",
								fmt.Sprintf(
									"agent investigate --site %q && agent gen-tests && agent verify && agent review-self && agent submit --task-id %q",
									t.TargetSite, t.ID,
								),
							},
							Env: []corev1.EnvVar{
								{Name: "ANTHROPIC_API_KEY", ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{Name: "agent-secrets"},
										Key:                  "anthropic-api-key",
									},
								}},
								{Name: "NATS_URL", Value: os.Getenv("NATS_URL")},
							},
						},
					},
				},
			},
		},
	}

	created, err := a.K8sClient.BatchV1().Jobs("test-agents").Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("create job: %w", err)
	}
	return created.Name, nil
}

// updateTaskStatus posts a status update to the ingress API.
func (a *Activities) updateTaskStatus(ctx context.Context, taskID, status, stage string) error {
	// Best-effort HTTP call to update ingress store.
	b, _ := json.Marshal(map[string]string{"status": status, "stage": stage})
	_ = b
	// In a full impl this would call: POST {IngressURL}/api/tasks/{taskID}/status
	return nil
}

// InvestigateSite signals that the investigate stage is running.
func (a *Activities) InvestigateSite(ctx context.Context, t Task, jobName string) error {
	activity.GetLogger(ctx).Info("InvestigateSite", "task", t.ID, "job", jobName)
	return a.waitForJobStage(ctx, t.ID, "investigate", "Investigate")
}

// GenTestCases signals that the gen-tests stage is running.
func (a *Activities) GenTestCases(ctx context.Context, t Task, jobName string) error {
	activity.GetLogger(ctx).Info("GenTestCases", "task", t.ID)
	return a.waitForJobStage(ctx, t.ID, "gen-tests", "GenTests")
}

// RunVerify signals that the verify stage is running.
func (a *Activities) RunVerify(ctx context.Context, t Task, jobName string) error {
	activity.GetLogger(ctx).Info("RunVerify", "task", t.ID)
	return a.waitForJobStage(ctx, t.ID, "verify", "Verify")
}

// SelfReviewCode signals that the self-review stage is running.
func (a *Activities) SelfReviewCode(ctx context.Context, t Task, jobName string) error {
	activity.GetLogger(ctx).Info("SelfReviewCode", "task", t.ID)
	return a.waitForJobStage(ctx, t.ID, "review-self", "SelfReview")
}

// SubmitToBigReview waits for the review-queue submission and returns a report placeholder.
func (a *Activities) SubmitToBigReview(ctx context.Context, t Task) (string, error) {
	activity.GetLogger(ctx).Info("SubmitToBigReview", "task", t.ID)
	_ = a.updateTaskStatus(ctx, t.ID, "pending-big-review", "BigReview")
	// In a full impl, poll the review-agent API for completion.
	time.Sleep(2 * time.Second)
	return fmt.Sprintf(`{"task_id":%q,"status":"submitted"}`, t.ID), nil
}

// Notify posts the final notification (best-effort).
func (a *Activities) Notify(ctx context.Context, t Task, report string) error {
	activity.GetLogger(ctx).Info("Notify", "task", t.ID, "report_len", len(report))
	return a.updateTaskStatus(ctx, t.ID, "big-review-done", "Notify")
}

// waitForJobStage polls the K8s Job until it progresses (simplified).
func (a *Activities) waitForJobStage(ctx context.Context, taskID, stage, uiStage string) error {
	_ = a.updateTaskStatus(ctx, taskID, "running", uiStage)
	// Real impl would watch Job pod logs / conditions; here we yield briefly.
	time.Sleep(500 * time.Millisecond)
	return nil
}
