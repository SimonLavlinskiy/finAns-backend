//go:build integration

// Package itest — интеграционные тесты change analytics-redesign (миграция 000015,
// мягкое удаление тегов, лимиты по категории, аналитика месяца/года, планирование).
package itest

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/SimonLavlinskiy/finAns-backend/internal/repository"
	"github.com/SimonLavlinskiy/finAns-backend/internal/service"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const migrationRedesign = 15

type env struct {
	pool      *pgxpool.Pool
	migrator  *migrate.Migrate
	projectID int64
	simonID   int64
	annaID    int64
}

// setup поднимает Postgres и применяет миграции до version (0 — все).
func setup(t *testing.T, version uint) *env {
	t.Helper()
	ctx := context.Background()

	pg, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("finans"),
		postgres.WithUsername("finans"),
		postgres.WithPassword("finans"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	connStr, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	migrationsPath, err := filepath.Abs(filepath.Join("..", "..", "db", "migrations"))
	require.NoError(t, err)
	m, err := migrate.New("file://"+migrationsPath, connStr)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = m.Close() })
	if version == 0 {
		require.NoError(t, m.Up())
	} else {
		require.NoError(t, m.Migrate(version))
	}

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	e := &env{pool: pool, migrator: m}
	require.NoError(t, pool.QueryRow(ctx, `SELECT id FROM projects WHERE name = 'Мои финансы'`).Scan(&e.projectID))
	require.NoError(t, pool.QueryRow(ctx, `SELECT id FROM users WHERE username = 'simon'`).Scan(&e.simonID))
	require.NoError(t, pool.QueryRow(ctx, `SELECT id FROM users WHERE username = 'anna'`).Scan(&e.annaID))
	return e
}

func (e *env) tag(t *testing.T, name string, parent *int64) int64 {
	t.Helper()
	var id int64
	require.NoError(t, e.pool.QueryRow(context.Background(),
		`INSERT INTO tags (name, color, parent_id, project_id) VALUES ($1, '#112250', $2, $3) RETURNING id`,
		name, parent, e.projectID).Scan(&id))
	return id
}

func (e *env) tx(t *testing.T, date string, amount int64, tagID int64, category, specificity string) {
	t.Helper()
	_, err := e.pool.Exec(context.Background(), `
		INSERT INTO transactions (title, amount, date, tag_id, category, specificity, project_id)
		VALUES ('t', $1, $2, $3, $4::transaction_category, $5::transaction_specificity, $6)`,
		amount, date, tagID, category, specificity, e.projectID)
	require.NoError(t, err)
}

// --- 1. Миграция ---

