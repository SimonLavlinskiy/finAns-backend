package service

import (
	"testing"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
)

func TestComputeEffectivePriority(t *testing.T) {
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)

	dueDate := func(daysOffset int) *time.Time {
		d := now.AddDate(0, 0, daysOffset)
		return &d
	}

	tests := []struct {
		name         string
		priority     string
		dueDate      *time.Time
		wantPriority string
		wantDueSoon  bool
	}{
		{"no due date keeps manual priority", domain.PlannedExpensePriorityLow, nil, domain.PlannedExpensePriorityLow, false},
		{"due date in the past is not due soon", domain.PlannedExpensePriorityLow, dueDate(-1), domain.PlannedExpensePriorityLow, false},
		{"due date today is due soon", domain.PlannedExpensePriorityLow, dueDate(0), domain.PlannedExpensePriorityHigh, true},
		{"due date in 3 days is due soon (boundary)", domain.PlannedExpensePriorityMedium, dueDate(3), domain.PlannedExpensePriorityHigh, true},
		{"due date in 4 days is not due soon", domain.PlannedExpensePriorityMedium, dueDate(4), domain.PlannedExpensePriorityMedium, false},
		{"already high priority stays high when due soon", domain.PlannedExpensePriorityHigh, dueDate(1), domain.PlannedExpensePriorityHigh, true},
		{"already high priority unaffected when not due soon", domain.PlannedExpensePriorityHigh, dueDate(30), domain.PlannedExpensePriorityHigh, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPriority, gotDueSoon := computeEffectivePriority(tt.priority, tt.dueDate, now)
			if gotPriority != tt.wantPriority {
				t.Errorf("priority = %q; want %q", gotPriority, tt.wantPriority)
			}
			if gotDueSoon != tt.wantDueSoon {
				t.Errorf("isDueSoon = %v; want %v", gotDueSoon, tt.wantDueSoon)
			}
		})
	}
}
