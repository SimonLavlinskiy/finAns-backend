package service

import (
	"context"
	"testing"
)

func TestPlannedExpenseService_ValidateAndBuild_EmptyTitle(t *testing.T) {
	svc := &PlannedExpenseService{}
	_, err := svc.validateAndBuild(context.Background(), 0, "", nil, nil, nil, "medium", nil, nil)
	if err == nil {
		t.Fatal("expected validation error for empty title")
	}
}

func TestPlannedExpenseService_ValidateAndBuild_BadPriority(t *testing.T) {
	svc := &PlannedExpenseService{}
	title := "Наушники"
	_, err := svc.validateAndBuild(context.Background(), 0, title, nil, nil, nil, "urgent", nil, nil)
	if err == nil {
		t.Fatal("expected validation error for invalid priority")
	}
}

func TestPlannedExpenseService_ValidateAndBuild_NegativeCost(t *testing.T) {
	svc := &PlannedExpenseService{}
	cost := int64(-100)
	_, err := svc.validateAndBuild(context.Background(), 0, "Наушники", &cost, nil, nil, "medium", nil, nil)
	if err == nil {
		t.Fatal("expected validation error for negative cost_kopecks")
	}
}

func TestPlannedExpenseService_ValidateAndBuild_BadDueDate(t *testing.T) {
	svc := &PlannedExpenseService{}
	bad := "not-a-date"
	_, err := svc.validateAndBuild(context.Background(), 0, "Наушники", nil, &bad, nil, "medium", nil, nil)
	if err == nil {
		t.Fatal("expected validation error for malformed due_date")
	}
}

func TestPlannedExpenseService_ValidateAndBuild_MissingCategory(t *testing.T) {
	svc := &PlannedExpenseService{}
	_, err := svc.validateAndBuild(context.Background(), 0, "Наушники", nil, nil, nil, "medium", nil, nil)
	if err == nil {
		t.Fatal("expected validation error when neither category_id nor new_category is set")
	}
}

func TestPriorityRank(t *testing.T) {
	if priorityRank("high") <= priorityRank("medium") {
		t.Error("high must outrank medium")
	}
	if priorityRank("medium") <= priorityRank("low") {
		t.Error("medium must outrank low")
	}
}