func TestMigration000015_UpDownUp(t *testing.T) {
	ctx := context.Background()
	e := setup(t, migrationRedesign-1)

	salary := e.tag(t, "Зарплата", nil)
	bonus := e.tag(t, "Премия", &salary)
	food := e.tag(t, "Еда", nil)
	mixed := e.tag(t, "Смешанная", nil)
	e.tx(t, "2026-09-01", 100, bonus, "income", "simple")
	e.tx(t, "2026-09-02", 100, salary, "income", "simple")
	e.tx(t, "2026-09-03", 100, food, "expense", "simple")
	e.tx(t, "2026-09-03", 100, mixed, "expense", "simple")
	e.tx(t, "2026-09-04", 100, mixed, "income", "simple")

	for _, amount := range []int64{1000, 2000} { // дубли месячного лимита
		_, err := e.pool.Exec(ctx, `
			INSERT INTO spending_limits (tag_id, amount, period_type, project_id)
			VALUES ($1, $2, 'month', $3)`, food, amount, e.projectID)
		require.NoError(t, err)
	}

	require.NoError(t, e.migrator.Migrate(migrationRedesign))

	kind := func(id int64) string {
		var k string
		require.NoError(t, e.pool.QueryRow(ctx, `SELECT kind FROM tags WHERE id = $1`, id).Scan(&k))
		return k
	}
	assert.Equal(t, "income", kind(salary), "все операции доходные")
	assert.Equal(t, "income", kind(bonus), "подтег наследует kind")
	assert.Equal(t, "expense", kind(food))
	assert.Equal(t, "expense", kind(mixed), "смешанная категория остаётся расходной")

	var limits int
	var amount int64
	require.NoError(t, e.pool.QueryRow(ctx,
		`SELECT COUNT(*), MAX(amount) FROM spending_limits WHERE tag_id = $1`, food).Scan(&limits, &amount))
	assert.Equal(t, 1, limits, "дубли удалены")
	assert.Equal(t, int64(2000), amount, "остался самый свежий")

	_, err := e.pool.Exec(ctx, `INSERT INTO spending_limits (tag_id, amount, period_type, project_id) VALUES ($1, 1, 'month', $2)`, food, e.projectID)
	assert.Error(t, err, "уникальный индекс месячного лимита")

	require.NoError(t, e.migrator.Steps(-1))
	require.NoError(t, e.pool.QueryRow(ctx, `SELECT COUNT(*) FROM spending_limits WHERE tag_id = $1`, food).Scan(&limits))
	assert.Equal(t, 2, limits, "down восстанавливает удалённые дубли")

	require.NoError(t, e.migrator.Migrate(migrationRedesign))
}

// --- 2. Мягкое удаление тегов ---

func TestTags_SoftDelete(t *testing.T) {
	ctx := context.Background()
	e := setup(t, 0)
	tagRepo := repository.NewTagRepository(e.pool)
	tagSvc := service.NewTagService(tagRepo)
	slSvc := service.NewSpendingLimitService(repository.NewSpendingLimitRepository(e.pool), tagRepo, tagSvc)

	sport, err := tagSvc.Create(ctx, dto.CreateTagRequest{Name: "Спорт"}, e.projectID)
	require.NoError(t, err)
	gym, err := tagSvc.Create(ctx, dto.CreateTagRequest{Name: "Зал", ParentID: &sport.ID}, e.projectID)
	require.NoError(t, err)
	income, err := tagSvc.Create(ctx, dto.CreateTagRequest{Name: "Доход", Kind: "income"}, e.projectID)
	require.NoError(t, err)
	assert.Equal(t, "income", income.Kind)
	assert.Equal(t, "expense", gym.Kind)
	assert.Greater(t, income.Order, sport.Order, "новая категория — в конец списка")

	e.tx(t, "2026-04-10", 500_000, gym.ID, "expense", "simple")
	_, err = slSvc.UpsertForCategory(ctx, sport.ID, 1_000_000, e.projectID)
	require.NoError(t, err)

	var ve *apperrors.ValidationError
	require.ErrorAs(t, tagSvc.Delete(ctx, sport.ID, false), &ve, "без cascade с подтегами нельзя")
	require.NoError(t, tagSvc.Delete(ctx, sport.ID, true))

	alive, err := tagSvc.ListTree(ctx, e.projectID, false)
	require.NoError(t, err)
	for _, c := range alive {
		assert.NotEqual(t, sport.ID, c.ID)
	}
	all, err := tagSvc.ListTree(ctx, e.projectID, true)
	require.NoError(t, err)
	var found bool
	for _, c := range all {
		if c.ID == sport.ID {
			found = true
			assert.NotNil(t, c.DeletedAt)
			require.Len(t, c.Children, 1)
			assert.NotNil(t, c.Children[0].DeletedAt)
		}
	}
	assert.True(t, found, "include_deleted возвращает удалённую категорию")

	var txCount int
	require.NoError(t, e.pool.QueryRow(ctx, `SELECT COUNT(*) FROM transactions WHERE tag_id = $1`, gym.ID).Scan(&txCount))
	assert.Equal(t, 1, txCount, "операции сохраняют ссылку")

	limits, err := slSvc.List(ctx, e.projectID)
	require.NoError(t, err)
	assert.Empty(t, limits, "лимит удалён вместе с категорией")

	tagged, err := tagSvc.TagsWithParentBatch(ctx, []int64{gym.ID})
	require.NoError(t, err)
	assert.Equal(t, "Зал", tagged[gym.ID].Name, "операции показывают название удалённой категории")

	var nf *apperrors.NotFoundError
	assert.ErrorAs(t, tagSvc.Delete(ctx, sport.ID, true), &nf, "повторное удаление")
	assert.Error(t, service.ValidateTagExists(ctx, tagRepo, gym.ID), "удалённый тег нельзя выбрать в формах")
}

