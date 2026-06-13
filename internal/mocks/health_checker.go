package mocks

import "context"

// HealthChecker is a mock of domain.HealthChecker (generated manually for bootstrap).
type HealthChecker struct {
	PingFunc func(ctx context.Context) error
}

func (m *HealthChecker) Ping(ctx context.Context) error {
	if m.PingFunc != nil {
		return m.PingFunc(ctx)
	}
	return nil
}
