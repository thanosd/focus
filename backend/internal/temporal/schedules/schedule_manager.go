// Package schedules creates and verifies Temporal schedules.
package schedules

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"

	"github.com/thanosd/focus/backend/internal/temporal/workflows"
)

// TaskQueueName is the worker's task queue.
const TaskQueueName = "focus-queue"

const maintenanceScheduleID = "focus-maintenance"

// ScheduleManager manages the app's Temporal schedules.
type ScheduleManager struct{ client client.Client }

// NewScheduleManager constructs a ScheduleManager.
func NewScheduleManager(c client.Client) *ScheduleManager { return &ScheduleManager{client: c} }

// EnsureMaintenanceSchedule creates or updates the nightly maintenance
// schedule (02:00 UTC).
func (sm *ScheduleManager) EnsureMaintenanceSchedule(ctx context.Context) error {
	spec := client.ScheduleSpec{CronExpressions: []string{"0 2 * * *"}}
	action := &client.ScheduleWorkflowAction{
		ID:        "focus-maintenance-workflow",
		Workflow:  workflows.MaintenanceWorkflowName,
		TaskQueue: TaskQueueName,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	_, err := sm.client.ScheduleClient().Create(ctx, client.ScheduleOptions{ID: maintenanceScheduleID, Spec: spec, Action: action})
	if err == nil {
		return nil
	}
	handle := sm.client.ScheduleClient().GetHandle(ctx, maintenanceScheduleID)
	if err := handle.Update(ctx, client.ScheduleUpdateOptions{
		DoUpdate: func(input client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
			input.Description.Schedule.Spec = &spec
			input.Description.Schedule.Action = action
			return &client.ScheduleUpdate{Schedule: &input.Description.Schedule}, nil
		},
	}); err != nil {
		return fmt.Errorf("update maintenance schedule: %w", err)
	}
	return nil
}
