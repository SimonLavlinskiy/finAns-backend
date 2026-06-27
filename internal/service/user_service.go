package service

import (
	"context"
	"regexp"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/SimonLavlinskiy/finAns-backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`)

type UserService struct {
	repo        *repository.UserRepository
	projectRepo *repository.ProjectRepository
}

func NewUserService(repo *repository.UserRepository, projectRepo *repository.ProjectRepository) *UserService {
	return &UserService{repo: repo, projectRepo: projectRepo}
}

func (s *UserService) Login(ctx context.Context, req dto.LoginRequest) (dto.UserResponse, error) {
	user, err := s.repo.GetByUsername(ctx, req.Username)
	if err != nil {
		return dto.UserResponse{}, &apperrors.ValidationError{
			Message: "INVALID_CREDENTIALS",
			Fields:  map[string]string{"password": "неверный логин или пароль"},
		}
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return dto.UserResponse{}, &apperrors.ValidationError{
			Message: "INVALID_CREDENTIALS",
			Fields:  map[string]string{"password": "неверный логин или пароль"},
		}
	}
	return toUserResponse(user), nil
}

func (s *UserService) Create(ctx context.Context, req dto.CreateUserRequest) (dto.UserResponse, error) {
	if !usernameRegex.MatchString(req.Username) {
		return dto.UserResponse{}, &apperrors.ValidationError{
			Message: "INVALID_USERNAME",
			Fields:  map[string]string{"username": "3-20 символов, только буквы/цифры/_"},
		}
	}
	if req.DisplayName == "" {
		return dto.UserResponse{}, &apperrors.ValidationError{
			Message: "INVALID_DISPLAY_NAME",
			Fields:  map[string]string{"display_name": "обязательное поле"},
		}
	}
	if len(req.Password) < 6 {
		return dto.UserResponse{}, &apperrors.ValidationError{
			Message: "INVALID_PASSWORD",
			Fields:  map[string]string{"password": "минимум 6 символов"},
		}
	}

	exists, err := s.repo.Exists(ctx, req.Username)
	if err != nil {
		return dto.UserResponse{}, err
	}
	if exists {
		return dto.UserResponse{}, &apperrors.ConflictError{Code: "USERNAME_TAKEN", Message: "логин уже занят"}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return dto.UserResponse{}, err
	}

	user, err := s.repo.Create(ctx, req.Username, req.DisplayName, string(hash))
	if err != nil {
		return dto.UserResponse{}, err
	}

	if s.projectRepo != nil {
		if orphaned, oErr := s.projectRepo.ListOrphaned(ctx); oErr == nil {
			for _, p := range orphaned {
				_ = s.projectRepo.AddMember(ctx, p.ID, user.ID, "owner")
			}
		}
	}

	return toUserResponse(user), nil
}

func (s *UserService) List(ctx context.Context) ([]dto.UserResponse, error) {
	users, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	resp := make([]dto.UserResponse, len(users))
	for i, u := range users {
		resp[i] = toUserResponse(u)
	}
	return resp, nil
}

func (s *UserService) GetByUsername(ctx context.Context, username string) (dto.UserResponse, error) {
	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return dto.UserResponse{}, err
	}
	return toUserResponse(user), nil
}

func toUserResponse(u domain.User) dto.UserResponse {
	return dto.UserResponse{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		IsAdmin:     u.IsAdmin,
	}
}
