package handler

import (
	"net/http"

	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/SimonLavlinskiy/finAns-backend/internal/middleware"
	"github.com/SimonLavlinskiy/finAns-backend/internal/service"
	"github.com/SimonLavlinskiy/finAns-backend/pkg/httputil"
	"github.com/go-chi/chi/v5"
)

const maxImportFileSize = 10 << 20 // 10MB

type ImportHandler struct {
	svc *service.ImportService
}

func NewImportHandler(svc *service.ImportService) *ImportHandler {
	return &ImportHandler{svc: svc}
}

// UploadBatch godoc
// @Summary      Upload a CSV file for import moderation
// @Tags         import
// @Accept       multipart/form-data
// @Produce      json
// @Param        file formData file true "CSV file"
// @Success      201 {object} dto.ImportBatchWithRowsResponse
// @Router       /api/v1/import/batches [post]
func (h *ImportHandler) UploadBatch(w http.ResponseWriter, r *http.Request) {
	projectID, ok := middleware.ProjectIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusBadRequest, "PROJECT_ID_REQUIRED", "X-Project-ID required")
		return
	}
	if err := r.ParseMultipartForm(maxImportFileSize); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid multipart")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "file required")
		return
	}
	defer file.Close()

	batch, rows, err := h.svc.UploadBatch(r.Context(), header.Filename, file, projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusCreated, dto.ImportBatchWithRowsResponse{
		Batch: dto.ImportBatchToResponse(batch),
		Rows:  dto.ModerationRowsToResponse(rows),
	})
}

// GetActiveBatch godoc
// @Summary      Get the active (not yet closed) import batch, if any
// @Tags         import
// @Produce      json
// @Success      200 {object} dto.ImportBatchWithRowsResponse
// @Success      204
// @Router       /api/v1/import/batches/active [get]
func (h *ImportHandler) GetActiveBatch(w http.ResponseWriter, r *http.Request) {
	batch, rows, ok, err := h.svc.GetActiveBatch(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	httputil.WriteData(w, http.StatusOK, dto.ImportBatchWithRowsResponse{
		Batch: dto.ImportBatchToResponse(batch),
		Rows:  dto.ModerationRowsToResponse(rows),
	})
}

// UpdateRow godoc
// @Summary      Inline-edit a moderation row
// @Tags         import
// @Accept       json
// @Produce      json
// @Param        id path int true "Row ID"
// @Success      200 {object} dto.ModerationRowResponse
// @Router       /api/v1/import/rows/{id} [patch]
func (h *ImportHandler) UpdateRow(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id")
		return
	}
	var req dto.UpdateModerationRowRequest
	if err := decodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	row, err := h.svc.UpdateRow(r.Context(), id, service.UpdateRowInput{
		Title:       req.Title,
		Amount:      req.Amount,
		Date:        req.Date,
		TagID:       req.TagID,
		Category:    req.Category,
		Specificity: req.Specificity,
		Comment:     req.Comment,
		URL:         req.URL,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusOK, dto.ModerationRowToResponse(row))
}

// AcceptRow godoc
// @Summary      Accept a single ready moderation row into transactions
// @Tags         import
// @Produce      json
// @Param        id path int true "Row ID"
// @Success      201 {object} dto.AcceptedTransactionResponse
// @Router       /api/v1/import/rows/{id}/accept [post]
func (h *ImportHandler) AcceptRow(w http.ResponseWriter, r *http.Request) {
	projectID, ok := middleware.ProjectIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusBadRequest, "PROJECT_ID_REQUIRED", "X-Project-ID required")
		return
	}
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id")
		return
	}
	tx, err := h.svc.AcceptRow(r.Context(), id, projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusCreated, dto.AcceptedTransactionToResponse(tx))
}

// AcceptBatch godoc
// @Summary      Accept multiple ready moderation rows into transactions
// @Tags         import
// @Accept       json
// @Produce      json
// @Param        id path int true "Batch ID"
// @Success      201 {array} dto.AcceptedTransactionResponse
// @Router       /api/v1/import/batches/{id}/accept [post]
func (h *ImportHandler) AcceptBatch(w http.ResponseWriter, r *http.Request) {
	projectID, ok := middleware.ProjectIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusBadRequest, "PROJECT_ID_REQUIRED", "X-Project-ID required")
		return
	}
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id")
		return
	}
	var req dto.AcceptBatchRequest
	if err := decodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	txs, err := h.svc.AcceptBatch(r.Context(), id, req.RowIDs, projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteData(w, http.StatusCreated, dto.AcceptedTransactionsToResponse(txs))
}

// CloseBatch godoc
// @Summary      Close an import batch (discard the moderation session)
// @Tags         import
// @Param        id path int true "Batch ID"
// @Success      204
// @Router       /api/v1/import/batches/{id}/close [post]
func (h *ImportHandler) CloseBatch(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id")
		return
	}
	if err := h.svc.CloseBatch(r.Context(), id); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
