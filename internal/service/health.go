package service

import (
	"context"

	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
)

type HealthService struct {
	checker domain.HealthChecker
	version string
}

func NewHealthService(checker domain.HealthChecker, version string) *HealthService {
	return &HealthService{checker: checker, version: version}
}

func (s *HealthService) Check(ctx context.Context) dto.HealthResponse {
	resp := dto.HealthResponse{
		Status:  "ok",
		DB:      "down",
		Version: s.version,
	}

	if err := s.checker.Ping(ctx); err == nil {
		resp.DB = "up"
	}

	return resp
}
