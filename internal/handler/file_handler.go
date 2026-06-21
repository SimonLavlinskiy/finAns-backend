package handler

import (
	"net/http"

	"github.com/SimonLavlinskiy/finAns-backend/internal/service"
	"github.com/SimonLavlinskiy/finAns-backend/pkg/httputil"
	"github.com/go-chi/chi/v5"
)

type FileHandler struct {
	svc *service.FileService
}

func NewFileHandler(svc *service.FileService) *FileHandler {
	return &FileHandler{svc: svc}
}

func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid multipart")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "file required")
		return
	}
	defer file.Close()

	if err := h.svc.Upload(r.Context(), id, header.Filename, header.Header.Get("Content-Type"), file, header.Size); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *FileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
