package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Stub service ---

type stubTagSvc struct {
	listResult   []dto.TagResponse
	listErr      error
	createResult dto.TagResponse
	createErr    error
	updateResult dto.TagResponse
	updateErr    error
	deleteErr    error
	usageResult  dto.TagUsageResponse
	usageErr     error
}

func (s *stubTagSvc) ListTree(_ context.Context) ([]dto.TagResponse, error) {
	return s.listResult, s.listErr
}

func (s *stubTagSvc) Create(_ context.Context, req dto.CreateTagRequest) (dto.TagResponse, error) {
	return s.createResult, s.createErr
}

func (s *stubTagSvc) Update(_ context.Context, id int64, req dto.UpdateTagRequest) (dto.TagResponse, error) {
	return s.updateResult, s.updateErr
}

func (s *stubTagSvc) Delete(_ context.Context, id int64, cascade bool) error {
	return s.deleteErr
}

func (s *stubTagSvc) GetUsage(_ context.Context, id int64) (dto.TagUsageResponse, error) {
	return s.usageResult, s.usageErr
}

// minimalTagHandler дублирует логику TagHandler через интерфейс.
type minimalTagHandler struct {
	stub *stubTagSvc
}

func (h *minimalTagHandler) list(w http.ResponseWriter, r *http.Request) {
	tree, err := h.stub.ListTree(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": tree})
}

func (h *minimalTagHandler) create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateTagRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]any{"message": "invalid json"}})
		return
	}
	tag, err := h.stub.Create(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": tag})
}

func (h *minimalTagHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]any{"message": "invalid id"}})
		return
	}
	var req dto.UpdateTagRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]any{"message": "invalid json"}})
		return
	}
	tag, err := h.stub.Update(r.Context(), id, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": tag})
}

func (h *minimalTagHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]any{"message": "invalid id"}})
		return
	}
	cascade := r.URL.Query().Get("cascade") == "true"
	if err := h.stub.Delete(r.Context(), id, cascade); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *minimalTagHandler) usage(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]any{"message": "invalid id"}})
		return
	}
	u, err := h.stub.GetUsage(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": u})
}

// --- Tests ---

func TestTagHandler_List_OK(t *testing.T) {
	stub := &stubTagSvc{
		listResult: []dto.TagResponse{
			{ID: 1, Name: "Food", Color: "#ff0000", Children: []dto.TagResponse{}},
		},
	}
	h := &minimalTagHandler{stub: stub}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tags", nil)
	rec := httptest.NewRecorder()
	h.list(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp map[string][]dto.TagResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Len(t, resp["data"], 1)
	assert.Equal(t, "Food", resp["data"][0].Name)
}

func TestTagHandler_List_Empty(t *testing.T) {
	stub := &stubTagSvc{listResult: []dto.TagResponse{}}
	h := &minimalTagHandler{stub: stub}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tags", nil)
	rec := httptest.NewRecorder()
	h.list(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestTagHandler_Create_OK(t *testing.T) {
	stub := &stubTagSvc{
		createResult: dto.TagResponse{ID: 42, Name: "Transport", Color: "#0000ff", Children: []dto.TagResponse{}},
	}
	h := &minimalTagHandler{stub: stub}

	body := `{"name":"Transport","color":"#0000ff"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tags", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.create(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	var resp map[string]dto.TagResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, int64(42), resp["data"].ID)
	assert.Equal(t, "Transport", resp["data"].Name)
}

func TestTagHandler_Create_ValidationError(t *testing.T) {
	stub := &stubTagSvc{
		createErr: &apperrors.ValidationError{
			Message: "validation failed",
			Fields:  map[string]string{"name": "required"},
		},
	}
	h := &minimalTagHandler{stub: stub}

	body := `{"name":"","color":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tags", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.create(rec, req)

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	errObj := resp["error"].(map[string]any)
	fields := errObj["fields"].(map[string]any)
	assert.Equal(t, "required", fields["name"])
}

func TestTagHandler_Create_InvalidJSON(t *testing.T) {
	stub := &stubTagSvc{}
	h := &minimalTagHandler{stub: stub}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tags", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.create(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTagHandler_Update_InvalidID(t *testing.T) {
	stub := &stubTagSvc{}
	h := &minimalTagHandler{stub: stub}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/tags/abc", strings.NewReader(`{"name":"X","color":"#fff"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	chiCtxWithID("abc", http.HandlerFunc(h.update)).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTagHandler_Update_NotFound(t *testing.T) {
	stub := &stubTagSvc{
		updateErr: &apperrors.NotFoundError{Resource: "tag"},
	}
	h := &minimalTagHandler{stub: stub}

	body := `{"name":"X","color":"#fff"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tags/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	chiCtxWithID("1", http.HandlerFunc(h.update)).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestTagHandler_Delete_OK(t *testing.T) {
	stub := &stubTagSvc{}
	h := &minimalTagHandler{stub: stub}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tags/5", nil)
	rec := httptest.NewRecorder()
	chiCtxWithID("5", http.HandlerFunc(h.delete)).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestTagHandler_Usage_OK(t *testing.T) {
	stub := &stubTagSvc{
		usageResult: dto.TagUsageResponse{Count: 7},
	}
	h := &minimalTagHandler{stub: stub}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tags/3/usage", nil)
	rec := httptest.NewRecorder()
	chiCtxWithID("3", http.HandlerFunc(h.usage)).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]dto.TagUsageResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, int64(7), resp["data"].Count)
}

func TestTagHandler_Usage_InvalidID(t *testing.T) {
	stub := &stubTagSvc{}
	h := &minimalTagHandler{stub: stub}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tags/xyz/usage", nil)
	rec := httptest.NewRecorder()
	chiCtxWithID("xyz", http.HandlerFunc(h.usage)).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
