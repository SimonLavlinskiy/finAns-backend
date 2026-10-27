package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TagRepository struct {
	pool *pgxpool.Pool
}

func NewTagRepository(pool *pgxpool.Pool) *TagRepository {
	return &TagRepository{pool: pool}
}

const tagColumns = `id, parent_id, name, color, kind, sort_order, created_at, deleted_at`

func scanTag(row pgx.Row) (domain.Tag, error) {
	var t domain.Tag
	err := row.Scan(&t.ID, &t.ParentID, &t.Name, &t.Color, &t.Kind, &t.Order, &t.CreatedAt, &t.DeletedAt)
	return t, err
}

func collectTags(rows pgx.Rows) ([]domain.Tag, error) {
	defer rows.Close()
	var tags []domain.Tag
	for rows.Next() {
		t, err := scanTag(rows)
		if err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

// List возвращает действующие (не удалённые) теги проекта.
func (r *TagRepository) List(ctx context.Context, projectID int64) ([]domain.Tag, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+tagColumns+`
		FROM tags WHERE project_id = $1 AND deleted_at IS NULL
		ORDER BY sort_order, name`, projectID)
	if err != nil {
		return nil, err
	}
	return collectTags(rows)
}

// ListWithDeleted возвращает все теги проекта, включая мягко удалённые.
func (r *TagRepository) ListWithDeleted(ctx context.Context, projectID int64) ([]domain.Tag, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+tagColumns+`
		FROM tags WHERE project_id = $1
		ORDER BY sort_order, name`, projectID)
	if err != nil {
		return nil, err
	}
	return collectTags(rows)
}

// GetManyByIDs loads tags by IDs (no project filter — used for internal batch lookups).
func (r *TagRepository) GetManyByIDs(ctx context.Context, ids []int64) ([]domain.Tag, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT `+tagColumns+` FROM tags WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	return collectTags(rows)
}

