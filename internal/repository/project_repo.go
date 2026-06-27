package repository

import (
	"context"
	"errors"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProjectRepository struct {
	pool *pgxpool.Pool
}

func NewProjectRepository(pool *pgxpool.Pool) *ProjectRepository {
	return &ProjectRepository{pool: pool}
}

func (r *ProjectRepository) Create(ctx context.Context, name string, initialBalance int64, startedAt *time.Time) (domain.Project, error) {
	var p domain.Project
	err := r.pool.QueryRow(ctx, `
		INSERT INTO projects (name, initial_balance_kopecks, started_at)
		VALUES ($1, $2, $3)
		RETURNING id, name, initial_balance_kopecks, started_at, created_at`,
		name, initialBalance, startedAt).
		Scan(&p.ID, &p.Name, &p.InitialBalanceKopecks, &p.StartedAt, &p.CreatedAt)
	return p, err
}

func (r *ProjectRepository) Get(ctx context.Context, id int64) (domain.Project, error) {
	var p domain.Project
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, initial_balance_kopecks, started_at, created_at FROM projects WHERE id = $1`, id).
		Scan(&p.ID, &p.Name, &p.InitialBalanceKopecks, &p.StartedAt, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Project{}, &apperrors.NotFoundError{Resource: "project"}
	}
	return p, err
}

func (r *ProjectRepository) ListByUser(ctx context.Context, userID int64) ([]domain.Project, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.id, p.name, p.initial_balance_kopecks, p.started_at, p.created_at
		FROM projects p
		JOIN project_members pm ON pm.project_id = p.id
		WHERE pm.user_id = $1
		ORDER BY p.created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []domain.Project
	for rows.Next() {
		var p domain.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.InitialBalanceKopecks, &p.StartedAt, &p.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (r *ProjectRepository) AddMember(ctx context.Context, projectID, userID int64, role string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO project_members (project_id, user_id, role) VALUES ($1, $2, $3)`,
		projectID, userID, role)
	return err
}

func (r *ProjectRepository) RemoveMember(ctx context.Context, projectID, userID int64) error {
	ct, err := r.pool.Exec(ctx, `
		DELETE FROM project_members WHERE project_id = $1 AND user_id = $2`, projectID, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return &apperrors.NotFoundError{Resource: "project member"}
	}
	return nil
}

func (r *ProjectRepository) GetMember(ctx context.Context, projectID, userID int64) (domain.ProjectMember, error) {
	var m domain.ProjectMember
	err := r.pool.QueryRow(ctx, `
		SELECT project_id, user_id, role FROM project_members WHERE project_id = $1 AND user_id = $2`,
		projectID, userID).
		Scan(&m.ProjectID, &m.UserID, &m.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ProjectMember{}, &apperrors.NotFoundError{Resource: "project member"}
	}
	return m, err
}

func (r *ProjectRepository) ListMembers(ctx context.Context, projectID int64) ([]domain.ProjectMember, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT project_id, user_id, role FROM project_members WHERE project_id = $1`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []domain.ProjectMember
	for rows.Next() {
		var m domain.ProjectMember
		if err := rows.Scan(&m.ProjectID, &m.UserID, &m.Role); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

// ListOrphaned returns projects that have no members — used to auto-join the first user.
func (r *ProjectRepository) ListOrphaned(ctx context.Context) ([]domain.Project, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.id, p.name, p.initial_balance_kopecks, p.started_at, p.created_at
		FROM projects p
		WHERE NOT EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id = p.id)
		ORDER BY p.created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var projects []domain.Project
	for rows.Next() {
		var p domain.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.InitialBalanceKopecks, &p.StartedAt, &p.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (r *ProjectRepository) CountOwners(ctx context.Context, projectID int64) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM project_members WHERE project_id = $1 AND role = 'owner'`, projectID).Scan(&count)
	return count, err
}
