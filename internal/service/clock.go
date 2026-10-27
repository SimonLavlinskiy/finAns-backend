package service

import "time"

// Clock — источник «сегодня»; в тестах подменяется фиксированной датой.
type Clock interface {
	Now() time.Time
}

type locationClock struct {
	loc *time.Location
}

// NewClock возвращает часы в заданном часовом поясе (по умолчанию UTC, если пояс не найден).
func NewClock(tz string) Clock {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	return locationClock{loc: loc}
}

func (c locationClock) Now() time.Time { return time.Now().In(c.loc) }

// FixedClock — часы с фиксированным временем.
type FixedClock struct{ T time.Time }

func (c FixedClock) Now() time.Time { return c.T }