func (r *TagRepository) GetByID(ctx context.Context, id int64) (domain.Tag, error) {
	t, err := scanTag(r.pool.QueryRow(ctx, `SELECT `+tagColumns+` FROM tags WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Tag{}, &apperrors.NotFoundError{Resource: "tag"}
	}
	return t, err
}

// Create создаёт тег в конце списка своего уровня (sort_order = max + 1).
func (r *TagRepository) Create(ctx context.Context, name, color string, parentID *int64, projectID int64, kind string) (domain.Tag, error) {
	return scanTag(r.pool.QueryRow(ctx, `
		INSERT INTO tags (name, color, parent_id, project_id, kind, sort_order)
		VALUES ($1, $2, $3, $4, $5, (
			SELECT COALESCE(MAX(sort_order), 0) + 1 FROM tags
			WHERE project_id IS NOT DISTINCT FROM $4 AND parent_id IS NOT DISTINCT FROM $3
		))
		RETURNING `+tagColumns,
		name, color, parentID, projectID, kind))
}

func (r *TagRepository) Update(ctx context.Context, id int64, name, color, kind string, order int) (domain.Tag, error) {
	t, err := scanTag(r.pool.QueryRow(ctx, `
		UPDATE tags SET name = $2, color = $3, kind = $4, sort_order = $5 WHERE id = $1
		RETURNING `+tagColumns, id, name, color, kind, order))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Tag{}, &apperrors.NotFoundError{Resource: "tag"}
	}
	return t, err
}

// UpdateChildrenKind — подтеги наследуют вид корневой категории.
func (r *TagRepository) UpdateChildrenKind(ctx context.Context, parentID int64, kind string) error {
	_, err := r.pool.Exec(ctx, `
		WITH RECURSIVE subtree AS (
			SELECT id FROM tags WHERE parent_id = $1
			UNION ALL
			SELECT t.id FROM tags t JOIN subtree s ON t.parent_id = s.id
		)
		UPDATE tags SET kind = $2 WHERE id IN (SELECT id FROM subtree)`, parentID, kind)
	return err
}

func (r *TagRepository) UpdateChildrenColor(ctx context.Context, parentID int64, color string) error {
	_, err := r.pool.Exec(ctx, `UPDATE tags SET color = $2 WHERE parent_id = $1`, parentID, color)
	return err
}

// SoftDeleteSubtree мягко удаляет тег и всё его поддерево и удаляет лимиты
// этих тегов. Операции сохраняют ссылку на теги.
func (r *TagRepository) SoftDeleteSubtree(ctx context.Context, id int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `
		WITH RECURSIVE subtree AS (
			SELECT id FROM tags WHERE id = $1 AND deleted_at IS NULL
			UNION ALL
			SELECT t.id FROM tags t JOIN subtree s ON t.parent_id = s.id
		)
		UPDATE tags SET deleted_at = NOW()
		WHERE id IN (SELECT id FROM subtree) AND deleted_at IS NULL
		RETURNING id`, id)
	if err != nil {
		return err
	}
	var ids []int64
	for rows.Next() {
		var tagID int64
		if err := rows.Scan(&tagID); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, tagID)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(ids) == 0 {
		return &apperrors.NotFoundError{Resource: "tag"}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM spending_limits WHERE tag_id = ANY($1)`, ids); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *TagRepository) CountUsage(ctx context.Context, id int64) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM transactions WHERE tag_id = $1`, id).Scan(&count)
	return count, err
}

func (r *TagRepository) ListDescendantIDs(ctx context.Context, id int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx, `
		WITH RECURSIVE descendants AS (
			SELECT id FROM tags WHERE parent_id = $1
			UNION ALL
			SELECT t.id FROM tags t JOIN descendants d ON t.parent_id = d.id
		)
		SELECT id FROM descendants`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var childID int64
		if err := rows.Scan(&childID); err != nil {
			return nil, err
		}
		ids = append(ids, childID)
	}
	return ids, rows.Err()
}

// CountUsageSubtree counts transactions for a tag and all its descendants.
func (r *TagRepository) CountUsageSubtree(ctx context.Context, rootID int64) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, `
		WITH RECURSIVE subtree AS (
			SELECT id FROM tags WHERE id = $1
			UNION ALL
			SELECT t.id FROM tags t JOIN subtree s ON t.parent_id = s.id
		)
		SELECT COUNT(*) FROM transactions WHERE tag_id IN (SELECT id FROM subtree)`, rootID).Scan(&count)
	return count, err
}

func BuildTagTree(tags []domain.Tag) []domain.Tag {
	byParent := make(map[int64][]domain.Tag)
	var roots []domain.Tag
	for _, t := range tags {
		if t.ParentID == nil {
			roots = append(roots, t)
		} else {
			byParent[*t.ParentID] = append(byParent[*t.ParentID], t)
		}
	}
	_ = byParent
	return roots
}

func TagTreeIDs(tags []domain.Tag, rootID int64) []int64 {
	children := make(map[int64][]int64)
	for _, t := range tags {
		if t.ParentID != nil {
			children[*t.ParentID] = append(children[*t.ParentID], t.ID)
		}
	}
	var result []int64
	var walk func(int64)
	walk = func(id int64) {
		result = append(result, id)
		for _, cid := range children[id] {
			walk(cid)
		}
	}
	walk(rootID)
	return result
}

// Exists — тег существует и не удалён.
func (r *TagRepository) Exists(ctx context.Context, id int64) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM tags WHERE id = $1 AND deleted_at IS NULL)`, id).Scan(&exists)
	return exists, err
}

func (r *TagRepository) ParentID(ctx context.Context, id int64) (*int64, error) {
	var parentID *int64
	err := r.pool.QueryRow(ctx, `SELECT parent_id FROM tags WHERE id = $1`, id).Scan(&parentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, &apperrors.NotFoundError{Resource: "tag"}
	}
	return parentID, err
}

func (r *TagRepository) ValidateParent(ctx context.Context, parentID int64) error {
	exists, err := r.Exists(ctx, parentID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("parent tag not found")
	}
	return nil
}
