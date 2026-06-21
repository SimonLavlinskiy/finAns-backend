package service

import (
	"testing"
	"time"
)

func TestAdvanceDate(t *testing.T) {
	base := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		recurrence string
		wantYear   int
		wantMonth  time.Month
		wantDay    int
	}{
		{"daily", 2026, time.February, 1},
		{"weekly", 2026, time.February, 7},
		// 31 Jan + 1 month = 28 Feb (или 29 в високосный)
		{"monthly", 2026, time.February, 28},
		{"quarterly", 2026, time.April, 30},
		{"semi_annual", 2026, time.July, 31},
		{"yearly", 2027, time.January, 31},
	}

	for _, tt := range tests {
		t.Run(tt.recurrence, func(t *testing.T) {
			got := advanceDate(base, tt.recurrence)
			if got.Year() != tt.wantYear || got.Month() != tt.wantMonth || got.Day() != tt.wantDay {
				t.Errorf("advanceDate(%s): got %v, want %04d-%02d-%02d",
					tt.recurrence, got.Format("2006-01-02"), tt.wantYear, tt.wantMonth, tt.wantDay)
			}
		})
	}
}

func TestAdvanceDate_Unknown(t *testing.T) {
	base := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	got := advanceDate(base, "unknown")
	want := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("advanceDate(unknown): got %v, want %v", got, want)
	}
}

func TestValidateAndBuild_BadDate(t *testing.T) {
	svc := &MandatoryPaymentService{}
	// tagID=0 → fails on "обязательное поле", no DB call needed
	_, err := svc.validateAndBuild(
		nil, "test", 1000, 0, "monthly", "bad-date",
	)
	if err == nil {
		t.Fatal("expected validation error for bad date and missing tag")
	}
}

func TestValidateAndBuild_ZeroAmount(t *testing.T) {
	svc := &MandatoryPaymentService{}
	_, err := svc.validateAndBuild(
		nil, "title", 0, 0, "monthly", "2026-01-01",
	)
	if err == nil {
		t.Fatal("expected validation error for zero amount")
	}
}

func TestValidateAndBuild_EmptyTitle(t *testing.T) {
	svc := &MandatoryPaymentService{}
	_, err := svc.validateAndBuild(
		nil, "", 500, 0, "monthly", "2026-01-01",
	)
	if err == nil {
		t.Fatal("expected validation error for empty title")
	}
}

func TestValidateAndBuild_BadRecurrence(t *testing.T) {
	svc := &MandatoryPaymentService{}
	_, err := svc.validateAndBuild(
		nil, "title", 500, 0, "bad_recurrence", "2026-01-01",
	)
	if err == nil {
		t.Fatal("expected validation error for bad recurrence")
	}
}
