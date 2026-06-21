package service

import (
	"context"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/repository"
	"github.com/SimonLavlinskiy/finAns-backend/pkg/authtoken"
	"golang.org/x/crypto/bcrypt"
)

const SessionTTL = 7 * 24 * time.Hour

type AuthService struct {
	repo   *repository.UserRepository
	secret []byte
}

func NewAuthService(repo *repository.UserRepository, secret []byte) *AuthService {
	return &AuthService{repo: repo, secret: secret}
}

func (s *AuthService) Login(ctx context.Context, login, password string) (string, error) {
	user, err := s.repo.GetByLogin(ctx, login)
	if err != nil {
		return "", &apperrors.UnauthorizedError{Message: "invalid login or password"}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", &apperrors.UnauthorizedError{Message: "invalid login or password"}
	}

	return authtoken.Generate(s.secret, user.ID, user.Login, SessionTTL)
}
