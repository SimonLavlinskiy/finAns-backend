package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TransactionRepository struct {
	pool *pgxpool.Pool
}

func NewTransactionRepository(pool *pgxpool.Pool) *TransactionRepository {
	return &TransactionRepository{pool: pool}
}

func (r *TransactionRepository) List(ctx context.Context, f domain.TransactionFilters) (domain.ListResult, error) {
	where := []string{"1=1"}
	args := []any{}
	argN := 1

	if f.Search != "" {
		where = append(where, fmt.Sprintf("title ILIKE $%d", argN))
		args = append(args, "%"+f.Search+"%")
		argN++
	}
	if f.AmountMin != nil {
		where = append(where, fmt.Sprintf("amount >= $%d", argN))
		args = append(args, *f.AmountMin)
		argN++
	}
	if f.AmountMax != nil {
		where = append(where, fmt.Sprintf("amount <= $%d", argN))
		args = append(args, *f.AmountMax)
		argN++
	}
	if f.DateFrom != nil {
		where = append(where, fmt.Sprintf("date >= $%d", argN))
		args = append(args, *f.DateFrom)
		argN++
	}
	if f.DateTo != nil {
		where = append(where, fmt.Sprintf("date <= $%d", argN))
		args = append(args, *f.DateTo)
		argN++
	}
	if len(f.TagIDs) > 0 {
		where = append(where, fmt.Sprintf("tag_id = ANY($%d)", argN))
		args = append(args, f.TagIDs)
		argN++
	}
	if f.Category != "" {
		where = append(where, fmt.Sprintf("category = $%d::transaction_category", argN))
		args = append(args, f.Category)
		argN++
	}
	if f.Specificity != "" {
		where = append(where, fmt.Sprintf("specificity = $%d::transaction_specificity", argN))
		args = append(args, f.Specificity)
		argN++
	}

	whereSQL := strings.Join(where, " AND ")

	var total int64
	countQ := fmt.Sprintf("SELECT COUNT(*) FROM transactions WHERE %s", whereSQL)
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return domain.ListResult{}, err
	}

	sortBy := "date"
	switch f.SortBy {
	case "amount", "title", "date":
		sortBy = f.SortBy
	}
	sortOrder := "DESC"
	if strings.EqualFold(f.SortOrder, "asc") {
		sortOrder = "ASC"
	}

	offset := (f.Page - 1) * f.PerPage
	listArgs := append(args, f.PerPage, offset)
	listQ := fmt.Sprintf(`
		SELECT id, title, amount, date, tag_id, category::text, specificity::text,
		       comment, url, file_path, file_name, file_mime_type, created_at, updated_at
		FROM transactions WHERE %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`, whereSQL, sortBy, sortOrder, argN, argN+1)

	rows, err := r.pool.Query(ctx, listQ, listArgs...)
	if err != nil {
		return domain.ListResult{}, err
	}
	defer rows.Close()

	var items []domain.Transaction
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
			return domain.ListResult{}, err
		}
		items = append(items, t)
	}
	return domain.ListResult{Items: items, Total: total}, rows.Err()
}

func (r *TransactionRepository) GetByID(ctx context.Context, id int64) (domain.Transaction, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, title, amount, date, tag_id, category::text, specificity::text,
		       comment, url, file_path, file_name, file_mime_type, created_at, updated_at
		FROM transactions WHERE id = $1`, id)
	t, err := scanTransactionRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Transaction{}, &apperrors.NotFoundError{Resource: "transaction"}
	}
	return t, err
}

func (r *TransactionRepository) Create(ctx context.Context, t domain.Transaction) (domain.Transaction, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO transactions (title, amount, date, tag_id, category, specificity, comment, url,
		                          file_path, file_name, file_mime_type)
		VALUES ($1,$2,$3,$4,$5::transaction_category,$6::transaction_specificity,$7,$8,$9,$10,$11)
		RETURNING id, title, amount, date, tag_id, category::text, specificity::text,
		          comment, url, file_path, file_name, file_mime_type, created_at, updated_at`,
		t.Title, t.Amount, t.Date, t.TagID, t.Category, t.Specificity,
		t.Comment, t.URL, t.FilePath, t.FileName, t.FileMIME)
	return scanTransactionRow(row)
}

func (r *TransactionRepository) Update(ctx context.Context, t domain.Transaction) (domain.Transaction, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE transactions SET
			title=$2, amount=$3, date=$4, tag_id=$5,
			category=$6::transaction_category, specificity=$7::transaction_specificity,
			comment=$8, url=$9, file_path=$10, file_name=$11, file_mime_type=$12,
			updated_at=NOW()
		WHERE id=$1
		RETURNING id, title, amount, date, tag_id, category::text, specificity::text,
		          comment, url, file_path, file_name, file_mime_type, created_at, updated_at`,
		t.ID, t.Title, t.Amount, t.Date, t.TagID, t.Category, t.Specificity,
		t.Comment, t.URL, t.FilePath, t.FileName, t.FileMIME)
	result, err := scanTransactionRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Transaction{}, &apperrors.NotFoundError{Resource: "transaction"}
	}
	return result, err
}

func (r *TransactionRepository) Delete(ctx context.Context, id int64) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM transactions WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return &apperrors.NotFoundError{Resource: "transaction"}
	}
	return nil
}

func (r *TransactionRepository) Suggestions(ctx context.Context, q string, limit int) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT title FROM transactions
		WHERE title ILIKE $1
		ORDER BY title LIMIT $2`, "%"+q+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var titles []string
	for rows.Next() {
		var title string
		if err := rows.Scan(&title); err != nil {
			return nil, err
		}
		titles = append(titles, title)
	}
	return titles, rows.Err()
}

func (r *TransactionRepository) UpdateFile(ctx context.Context, id int64, path, name, mime *string) error {
	ct, err := r.pool.Exec(ctx, `
		UPDATE transactions SET file_path=$2, file_name=$3, file_mime_type=$4, updated_at=NOW()
		WHERE id=$1`, id, path, name, mime)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return &apperrors.NotFoundError{Resource: "transaction"}
	}
	return nil
}

func (r *TransactionRepository) ClearFile(ctx context.Context, id int64) error {
	ct, err := r.pool.Exec(ctx, `
		UPDATE transactions SET file_path=NULL, file_name=NULL, file_mime_type=NULL, updated_at=NOW()
		WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return &apperrors.NotFoundError{Resource: "transaction"}
	}
	return nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanTransaction(rows pgx.Rows) (domain.Transaction, error) {
	var t domain.Transaction
	var date time.Time
	err := rows.Scan(
		&t.ID, &t.Title, &t.Amount, &date, &t.TagID, &t.Category, &t.Specificity,
		&t.Comment, &t.URL, &t.FilePath, &t.FileName, &t.FileMIME, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return domain.Transaction{}, err
	}
	t.Date = date
	return t, nil
}

func scanTransactionRow(row pgx.Row) (domain.Transaction, error) {
	var t domain.Transaction
	var date time.Time
	err := row.Scan(
		&t.ID, &t.Title, &t.Amount, &date, &t.TagID, &t.Category, &t.Specificity,
		&t.Comment, &t.URL, &t.FilePath, &t.FileName, &t.FileMIME, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return domain.Transaction{}, err
	}
	t.Date = date
	return t, nil
}
