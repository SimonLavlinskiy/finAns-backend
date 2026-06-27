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

type MandatoryPaymentRepository struct {
	pool *pgxpool.Pool
}

func NewMandatoryPaymentRepository(pool *pgxpool.Pool) *MandatoryPaymentRepository {
	return &MandatoryPaymentRepository{pool: pool}
}

const mpColumns = `id, title, amount, tag_id, recurrence::text, next_payment_date, created_at, updated_at`

func scanMandatoryPayment(row pgx.Row) (domain.MandatoryPayment, error) {
	var p domain.MandatoryPayment
	err := row.Scan(
		&p.ID, &p.Title, &p.Amount, &p.TagID, &p.Recurrence,
		&p.NextPaymentDate, &p.CreatedAt, &p.UpdatedAt,
	)
	return p, err
}

func (r *MandatoryPaymentRepository) List(ctx context.Context, projectID int64) ([]domain.MandatoryPayment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+mpColumns+`
		FROM mandatory_payments
		WHERE project_id = $1
		ORDER BY next_payment_date ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.MandatoryPayment
	for rows.Next() {
		p, err := scanMandatoryPayment(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (r *MandatoryPaymentRepository) GetByID(ctx context.Context, id int64) (domain.MandatoryPayment, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT `+mpColumns+`
		FROM mandatory_payments WHERE id = $1`, id)
	p, err := scanMandatoryPayment(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.MandatoryPayment{}, &apperrors.NotFoundError{Resource: "mandatory_payment"}
	}
	return p, err
}

func (r *MandatoryPaymentRepository) Create(ctx context.Context, p domain.MandatoryPayment, projectID int64) (domain.MandatoryPayment, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO mandatory_payments (title, amount, tag_id, recurrence, next_payment_date, project_id)
		VALUES ($1, $2, $3, $4::payment_recurrence, $5, $6)
		RETURNING `+mpColumns,
		p.Title, p.Amount, p.TagID, p.Recurrence, p.NextPaymentDate.Format("2006-01-02"), projectID)
	created, err := scanMandatoryPayment(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.MandatoryPayment{}, &apperrors.NotFoundError{Resource: "mandatory_payment"}
	}
	return created, err
}

func (r *MandatoryPaymentRepository) Update(ctx context.Context, p domain.MandatoryPayment) (domain.MandatoryPayment, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE mandatory_payments SET
			title=$2, amount=$3, tag_id=$4, recurrence=$5::payment_recurrence,
			next_payment_date=$6, updated_at=NOW()
		WHERE id=$1
		RETURNING `+mpColumns,
		p.ID, p.Title, p.Amount, p.TagID, p.Recurrence, p.NextPaymentDate.Format("2006-01-02"))
	updated, err := scanMandatoryPayment(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.MandatoryPayment{}, &apperrors.NotFoundError{Resource: "mandatory_payment"}
	}
	return updated, err
}

func (r *MandatoryPaymentRepository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM mandatory_payments WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return &apperrors.NotFoundError{Resource: "mandatory_payment"}
	}
	return nil
}

func (r *MandatoryPaymentRepository) AdvanceDate(ctx context.Context, id int64, newDate time.Time) (domain.MandatoryPayment, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE mandatory_payments
		SET next_payment_date=$2, updated_at=NOW()
		WHERE id=$1
		RETURNING `+mpColumns,
		id, newDate.Format("2006-01-02"))
	updated, err := scanMandatoryPayment(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.MandatoryPayment{}, &apperrors.NotFoundError{Resource: "mandatory_payment"}
	}
	return updated, err
}
