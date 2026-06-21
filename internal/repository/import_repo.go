package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ImportRepository struct {
	pool *pgxpool.Pool
}

func NewImportRepository(pool *pgxpool.Pool) *ImportRepository {
	return &ImportRepository{pool: pool}
}

func (r *ImportRepository) CreateBatch(ctx context.Context, fileName string, totalRows int) (domain.ImportBatch, error) {
	var b domain.ImportBatch
	err := r.pool.QueryRow(ctx, `
		INSERT INTO import_batches (file_name, total_rows, status)
		VALUES ($1, $2, 'open')
		RETURNING id, file_name, total_rows, status::text, created_at, closed_at`,
		fileName, totalRows).
		Scan(&b.ID, &b.FileName, &b.TotalRows, &b.Status, &b.CreatedAt, &b.ClosedAt)
	return b, err
}

func (r *ImportRepository) InsertRows(ctx context.Context, rows []domain.ModerationRow) ([]domain.ModerationRow, error) {
	result := make([]domain.ModerationRow, 0, len(rows))
	for _, row := range rows {
		fieldErrors, err := json.Marshal(row.FieldErrors)
		if err != nil {
			return nil, err
		}
		inserted := row
		err = r.pool.QueryRow(ctx, `
			INSERT INTO moderation_transactions
				(batch_id, row_number, title, amount, date, tag_id, category, specificity, comment, url, status, field_errors)
			VALUES ($1,$2,$3,$4,$5,$6,$7::transaction_category,$8::transaction_specificity,$9,$10,$11::moderation_row_status,$12::jsonb)
			RETURNING id, created_at, updated_at`,
			row.BatchID, row.RowNumber, row.Title, row.Amount, row.Date, row.TagID,
			row.Category, row.Specificity, row.Comment, row.URL, row.Status, fieldErrors).
			Scan(&inserted.ID, &inserted.CreatedAt, &inserted.UpdatedAt)
		if err != nil {
			return nil, err
		}
		result = append(result, inserted)
	}
	return result, nil
}

func (r *ImportRepository) GetActiveBatch(ctx context.Context) (domain.ImportBatch, bool, error) {
	var b domain.ImportBatch
	err := r.pool.QueryRow(ctx, `
		SELECT id, file_name, total_rows, status::text, created_at, closed_at
		FROM import_batches WHERE status = 'open'
		ORDER BY created_at DESC LIMIT 1`).
		Scan(&b.ID, &b.FileName, &b.TotalRows, &b.Status, &b.CreatedAt, &b.ClosedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ImportBatch{}, false, nil
	}
	if err != nil {
		return domain.ImportBatch{}, false, err
	}
	return b, true, nil
}

func (r *ImportRepository) GetBatch(ctx context.Context, id int64) (domain.ImportBatch, error) {
	var b domain.ImportBatch
	err := r.pool.QueryRow(ctx, `
		SELECT id, file_name, total_rows, status::text, created_at, closed_at
		FROM import_batches WHERE id = $1`, id).
		Scan(&b.ID, &b.FileName, &b.TotalRows, &b.Status, &b.CreatedAt, &b.ClosedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ImportBatch{}, &apperrors.NotFoundError{Resource: "import_batch"}
	}
	return b, err
}

func (r *ImportRepository) ListRows(ctx context.Context, batchID int64) ([]domain.ModerationRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, batch_id, row_number, title, amount, date, tag_id,
		       category::text, specificity::text, comment, url, status::text, field_errors, created_at, updated_at
		FROM moderation_transactions WHERE batch_id = $1 ORDER BY row_number`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.ModerationRow
	for rows.Next() {
		row, err := scanModerationRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *ImportRepository) GetRow(ctx context.Context, id int64) (domain.ModerationRow, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, batch_id, row_number, title, amount, date, tag_id,
		       category::text, specificity::text, comment, url, status::text, field_errors, created_at, updated_at
		FROM moderation_transactions WHERE id = $1`, id)
	result, err := scanModerationRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ModerationRow{}, &apperrors.NotFoundError{Resource: "moderation_row"}
	}
	return result, err
}

