package service

import (
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
)

const dueSoonWindow = 3 * 24 * time.Hour

// computeEffectivePriority вычисляет приоритет и признак "горящего" срока
// на чтении, без сохранения в БД (design.md, решение 3). Срок считается
// горящим, если due_date не в прошлом и не дальше today+3 дней — в этом
// случае эффективный приоритет принудительно становится "high" независимо
// от вручную заданного значения.
func computeEffectivePriority(priority string, dueDate *time.Time, now time.Time) (string, bool) {
	if dueDate == nil {
		return priority, false
	}

	today := truncateToDate(now)
	due := truncateToDate(*dueDate)

	isDueSoon := !due.Before(today) && !due.After(today.Add(dueSoonWindow))
	if isDueSoon {
		return domain.PlannedExpensePriorityHigh, true
	}
	return priority, false
}

func truncateToDate(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
