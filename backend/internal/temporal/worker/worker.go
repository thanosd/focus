// Package worker wires workflows and activities into a Temporal worker.
package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/thanosd/focus/backend/internal/ports"
	"github.com/thanosd/focus/backend/internal/temporal/activities"
	"github.com/thanosd/focus/backend/internal/temporal/schedules"
	"github.com/thanosd/focus/backend/internal/temporal/workflows"
)

// Config is the worker's Temporal connection settings.
type Config struct {
	HostPort  string
	Namespace string
}

// Worker wraps the Temporal client and worker.
type Worker struct {
	client client.Client
	worker worker.Worker
}

// New connects to Temporal, ensures the namespace, and registers everything.
func New(cfg Config, sessions ports.SessionRepository, states ports.OAuthStateRepository) (*Worker, error) {
	c, err := client.Dial(client.Options{HostPort: cfg.HostPort, Namespace: cfg.Namespace})
	if err != nil {
		return nil, fmt.Errorf("temporal dial: %w", err)
	}
	if err := ensureNamespace(c, cfg.Namespace); err != nil {
		c.Close()
		return nil, err
	}
	w := worker.New(c, schedules.TaskQueueName, worker.Options{})
	w.RegisterWorkflowWithOptions(workflows.MaintenanceWorkflow, workflow.RegisterOptions{Name: workflows.MaintenanceWorkflowName})
	maint := activities.NewMaintenanceActivities(sessions, states)
	w.RegisterActivity(maint.DeleteExpiredSessionsActivity)
	w.RegisterActivity(maint.DeleteExpiredOAuthStatesActivity)
	return &Worker{client: c, worker: w}, nil
}

// Start ensures schedules exist and runs the worker until interrupted.
func (w *Worker) Start() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := schedules.NewScheduleManager(w.client).EnsureMaintenanceSchedule(ctx); err != nil {
		log.Printf("Warning: failed to ensure maintenance schedule: %v", err)
	} else {
		log.Println("✓ Maintenance schedule verified (daily at 02:00 UTC)")
	}
	return w.worker.Run(worker.InterruptCh())
}

// Stop shuts the worker down.
func (w *Worker) Stop() {
	w.worker.Stop()
	w.client.Close()
}

func ensureNamespace(c client.Client, namespace string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := c.WorkflowService().DescribeNamespace(ctx, &workflowservice.DescribeNamespaceRequest{Namespace: namespace}); err == nil {
		return nil
	}
	log.Printf("Namespace %q not found, registering it", namespace)
	_, err := c.WorkflowService().RegisterNamespace(ctx, &workflowservice.RegisterNamespaceRequest{
		Namespace:                        namespace,
		WorkflowExecutionRetentionPeriod: durationpb.New(7 * 24 * time.Hour),
		Description:                      "Auto-registered namespace for Focus",
	})
	if err != nil {
		return fmt.Errorf("register namespace %q: %w", namespace, err)
	}
	return nil
}
