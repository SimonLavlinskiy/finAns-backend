package domain

import "context"

// HealthChecker checks database connectivity.
type HealthChecker interface {
	Ping(ctx context.Context) error
}
