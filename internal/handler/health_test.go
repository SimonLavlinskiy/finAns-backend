package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/SimonLavlinskiy/finAns-backend/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubHealthChecker struct {
	err error
}

func (s stubHealthChecker) Ping(_ context.Context) error {
	return s.err
}

func TestHealthHandler_OK_DBUp(t *testing.T) {
	h := NewHealthHandler(service.NewHealthService(stubHealthChecker{}, "test"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	h.HealthCheck(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp dto.HealthResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "ok", resp.Status)
	assert.Equal(t, "up", resp.DB)
	assert.Equal(t, "test", resp.Version)
}

func TestHealthHandler_DBDown(t *testing.T) {
	h := NewHealthHandler(service.NewHealthService(stubHealthChecker{err: assert.AnError}, "test"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	h.HealthCheck(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp dto.HealthResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "ok", resp.Status)
	assert.Equal(t, "down", resp.DB)
}

func TestNotFound(t *testing.T) {
	rec := httptest.NewRecorder()
	NotFound(rec, httptest.NewRequest(http.MethodGet, "/missing", nil))
	require.Equal(t, http.StatusNotFound, rec.Code)
}