func (r *ImportRepository) UpdateRow(ctx context.Context, row domain.ModerationRow) (domain.ModerationRow, error) {
	fieldErrors, err := json.Marshal(row.FieldErrors)
	if err != nil {
		return domain.ModerationRow{}, err
	}
	dbRow := r.pool.QueryRow(ctx, `
		UPDATE moderation_transactions SET
			title=$2, amount=$3, date=$4, tag_id=$5,
			category=$6::transaction_category, specificity=$7::transaction_specificity,
			comment=$8, url=$9, status=$10::moderation_row_status, field_errors=$11::jsonb,
			updated_at=NOW()
		WHERE id=$1
		RETURNING id, batch_id, row_number, title, amount, date, tag_id,
		          category::text, specificity::text, comment, url, status::text, field_errors, created_at, updated_at`,
		row.ID, row.Title, row.Amount, row.Date, row.TagID, row.Category, row.Specificity,
		row.Comment, row.URL, row.Status, fieldErrors)
	result, err := scanModerationRow(dbRow)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ModerationRow{}, &apperrors.NotFoundError{Resource: "moderation_row"}
	}
	return result, err
}

// AcceptRows переносит готовые строки модерации в transactions и удаляет их из
// moderation_transactions в одной БД-транзакции (атомарный перенос без статуса "Перенесено").
func (r *ImportRepository) AcceptRows(ctx context.Context, batchID int64, rowIDs []int64) ([]domain.Transaction, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `
		SELECT id, title, amount, date, tag_id, category::text, specificity::text, comment, url
		FROM moderation_transactions
		WHERE batch_id = $1 AND id = ANY($2) AND status = 'ready'
		FOR UPDATE`, batchID, rowIDs)
	if err != nil {
		return nil, err
	}

	type readyRow struct {
		id          int64
		title       string
		amount      int64
		date        time.Time
		tagID       int64
		category    string
		specificity string
		comment     *string
		url         *string
	}
	var ready []readyRow
	for rows.Next() {
		var rr readyRow
		if err := rows.Scan(&rr.id, &rr.title, &rr.amount, &rr.date, &rr.tagID, &rr.category, &rr.specificity, &rr.comment, &rr.url); err != nil {
			rows.Close()
			return nil, err
		}
		ready = append(ready, rr)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	created := make([]domain.Transaction, 0, len(ready))
	acceptedIDs := make([]int64, 0, len(ready))
	for _, rr := range ready {
		var t domain.Transaction
		err := tx.QueryRow(ctx, `
			INSERT INTO transactions (title, amount, date, tag_id, category, specificity, comment, url)
			VALUES ($1,$2,$3,$4,$5::transaction_category,$6::transaction_specificity,$7,$8)
			RETURNING id, title, amount, date, tag_id, category::text, specificity::text,
			          comment, url, created_at, updated_at`,
			rr.title, rr.amount, rr.date, rr.tagID, rr.category, rr.specificity, rr.comment, rr.url).
			Scan(&t.ID, &t.Title, &t.Amount, &t.Date, &t.TagID, &t.Category, &t.Specificity,
				&t.Comment, &t.URL, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, err
		}
		created = append(created, t)
		acceptedIDs = append(acceptedIDs, rr.id)
	}

	if len(acceptedIDs) > 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM moderation_transactions WHERE id = ANY($1)`, acceptedIDs); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return created, nil
}

func (r *ImportRepository) CloseBatch(ctx context.Context, batchID int64) error {
	ct, err := r.pool.Exec(ctx, `
		UPDATE import_batches SET status = 'closed', closed_at = NOW() WHERE id = $1`, batchID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return &apperrors.NotFoundError{Resource: "import_batch"}
	}
	return nil
}

func scanModerationRow(row pgx.Row) (domain.ModerationRow, error) {
	var m domain.ModerationRow
	var fieldErrors []byte
	err := row.Scan(
		&m.ID, &m.BatchID, &m.RowNumber, &m.Title, &m.Amount, &m.Date, &m.TagID,
		&m.Category, &m.Specificity, &m.Comment, &m.URL, &m.Status, &fieldErrors,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return domain.ModerationRow{}, err
	}
	if len(fieldErrors) > 0 {
		if err := json.Unmarshal(fieldErrors, &m.FieldErrors); err != nil {
			return domain.ModerationRow{}, err
		}
	}
	return m, nil
}