// --- 3. Лимиты по категории ---

func TestSpendingLimits_ByCategory(t *testing.T) {
	ctx := context.Background()
	e := setup(t, 0)
	tagRepo := repository.NewTagRepository(e.pool)
	tagSvc := service.NewTagService(tagRepo)
	slSvc := service.NewSpendingLimitService(repository.NewSpendingLimitRepository(e.pool), tagRepo, tagSvc)

	food, err := tagSvc.Create(ctx, dto.CreateTagRequest{Name: "Еда"}, e.projectID)
	require.NoError(t, err)
	cafe, err := tagSvc.Create(ctx, dto.CreateTagRequest{Name: "Кафе", ParentID: &food.ID}, e.projectID)
	require.NoError(t, err)
	income, err := tagSvc.Create(ctx, dto.CreateTagRequest{Name: "Доход", Kind: "income"}, e.projectID)
	require.NoError(t, err)

	_, err = slSvc.UpsertForCategory(ctx, food.ID, 1_000_000, e.projectID)
	require.NoError(t, err)
	l, err := slSvc.UpsertForCategory(ctx, food.ID, 1_200_000, e.projectID)
	require.NoError(t, err)
	assert.Equal(t, int64(1_200_000), l.Amount)
	assert.Equal(t, food.ID, l.CategoryID)

	list, err := slSvc.List(ctx, e.projectID)
	require.NoError(t, err)
	require.Len(t, list, 1, "один лимит на категорию")

	// старый POST с period_type=month тоже заменяет лимит
	_, err = slSvc.Create(ctx, dto.CreateSpendingLimitRequest{TagID: food.ID, Amount: 900_000, PeriodType: "month"}, e.projectID)
	require.NoError(t, err)
	list, err = slSvc.List(ctx, e.projectID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, int64(900_000), list[0].Amount)

	var ve *apperrors.ValidationError
	var nf *apperrors.NotFoundError
	_, err = slSvc.UpsertForCategory(ctx, food.ID, 0, e.projectID)
	assert.ErrorAs(t, err, &ve, "amount=0")
	_, err = slSvc.UpsertForCategory(ctx, income.ID, 100, e.projectID)
	assert.ErrorAs(t, err, &ve, "доходная категория")
	_, err = slSvc.UpsertForCategory(ctx, cafe.ID, 100, e.projectID)
	assert.ErrorAs(t, err, &ve, "подтег")
	_, err = slSvc.UpsertForCategory(ctx, food.ID, 100, e.projectID+999)
	assert.ErrorAs(t, err, &nf, "чужой проект")

	require.NoError(t, slSvc.DeleteForCategory(ctx, food.ID, e.projectID))
	require.NoError(t, slSvc.DeleteForCategory(ctx, food.ID, e.projectID), "повторное удаление идемпотентно")
	list, err = slSvc.List(ctx, e.projectID)
	require.NoError(t, err)
	assert.Empty(t, list)
}

// --- 4. Аналитика месяца, года, диапазона ---

