// Package workflows holds Temporal workflow definitions.
package workflows

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/thanosd/focus/backend/internal/temporal/activities"
)

// MaintenanceWorkflowName is the registered workflow name used by the schedule.
const MaintenanceWorkflowName = "MaintenanceWorkflow"

// MaintenanceWorkflow runs the nightly housekeeping activities.
func MaintenanceWorkflow(ctx workflow.Context) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	})
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting maintenance workflow")

	var a *activities.MaintenanceActivities
	var sessions, states int64
	if err := workflow.ExecuteActivity(ctx, a.DeleteExpiredSessionsActivity).Get(ctx, &sessions); err != nil {
		return err
	}
	if err := workflow.ExecuteActivity(ctx, a.DeleteExpiredOAuthStatesActivity).Get(ctx, &states); err != nil {
		return err
	}
	logger.Info("Maintenance complete", "sessions", sessions, "oauth_states", states)
	return nil
}
