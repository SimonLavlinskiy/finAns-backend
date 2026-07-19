package service

import (
	"testing"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
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

func TestIsPaidCurrentPeriod_StaleAfterPeriodEnds(t *testing.T) {
	past := time.Now().Add(-48 * time.Hour)
	nextPaymentDate := time.Now().Add(-24 * time.Hour) // уже прошла — новый период наступил
	if isPaidCurrentPeriod(&past, nextPaymentDate) {
		t.Error("expected false: next_payment_date уже наступил, статус не должен считаться оплаченным")
	}
}

func TestIsPaidCurrentPeriod_TrueWithinPeriod(t *testing.T) {
	paidAt := time.Now().Add(-1 * time.Hour)
	nextPaymentDate := time.Now().Add(30 * 24 * time.Hour) // в будущем — текущий период ещё не закончился
	if !isPaidCurrentPeriod(&paidAt, nextPaymentDate) {
		t.Error("expected true: платёж оплачен и next_payment_date ещё не наступил")
	}
}

func TestIsPaidCurrentPeriod_NilLastPaidAt(t *testing.T) {
	nextPaymentDate := time.Now().Add(24 * time.Hour)
	if isPaidCurrentPeriod(nil, nextPaymentDate) {
		t.Error("expected false: платёж не был оплачен")
	}
}

func TestResolveMarkPaidAmount_NoOverride(t *testing.T) {
	got := resolveMarkPaidAmount(50000, dto.MarkPaidRequest{Amount: nil})
	if got != 50000 {
		t.Errorf("expected default payment amount 50000, got %d", got)
	}
}

func TestResolveMarkPaidAmount_PositiveOverride(t *testing.T) {
	override := int64(75000)
	got := resolveMarkPaidAmount(50000, dto.MarkPaidRequest{Amount: &override})
	if got != 75000 {
		t.Errorf("expected overridden amount 75000, got %d", got)
	}
}

func TestResolveMarkPaidAmount_ZeroOrNegativeOverrideIgnored(t *testing.T) {
	zero := int64(0)
	if got := resolveMarkPaidAmount(50000, dto.MarkPaidRequest{Amount: &zero}); got != 50000 {
		t.Errorf("expected zero override to be ignored, got %d", got)
	}
	negative := int64(-100)
	if got := resolveMarkPaidAmount(50000, dto.MarkPaidRequest{Amount: &negative}); got != 50000 {
		t.Errorf("expected negative override to be ignored, got %d", got)
	}
}
