package service

import (
	"testing"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func intPtr(v int) *int { return &v }

func TestDefaultPlannerData(t *testing.T) {
	d := DefaultPlannerData()
	assert.Nil(t, d.Days, "дни — авто")
	require.Len(t, d.Incomes, 1, "одна пустая строка дохода")
	assert.Nil(t, d.Incomes[0].Amount)
	assert.Empty(t, d.PaymentOverrides)
	assert.Empty(t, d.Plans)
}

func TestValidatePlanner_OK(t *testing.T) {
	got, err := ValidatePlanner(dto.PlannerRequest{
		Days:             intPtr(30),
		Incomes:          []domain.PlannerIncome{{Name: "  Зарплата ", Amount: ptr64(15_000_000)}},
		PaymentOverrides: map[string]int64{"7": 0},
		Plans:            map[string]int64{"1": 2_000_000},
	})
	require.NoError(t, err)
	assert.Equal(t, "Зарплата", got.Incomes[0].Name)
	assert.Equal(t, int64(0), got.PaymentOverrides["7"], "0 — «не платим в этом месяце»")
	assert.Equal(t, 30, *got.Days)
}

func TestValidatePlanner_EmptyIncomesBecomeDefault(t *testing.T) {
	got, err := ValidatePlanner(dto.PlannerRequest{})
	require.NoError(t, err)
	require.Len(t, got.Incomes, 1)
	assert.NotNil(t, got.Plans)
	assert.NotNil(t, got.PaymentOverrides)
}

func TestValidatePlanner_Errors(t *testing.T) {
	cases := map[string]dto.PlannerRequest{
		"days=0":           {Days: intPtr(0)},
		"days=40":          {Days: intPtr(40)},
		"negative income":  {Incomes: []domain.PlannerIncome{{Name: "x", Amount: ptr64(-1)}}},
		"negative plan":    {Plans: map[string]int64{"1": -100}},
		"bad override key": {PaymentOverrides: map[string]int64{"abc": 100}},
		"zero plan key":    {Plans: map[string]int64{"0": 100}},
		"too many incomes": {Incomes: make([]domain.PlannerIncome, plannerMaxIncomes+1)},
		"too long name":    {Incomes: []domain.PlannerIncome{{Name: string(make([]rune, plannerMaxNameLen+1))}}},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := ValidatePlanner(req)
			var ve *apperrors.ValidationError
			assert.ErrorAs(t, err, &ve)
		})
	}
}
