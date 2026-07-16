package service

import (
	"testing"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
)

func TestPickBestPerTitle_DedupAndAlphabeticalOrder(t *testing.T) {
	items := []domain.TransactionSuggestion{
		{Title: "Сливки", Amount: 100, Count: 1, LastUsed: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Title: "Варенье из слив", Amount: 200, Count: 1, LastUsed: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	got := pickBestPerTitle(items)
	if len(got) != 2 {
		t.Fatalf("expected 2 suggestions, got %d", len(got))
	}
	if got[0].Title != "Варенье из слив" || got[1].Title != "Сливки" {
		t.Errorf("expected alphabetical order, got %v, %v", got[0].Title, got[1].Title)
	}
}

func TestPickBestPerTitle_MostFrequentCombinationWins(t *testing.T) {
	items := []domain.TransactionSuggestion{
		{Title: "Такси", Amount: 300, TagID: 1, Count: 5, LastUsed: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Title: "Такси", Amount: 500, TagID: 2, Count: 8, LastUsed: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)},
	}
	got := pickBestPerTitle(items)
	if len(got) != 1 {
		t.Fatalf("expected 1 suggestion (deduplicated), got %d", len(got))
	}
	if got[0].Amount != 500 || got[0].TagID != 2 {
		t.Errorf("expected the combination with count=8 (amount=500, tag=2), got amount=%d tag=%d", got[0].Amount, got[0].TagID)
	}
}

func TestPickBestPerTitle_TieBreakByLastUsed(t *testing.T) {
	items := []domain.TransactionSuggestion{
		{Title: "Кафе", Amount: 700, TagID: 1, Count: 3, LastUsed: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Title: "Кафе", Amount: 900, TagID: 2, Count: 3, LastUsed: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)},
	}
	got := pickBestPerTitle(items)
	if len(got) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(got))
	}
	if got[0].Amount != 900 {
		t.Errorf("expected the more recently used combination (amount=900) to win the tie, got amount=%d", got[0].Amount)
	}
}
