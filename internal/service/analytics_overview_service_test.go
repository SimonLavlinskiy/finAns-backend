package service

import (
	"testing"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var overviewNow = time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)

func ptr64(v int64) *int64 { return &v }

func TestParseMonth(t *testing.T) {
	m, err := ParseMonth("2026-09", overviewNow)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), m)

	_, err = ParseMonth("2026-10", overviewNow)
	assert.NoError(t, err, "текущий месяц допустим")

	var ve *apperrors.ValidationError
	for _, raw := range []string{"2026-11", "2026-13", "abc", ""} {
		_, err := ParseMonth(raw, overviewNow)
		assert.ErrorAs(t, err, &ve, raw)
	}
}

func TestBuildMonthSummary_RollsUpSubcategories(t *testing.T) {
	food := int64(1)
	tags := []domain.Tag{
		{ID: 1, Name: "Еда"},
		{ID: 2, Name: "Кафе", ParentID: &food},
		{ID: 3, Name: "Жильё"},
		{ID: 4, Name: "Аренда", ParentID: ptr64(3)},
		{ID: 5, Name: "Доход", Kind: domain.TagKindIncome},
	}
	amounts := []domain.TagCategoryAmount{
		{TagID: 5, Category: "income", Amount: 12_000_000},
		{TagID: 1, Category: "expense", Amount: 2_000_000},
		{TagID: 2, Category: "expense", Amount: 1_000_000},
		{TagID: 4, Category: "expense", Amount: 4_000_000, AllMandatory: true},
		{TagID: 1, Category: "income", Amount: 50_000}, // возврат — доход
	}

	got := buildMonthSummary(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), overviewNow, amounts, tags)

	assert.Equal(t, "2026-09", got.Month)
	assert.False(t, got.IsCurrent)
	assert.Nil(t, got.Today)
	assert.Equal(t, 30, got.DaysInMonth)
	assert.Equal(t, int64(12_050_000), got.Income)
	assert.Equal(t, int64(7_000_000), got.Expense)

	require.Len(t, got.Categories, 2)
	assert.Equal(t, int64(3), got.Categories[0].CategoryID, "по убыванию суммы")
	assert.Equal(t, int64(4_000_000), got.Categories[0].Amount)
	require.Len(t, got.Categories[0].Subcategories, 1)
	assert.Equal(t, int64(4), *got.Categories[0].Subcategories[0].SubcategoryID)
	assert.True(t, got.Categories[0].Subcategories[0].AllMandatory)

	foodCat := got.Categories[1]
	assert.Equal(t, int64(3_000_000), foodCat.Amount)
	require.Len(t, foodCat.Subcategories, 2)
	assert.Nil(t, foodCat.Subcategories[0].SubcategoryID, "операции самой категории — без подкатегории")
	assert.Equal(t, int64(2_000_000), foodCat.Subcategories[0].Amount)
	assert.Equal(t, int64(2), *foodCat.Subcategories[1].SubcategoryID)
	assert.False(t, foodCat.Subcategories[1].AllMandatory)
}

func TestBuildMonthSummary_DeepTreeMapsToFirstLevel(t *testing.T) {
	tags := []domain.Tag{
		{ID: 1, Name: "Транспорт"},
		{ID: 2, Name: "Такси", ParentID: ptr64(1)},
		{ID: 3, Name: "Ночное", ParentID: ptr64(2)},
	}
	amounts := []domain.TagCategoryAmount{{TagID: 3, Category: "expense", Amount: 100}}

	got := buildMonthSummary(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), overviewNow, amounts, tags)

	require.Len(t, got.Categories, 1)
	assert.Equal(t, int64(1), got.Categories[0].CategoryID)
	assert.Equal(t, int64(2), *got.Categories[0].Subcategories[0].SubcategoryID)
}

func TestBuildMonthSummary_CurrentMonthAndEmpty(t *testing.T) {
	got := buildMonthSummary(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), overviewNow, nil, nil)

	assert.True(t, got.IsCurrent)
	require.NotNil(t, got.Today)
	assert.Equal(t, 18, *got.Today)
	assert.Equal(t, 31, got.DaysInMonth)
	assert.Zero(t, got.Income)
	assert.Zero(t, got.Expense)
	assert.NotNil(t, got.Categories)
	assert.Empty(t, got.Categories)
}

func TestBuildYearSummary_CutsFutureAndMarksCurrent(t *testing.T) {
	months := []domain.MonthIncomeExpense{
		{Month: 1, Income: 100, Expense: 50},
		{Month: 10, Income: 40, Expense: 10},
		{Month: 12, Income: 1, Expense: 1}, // будущая операция
	}

	got := buildYearSummary(2026, overviewNow, months)

	require.Len(t, got.Months, 2)
	assert.Equal(t, "2026-01", got.Months[0].Month)
	assert.False(t, got.Months[0].IsCurrent)
	assert.Equal(t, "2026-10", got.Months[1].Month)
	assert.True(t, got.Months[1].IsCurrent)

	past := buildYearSummary(2025, overviewNow, []domain.MonthIncomeExpense{{Month: 12, Income: 1}})
	require.Len(t, past.Months, 1)
	assert.False(t, past.Months[0].IsCurrent)
}

func TestBuildRange(t *testing.T) {
	first := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	got := buildRange(overviewNow, &first, []int{2024, 2026, 2027})

	require.NotNil(t, got.FirstMonth)
	assert.Equal(t, "2026-03", *got.FirstMonth)
	assert.Equal(t, "2026-10", got.CurrentMonth)
	assert.Equal(t, "2026-10-18", got.Today)
	assert.Equal(t, []int{2024, 2026}, got.Years, "будущие годы отбрасываются")

	empty := buildRange(overviewNow, nil, nil)
	assert.Nil(t, empty.FirstMonth)
	assert.Equal(t, []int{}, empty.Years)
}