func TestAnalyticsOverview(t *testing.T) {
	ctx := context.Background()
	e := setup(t, 0)
	tagRepo := repository.NewTagRepository(e.pool)
	tagSvc := service.NewTagService(tagRepo)
	clock := service.FixedClock{T: time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)}
	svc := service.NewAnalyticsOverviewService(repository.NewAnalyticsRepository(e.pool), tagRepo, clock)

	mustTag := func(req dto.CreateTagRequest) dto.TagResponse {
		tg, err := tagSvc.Create(ctx, req, e.projectID)
		require.NoError(t, err)
		return tg
	}
	salary := mustTag(dto.CreateTagRequest{Name: "Зарплата", Kind: "income"})
	food := mustTag(dto.CreateTagRequest{Name: "Еда"})
	cafe := mustTag(dto.CreateTagRequest{Name: "Кафе", ParentID: &food.ID})
	rent := mustTag(dto.CreateTagRequest{Name: "Аренда"})
	sport := mustTag(dto.CreateTagRequest{Name: "Спорт"})

	e.tx(t, "2026-09-05", 12_000_000, salary.ID, "income", "simple")
	e.tx(t, "2026-09-06", 2_000_000, food.ID, "expense", "simple")
	e.tx(t, "2026-09-07", 1_000_000, cafe.ID, "expense", "simple")
	e.tx(t, "2026-09-08", 4_000_000, rent.ID, "expense", "required")
	e.tx(t, "2026-04-10", 300_000, sport.ID, "expense", "simple")
	e.tx(t, "2026-10-01", 500_000, food.ID, "expense", "simple")
	e.tx(t, "2025-12-31", 100, food.ID, "expense", "simple")
	e.tx(t, "2026-12-01", 100, food.ID, "expense", "simple") // будущая операция
	require.NoError(t, tagSvc.Delete(ctx, sport.ID, true))

	sep, err := svc.MonthSummary(ctx, "2026-09", e.projectID)
	require.NoError(t, err)
	assert.Equal(t, int64(12_000_000), sep.Income)
	assert.Equal(t, int64(7_000_000), sep.Expense)
	assert.False(t, sep.IsCurrent)
	require.Len(t, sep.Categories, 2)
	assert.Equal(t, rent.ID, sep.Categories[0].CategoryID)
	assert.True(t, sep.Categories[0].Subcategories[0].AllMandatory)
	assert.Equal(t, food.ID, sep.Categories[1].CategoryID)
	assert.Equal(t, int64(3_000_000), sep.Categories[1].Amount)

	apr, err := svc.MonthSummary(ctx, "2026-04", e.projectID)
	require.NoError(t, err)
	require.Len(t, apr.Categories, 1, "удалённая категория остаётся в своём месяце")
	assert.Equal(t, sport.ID, apr.Categories[0].CategoryID)

	oct, err := svc.MonthSummary(ctx, "2026-10", e.projectID)
	require.NoError(t, err)
	assert.True(t, oct.IsCurrent)
	require.NotNil(t, oct.Today)
	assert.Equal(t, 18, *oct.Today)

	var ve *apperrors.ValidationError
	_, err = svc.MonthSummary(ctx, "2026-11", e.projectID)
	assert.ErrorAs(t, err, &ve)

	year, err := svc.YearSummary(ctx, 2026, e.projectID)
	require.NoError(t, err)
	months := []string{}
	for _, m := range year.Months {
		months = append(months, m.Month)
	}
	assert.Equal(t, []string{"2026-04", "2026-09", "2026-10"}, months, "будущий декабрь отброшен")
	assert.True(t, year.Months[2].IsCurrent)

	rng, err := svc.Range(ctx, e.projectID)
	require.NoError(t, err)
	require.NotNil(t, rng.FirstMonth)
	assert.Equal(t, "2025-12", *rng.FirstMonth)
	assert.Equal(t, "2026-10", rng.CurrentMonth)
	assert.Equal(t, []int{2025, 2026}, rng.Years)

	// изоляция проектов
	other, err := svc.MonthSummary(ctx, "2026-09", e.projectID+999)
	require.NoError(t, err)
	assert.Zero(t, other.Income)
	assert.Empty(t, other.Categories)
}

