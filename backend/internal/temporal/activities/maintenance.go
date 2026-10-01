// Package activities holds Temporal activity implementations.
package activities

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/activity"

	"github.com/thanosd/focus/backend/internal/ports"
)

// MaintenanceActivities garbage-collect expired auth state.
type MaintenanceActivities struct {
	sessions ports.SessionRepository
	states   ports.OAuthStateRepository
}

// NewMaintenanceActivities constructs the activity set.
func NewMaintenanceActivities(sessions ports.SessionRepository, states ports.OAuthStateRepository) *MaintenanceActivities {
	return &MaintenanceActivities{sessions: sessions, states: states}
}

// DeleteExpiredSessionsActivity removes expired browser sessions.
func (a *MaintenanceActivities) DeleteExpiredSessionsActivity(ctx context.Context) (int64, error) {
	logger := activity.GetLogger(ctx)
	n, err := a.sessions.DeleteExpired(ctx)
	if err != nil {
		return 0, fmt.Errorf("delete expired sessions: %w", err)
	}
	logger.Info("Deleted expired sessions", "count", n)
	return n, nil
}

// DeleteExpiredOAuthStatesActivity removes stale sign-in nonces.
func (a *MaintenanceActivities) DeleteExpiredOAuthStatesActivity(ctx context.Context) (int64, error) {
	logger := activity.GetLogger(ctx)
	n, err := a.states.DeleteExpired(ctx)
	if err != nil {
		return 0, fmt.Errorf("delete expired oauth states: %w", err)
	}
	logger.Info("Deleted expired oauth states", "count", n)
	return n, nil
}
