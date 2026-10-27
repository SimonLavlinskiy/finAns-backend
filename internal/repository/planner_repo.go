package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PlannerRepository struct {
	pool *pgxpool.Pool
}

func NewPlannerRepository(pool *pgxpool.Pool) *PlannerRepository {
	return &PlannerRepository{pool: pool}
}

// Get возвращает планирование проекта; found=false, если оно ещё не сохранялось.
func (r *PlannerRepository) Get(ctx context.Context, projectID int64) (p domain.Planner, found bool, err error) {
	var raw []byte
	p.ProjectID = projectID
	err = r.pool.QueryRow(ctx, `
		SELECT data, updated_by, updated_at FROM planners WHERE project_id = $1`, projectID).
		Scan(&raw, &p.UpdatedBy, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, false, nil
	}
	if err != nil {
		return p, false, err
	}
	if err := json.Unmarshal(raw, &p.Data); err != nil {
		return p, false, err
	}
	return p, true, nil
}

// Upsert сохраняет планирование целиком (последняя запись побеждает).
func (r *PlannerRepository) Upsert(ctx context.Context, projectID int64, data domain.PlannerData, userID *int64) (domain.Planner, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return domain.Planner{}, err
	}
	p := domain.Planner{ProjectID: projectID, Data: data}
	err = r.pool.QueryRow(ctx, `
		INSERT INTO planners (project_id, data, updated_by, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (project_id) DO UPDATE
		SET data = EXCLUDED.data, updated_by = EXCLUDED.updated_by, updated_at = NOW()
		RETURNING updated_by, updated_at`, projectID, raw, userID).
		Scan(&p.UpdatedBy, &p.UpdatedAt)
	return p, err
}