// --- 5. Планирование ---

func TestPlanner_SharedPerProject(t *testing.T) {
	ctx := context.Background()
	e := setup(t, 0)
	svc := service.NewPlannerService(repository.NewPlannerRepository(e.pool))

	def, err := svc.Get(ctx, e.projectID)
	require.NoError(t, err)
	assert.Nil(t, def.Days)
	require.Len(t, def.Incomes, 1)
	assert.Nil(t, def.UpdatedAt)

	amount := int64(15_000_000)
	_, err = svc.Save(ctx, dto.PlannerRequest{
		Incomes: []domain.PlannerIncome{{Name: "Зарплата", Amount: &amount}},
		Plans:   map[string]int64{"1": 2_000_000},
	}, e.projectID, &e.annaID)
	require.NoError(t, err)

	// @simon видит то же, что сохранила @anna
	got, err := svc.Get(ctx, e.projectID)
	require.NoError(t, err)
	assert.Equal(t, "Зарплата", got.Incomes[0].Name)
	assert.Equal(t, int64(2_000_000), got.Plans["1"])
	assert.NotNil(t, got.UpdatedAt)

	// последняя запись побеждает
	days := 28
	_, err = svc.Save(ctx, dto.PlannerRequest{Days: &days}, e.projectID, &e.simonID)
	require.NoError(t, err)
	got, err = svc.Get(ctx, e.projectID)
	require.NoError(t, err)
	assert.Equal(t, 28, *got.Days)
	assert.Empty(t, got.Plans)

	var updatedBy int64
	require.NoError(t, e.pool.QueryRow(ctx, `SELECT updated_by FROM planners WHERE project_id = $1`, e.projectID).Scan(&updatedBy))
	assert.Equal(t, e.simonID, updatedBy)
}

// --- 6. Старые эндпоинты аналитики продолжают работать ---

func TestOldAnalyticsStillWork(t *testing.T) {
	ctx := context.Background()
	e := setup(t, 0)
	tagRepo := repository.NewTagRepository(e.pool)
	tagSvc := service.NewTagService(tagRepo)
	old := service.NewAnalyticsService(
		repository.NewAnalyticsRepository(e.pool), tagRepo,
		repository.NewMandatoryPaymentRepository(e.pool),
		repository.NewBalanceRepository(e.pool),
		repository.NewPlannedExpenseRepository(e.pool),
	)

	food, err := tagSvc.Create(ctx, dto.CreateTagRequest{Name: "Еда"}, e.projectID)
	require.NoError(t, err)
	today := time.Now().UTC()
	e.tx(t, today.Format("2006-01-02"), 100_000, food.ID, "expense", "simple")
	require.NoError(t, tagSvc.Delete(ctx, food.ID, true)) // удалённая категория не ломает старые виджеты

	_, err = old.GetFreedomIndex(ctx, 12, e.projectID)
	assert.NoError(t, err, "freedom-index")
	_, err = old.GetFixedCostRatio(ctx, today.Year(), int(today.Month()), e.projectID)
	assert.NoError(t, err, "fixed-cost-ratio")
	_, err = old.GetIncomeExpenseSankey(ctx, "month", today.Year(), int(today.Month()), e.projectID)
	assert.NoError(t, err, "sankey")
	_, err = old.GetBalanceForecast(ctx, 30, e.projectID)
	assert.NoError(t, err, "balance-forecast")
	_, err = old.GetSavingsGoals(ctx, e.projectID)
	assert.NoError(t, err, "savings-goals")
	_, err = old.GetInsights(ctx, e.projectID)
	assert.NoError(t, err, "insights")
	_, err = old.GetPeriodComparison(ctx, "prev_month", e.projectID)
	assert.NoError(t, err, "period-comparison")
}

// --- 7. Обновление тега: kind наследуется поддеревом; старый CRUD лимитов ---

