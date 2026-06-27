package service

import (
	"context"
	"time"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/SimonLavlinskiy/finAns-backend/internal/repository"
)

type ProjectService struct {
	repo     *repository.ProjectRepository
	userRepo *repository.UserRepository
}

func NewProjectService(repo *repository.ProjectRepository, userRepo *repository.UserRepository) *ProjectService {
	return &ProjectService{repo: repo, userRepo: userRepo}
}

func (s *ProjectService) Create(ctx context.Context, userID int64, req dto.CreateProjectRequest) (dto.ProjectResponse, error) {
	if req.Name == "" {
		return dto.ProjectResponse{}, &apperrors.ValidationError{
			Message: "INVALID_NAME",
			Fields:  map[string]string{"name": "required"},
		}
	}

	var initialBalance int64
	if req.InitialBalanceKopecks != nil {
		initialBalance = *req.InitialBalanceKopecks
	}

	var startedAt *time.Time
	if req.StartedAt != nil {
		t, err := time.Parse("2006-01-02", *req.StartedAt)
		if err != nil {
			return dto.ProjectResponse{}, &apperrors.ValidationError{
				Message: "INVALID_DATE",
				Fields:  map[string]string{"started_at": "must be YYYY-MM-DD"},
			}
		}
		startedAt = &t
	}

	project, err := s.repo.Create(ctx, req.Name, initialBalance, startedAt)
	if err != nil {
		return dto.ProjectResponse{}, err
	}

	if err := s.repo.AddMember(ctx, project.ID, userID, domain.RoleOwner); err != nil {
		return dto.ProjectResponse{}, err
	}

	return toProjectResponse(project), nil
}

func (s *ProjectService) ListForUser(ctx context.Context, userID int64) ([]dto.ProjectResponse, error) {
	projects, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	resp := make([]dto.ProjectResponse, len(projects))
	for i, p := range projects {
		resp[i] = toProjectResponse(p)
	}
	return resp, nil
}

func (s *ProjectService) Get(ctx context.Context, id int64) (dto.ProjectResponse, error) {
	project, err := s.repo.Get(ctx, id)
	if err != nil {
		return dto.ProjectResponse{}, err
	}
	return toProjectResponse(project), nil
}

func (s *ProjectService) ListMembers(ctx context.Context, projectID int64) ([]dto.ProjectMemberResponse, error) {
	members, err := s.repo.ListMembers(ctx, projectID)
	if err != nil {
		return nil, err
	}

	resp := make([]dto.ProjectMemberResponse, 0, len(members))
	for _, m := range members {
		user, err := s.userRepo.GetByID(ctx, m.UserID)
		if err != nil {
			return nil, err
		}
		resp = append(resp, dto.ProjectMemberResponse{
			UserID:      user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			Role:        m.Role,
		})
	}
	return resp, nil
}

func (s *ProjectService) AddMember(ctx context.Context, projectID, callerUserID int64, username string) error {
	caller, err := s.repo.GetMember(ctx, projectID, callerUserID)
	if err != nil {
		return &apperrors.ForbiddenError{Message: "not a member of this project"}
	}
	if caller.Role != domain.RoleOwner {
		return &apperrors.ForbiddenError{Message: "only project owners can add members"}
	}

	target, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		var nf *apperrors.NotFoundError
		if ok := isErr[*apperrors.NotFoundError](err, &nf); ok {
			return &apperrors.NotFoundError{Resource: "user"}
		}
		return err
	}

	_, err = s.repo.GetMember(ctx, projectID, target.ID)
	if err == nil {
		return &apperrors.ConflictError{Code: "ALREADY_MEMBER", Message: "user is already a member"}
	}

	return s.repo.AddMember(ctx, projectID, target.ID, domain.RoleMember)
}

func (s *ProjectService) RemoveMember(ctx context.Context, projectID, callerUserID, targetUserID int64) error {
	caller, err := s.repo.GetMember(ctx, projectID, callerUserID)
	if err != nil {
		return &apperrors.ForbiddenError{Message: "not a member of this project"}
	}
	if caller.Role != domain.RoleOwner {
		return &apperrors.ForbiddenError{Message: "only project owners can remove members"}
	}

	target, err := s.repo.GetMember(ctx, projectID, targetUserID)
	if err != nil {
		return err
	}

	if target.Role == domain.RoleOwner {
		count, err := s.repo.CountOwners(ctx, projectID)
		if err != nil {
			return err
		}
		if count <= 1 {
			return &apperrors.ConflictError{Code: "CANNOT_LEAVE_SOLE_OWNER", Message: "cannot remove the sole owner"}
		}
	}

	return s.repo.RemoveMember(ctx, projectID, targetUserID)
}

func toProjectResponse(p domain.Project) dto.ProjectResponse {
	resp := dto.ProjectResponse{
		ID:                    p.ID,
		Name:                  p.Name,
		InitialBalanceKopecks: p.InitialBalanceKopecks,
		CreatedAt:             p.CreatedAt.Format(time.RFC3339),
	}
	if p.StartedAt != nil {
		s := p.StartedAt.Format("2006-01-02")
		resp.StartedAt = &s
	}
	return resp
}

func isErr[T error](err error, target *T) bool {
	if err == nil {
		return false
	}
	t, ok := err.(T)
	if ok && target != nil {
		*target = t
	}
	return ok
}
