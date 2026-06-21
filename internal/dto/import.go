package dto

import (
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
)

type ImportBatchResponse struct {
	ID        int64   `json:"id"`
	FileName  string  `json:"file_name"`
	TotalRows int     `json:"total_rows"`
	Status    string  `json:"status"`
	CreatedAt string  `json:"created_at"`
	ClosedAt  *string `json:"closed_at,omitempty"`
}

type ModerationRowResponse struct {
	ID          int64             `json:"id"`
	BatchID     int64             `json:"batch_id"`
	RowNumber   int               `json:"row_number"`
	Title       *string           `json:"title"`
	Amount      *int64            `json:"amount"`
	Date        *string           `json:"date"`
	TagID       *int64            `json:"tag_id"`
	Category    *string           `json:"category"`
	Specificity *string           `json:"specificity"`
	Comment     *string           `json:"comment"`
	URL         *string           `json:"url"`
	Status      string            `json:"status"`
	FieldErrors map[string]string `json:"field_errors"`
}

type ImportBatchWithRowsResponse struct {
	Batch ImportBatchResponse     `json:"batch"`
	Rows  []ModerationRowResponse `json:"rows"`
}

type UpdateModerationRowRequest struct {
	Title       *string `json:"title"`
	Amount      *string `json:"amount"`
	Date        *string `json:"date"`
	TagID       *int64  `json:"tag_id"`
	Category    *string `json:"category"`
	Specificity *string `json:"specificity"`
	Comment     *string `json:"comment"`
	URL         *string `json:"url"`
}

type AcceptBatchRequest struct {
	RowIDs []int64 `json:"row_ids"`
}

type AcceptedTransactionResponse struct {
	ID          int64   `json:"id"`
	Title       string  `json:"title"`
	Amount      int64   `json:"amount"`
	Date        string  `json:"date"`
	TagID       int64   `json:"tag_id"`
	Category    string  `json:"category"`
	Specificity string  `json:"specificity"`
	Comment     *string `json:"comment"`
	URL         *string `json:"url"`
}

func AcceptedTransactionToResponse(t domain.Transaction) AcceptedTransactionResponse {
	return AcceptedTransactionResponse{
		ID:          t.ID,
		Title:       t.Title,
		Amount:      t.Amount,
		Date:        t.Date.Format("2006-01-02"),
		TagID:       t.TagID,
		Category:    t.Category,
		Specificity: t.Specificity,
		Comment:     t.Comment,
		URL:         t.URL,
	}
}

func AcceptedTransactionsToResponse(txs []domain.Transaction) []AcceptedTransactionResponse {
	resp := make([]AcceptedTransactionResponse, 0, len(txs))
	for _, t := range txs {
		resp = append(resp, AcceptedTransactionToResponse(t))
	}
	return resp
}

func ImportBatchToResponse(b domain.ImportBatch) ImportBatchResponse {
	resp := ImportBatchResponse{
		ID:        b.ID,
		FileName:  b.FileName,
		TotalRows: b.TotalRows,
		Status:    b.Status,
		CreatedAt: b.CreatedAt.Format(time.RFC3339),
	}
	if b.ClosedAt != nil {
		closedAt := b.ClosedAt.Format(time.RFC3339)
		resp.ClosedAt = &closedAt
	}
	return resp
}

func ModerationRowToResponse(m domain.ModerationRow) ModerationRowResponse {
	resp := ModerationRowResponse{
		ID:          m.ID,
		BatchID:     m.BatchID,
		RowNumber:   m.RowNumber,
		Title:       m.Title,
		Amount:      m.Amount,
		TagID:       m.TagID,
		Category:    m.Category,
		Specificity: m.Specificity,
		Comment:     m.Comment,
		URL:         m.URL,
		Status:      m.Status,
		FieldErrors: m.FieldErrors,
	}
	if m.Date != nil {
		date := m.Date.Format("2006-01-02")
		resp.Date = &date
	}
	if resp.FieldErrors == nil {
		resp.FieldErrors = map[string]string{}
	}
	return resp
}

func ModerationRowsToResponse(rows []domain.ModerationRow) []ModerationRowResponse {
	resp := make([]ModerationRowResponse, 0, len(rows))
	for _, r := range rows {
		resp = append(resp, ModerationRowToResponse(r))
	}
	return resp
}