func TestTags_UpdateKindAndLegacyLimits(t *testing.T) {
	ctx := context.Background()
	e := setup(t, 0)
	tagRepo := repository.NewTagRepository(e.pool)
	tagSvc := service.NewTagService(tagRepo)
	slSvc := service.NewSpendingLimitService(repository.NewSpendingLimitRepository(e.pool), tagRepo, tagSvc)

	gifts, err := tagSvc.Create(ctx, dto.CreateTagRequest{Name: "Подарки", Color: "#ff0000"}, e.projectID)
	require.NoError(t, err)
	child, err := tagSvc.Create(ctx, dto.CreateTagRequest{Name: "От родителей", ParentID: &gifts.ID}, e.projectID)
	require.NoError(t, err)

	income := "income"
	order := 42
	upd, err := tagSvc.Update(ctx, gifts.ID, dto.UpdateTagRequest{Name: "Подарки мне", Color: "#00ff00", Kind: &income, Order: &order})
	require.NoError(t, err)
	assert.Equal(t, "income", upd.Kind)
	assert.Equal(t, 42, upd.Order)

	got, err := tagRepo.GetByID(ctx, child.ID)
	require.NoError(t, err)
	assert.Equal(t, "income", got.Kind, "подтег наследует kind")

	bad := "savings"
	var ve *apperrors.ValidationError
	_, err = tagSvc.Update(ctx, gifts.ID, dto.UpdateTagRequest{Name: "x", Kind: &bad})
	assert.ErrorAs(t, err, &ve)
	_, err = tagSvc.Create(ctx, dto.CreateTagRequest{Name: "x", Kind: "savings"}, e.projectID)
	assert.ErrorAs(t, err, &ve)

	e.tx(t, "2026-09-01", 100, child.ID, "income", "simple")
	usage, err := tagSvc.GetUsage(ctx, child.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), usage.Count)
	tw, err := tagSvc.TagWithParent(ctx, child.ID)
	require.NoError(t, err)
	require.NotNil(t, tw.Parent)
	assert.Equal(t, "Подарки мне", tw.Parent.Name)

	require.NoError(t, tagSvc.Delete(ctx, child.ID, false))
	var nf *apperrors.NotFoundError
	_, err = tagSvc.Update(ctx, child.ID, dto.UpdateTagRequest{Name: "y"})
	assert.ErrorAs(t, err, &nf, "удалённый тег нельзя редактировать")

	// старый CRUD лимитов по id: недельный и произвольный периоды
	food, err := tagSvc.Create(ctx, dto.CreateTagRequest{Name: "Еда"}, e.projectID)
	require.NoError(t, err)
	week, err := slSvc.Create(ctx, dto.CreateSpendingLimitRequest{TagID: food.ID, Amount: 100, PeriodType: "week"}, e.projectID)
	require.NoError(t, err)
	start, end := "2026-07-01", "2026-07-15"
	upd2, err := slSvc.Update(ctx, week.ID, dto.UpdateSpendingLimitRequest{
		TagID: food.ID, Amount: 200, PeriodType: "custom", PeriodStart: &start, PeriodEnd: &end,
	}, e.projectID)
	require.NoError(t, err)
	assert.Equal(t, "custom", upd2.PeriodType)
	_, err = slSvc.Update(ctx, week.ID, dto.UpdateSpendingLimitRequest{TagID: food.ID, Amount: 1, PeriodType: "custom"}, e.projectID)
	assert.ErrorAs(t, err, &ve, "custom без дат")
	_, err = slSvc.Create(ctx, dto.CreateSpendingLimitRequest{TagID: 0, Amount: 0, PeriodType: "year"}, e.projectID)
	assert.ErrorAs(t, err, &ve)
	require.NoError(t, slSvc.Delete(ctx, week.ID))
	assert.ErrorAs(t, slSvc.Delete(ctx, week.ID), &nf)
}
