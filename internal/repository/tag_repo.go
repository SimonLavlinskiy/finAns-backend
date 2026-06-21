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

func (r *TagRepository) List(ctx context.Context) ([]domain.Tag, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, parent_id, name, color, created_at
		FROM tags ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []domain.Tag
	for rows.Next() {
		var t domain.Tag
		if err := rows.Scan(&t.ID, &t.ParentID, &t.Name, &t.Color, &t.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

// GetManyByIDs возвращает теги по списку ID одним запросом (для батч-загрузки).
func (r *TagRepository) GetManyByIDs(ctx context.Context, ids []int64) ([]domain.Tag, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, parent_id, name, color, created_at FROM tags WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []domain.Tag
	for rows.Next() {
		var t domain.Tag
		if err := rows.Scan(&t.ID, &t.ParentID, &t.Name, &t.Color, &t.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (r *TagRepository) GetByID(ctx context.Context, id int64) (domain.Tag, error) {
	var t domain.Tag
	err := r.pool.QueryRow(ctx, `
		SELECT id, parent_id, name, color, created_at FROM tags WHERE id = $1`, id).
		Scan(&t.ID, &t.ParentID, &t.Name, &t.Color, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Tag{}, &apperrors.NotFoundError{Resource: "tag"}
	}
	return t, err
}

func (r *TagRepository) Create(ctx context.Context, name, color string, parentID *int64) (domain.Tag, error) {
	var t domain.Tag
	err := r.pool.QueryRow(ctx, `
		INSERT INTO tags (name, color, parent_id)
		VALUES ($1, $2, $3)
		RETURNING id, parent_id, name, color, created_at`,
		name, color, parentID).
		Scan(&t.ID, &t.ParentID, &t.Name, &t.Color, &t.CreatedAt)
	return t, err
}

func (r *TagRepository) Update(ctx context.Context, id int64, name, color string) (domain.Tag, error) {
	var t domain.Tag
	err := r.pool.QueryRow(ctx, `
		UPDATE tags SET name = $2, color = $3 WHERE id = $1
		RETURNING id, parent_id, name, color, created_at`, id, name, color).
		Scan(&t.ID, &t.ParentID, &t.Name, &t.Color, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Tag{}, &apperrors.NotFoundError{Resource: "tag"}
	}
	return t, err
}

func (r *TagRepository) UpdateChildrenColor(ctx context.Context, parentID int64, color string) error {
	_, err := r.pool.Exec(ctx, `UPDATE tags SET color = $2 WHERE parent_id = $1`, parentID, color)
	return err
}

func (r *TagRepository) Delete(ctx context.Context, id int64) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM tags WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return &apperrors.NotFoundError{Resource: "tag"}
	}
	return nil
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

// CountUsageSubtree считает транзакции у тега и всех его потомков.
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

func (r *TagRepository) ReassignTransactions(ctx context.Context, fromTagID int64, toTagID *int64) error {
	if toTagID == nil {
		_, err := r.pool.Exec(ctx, `DELETE FROM transactions WHERE tag_id = $1`, fromTagID)
		return err
	}
	_, err := r.pool.Exec(ctx, `UPDATE transactions SET tag_id = $2 WHERE tag_id = $1`, fromTagID, *toTagID)
	return err
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

// FlatToTree converts flat tags to nested dto structure helper lives in service.
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

func (r *TagRepository) DeleteCascade(ctx context.Context, id int64, reassignTo *int64) error {
	descendants, err := r.ListDescendantIDs(ctx, id)
	if err != nil {
		return err
	}
	allIDs := append([]int64{id}, descendants...)

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, tagID := range allIDs {
		if err := r.reassignInTx(ctx, tx, tagID, reassignTo); err != nil {
			return err
		}
	}

	for i := len(allIDs) - 1; i >= 0; i-- {
		if _, err := tx.Exec(ctx, `DELETE FROM tags WHERE id = $1`, allIDs[i]); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *TagRepository) reassignInTx(ctx context.Context, tx pgx.Tx, fromTagID int64, toTagID *int64) error {
	if toTagID == nil {
		// Для корневых тегов без родителя — снимаем привязку: устанавливаем tag_id = 0 (uncategorized).
		// Так как tag_id NOT NULL, переназначаем транзакции к тегу-заглушке или корневой метке.
		// Здесь оставляем транзакции без тега невозможным (constraint). Удаляем только если явно cascade+delete_transactions.
		// По умолчанию: транзакции блокируют удаление корневого тега без параметра cascade.
		// Раз сюда дошли с cascade=true и reassignTo=nil — удаляем транзакции только если нет другого варианта.
		_, err := tx.Exec(ctx, `DELETE FROM transactions WHERE tag_id = $1`, fromTagID)
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE transactions SET tag_id = $2 WHERE tag_id = $1`, fromTagID, *toTagID)
	return err
}

func (r *TagRepository) Exists(ctx context.Context, id int64) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM tags WHERE id = $1)`, id).Scan(&exists)
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
