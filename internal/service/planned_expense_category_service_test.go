package service

import (
	"context"
	"testing"
)

func TestPlannedExpenseCategoryService_ValidateAndCreate_InvalidColor(t *testing.T) {
	svc := &PlannedExpenseCategoryService{}
	_, err := svc.validateAndCreate(context.Background(), "Электроника", "#ABCDEF")
	if err == nil {
		t.Fatal("expected validation error for color outside the fixed palette")
	}
}

func TestPlannedExpenseCategoryService_ValidateAndCreate_EmptyName(t *testing.T) {
	svc := &PlannedExpenseCategoryService{}
	_, err := svc.validateAndCreate(context.Background(), "  ", "#112250")
	if err == nil {
		t.Fatal("expected validation error for blank name")
	}
}
