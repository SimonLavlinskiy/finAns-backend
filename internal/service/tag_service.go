package service

import (
	"context"
	"net/url"
	"strings"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/SimonLavlinskiy/finAns-backend/internal/repository"
	"github.com/SimonLavlinskiy/finAns-backend/pkg/colorutil"
)

type TagService struct {
	repo *repository.TagRepository
}

func NewTagService(repo *repository.TagRepository) *TagService {
	return &TagService{repo: repo}
}

func (s *TagService) ListTree(ctx context.Context) ([]dto.TagResponse, error) {
	tags, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	return buildTagTree(tags), nil
}

func buildTagTree(tags []domain.Tag) []dto.TagResponse {
	byID := make(map[int64]domain.Tag)
	children := make(map[int64][]domain.Tag)
	for _, t := range tags {
		byID[t.ID] = t
		if t.ParentID == nil {
			continue
		}
		children[*t.ParentID] = append(children[*t.ParentID], t)
	}

	var build func(domain.Tag) dto.TagResponse
	build = func(t domain.Tag) dto.TagResponse {
		resp := dto.TagResponse{
			ID:       t.ID,
			Name:     t.Name,
			Color:    t.Color,
			Children: []dto.TagResponse{},
		}
		if t.ParentID != nil {
			resp.ParentID = t.ParentID
		}
		for _, ch := range children[t.ID] {
			resp.Children = append(resp.Children, build(ch))
		}
		return resp
	}

	roots := []dto.TagResponse{}
	for _, t := range tags {
		if t.ParentID == nil {
			roots = append(roots, build(t))
		}
	}
	return roots
}

const defaultTagColor = "#112250"
const childColorLighten = 0.4

func (s *TagService) Create(ctx context.Context, req dto.CreateTagRequest) (dto.TagResponse, error) {
	fields := map[string]string{}
	if strings.TrimSpace(req.Name) == "" {
		fields["name"] = "required"
	}
	if req.Color == "" {
		req.Color = defaultTagColor
	}
	if req.ParentID != nil {
		if err := s.repo.ValidateParent(ctx, *req.ParentID); err != nil {
			fields["parent_id"] = "not found"
		} else {
			parent, err := s.repo.GetByID(ctx, *req.ParentID)
			if err == nil {
				req.Color = colorutil.Lighten(parent.Color, childColorLighten)
			}
		}
	}
	if len(fields) > 0 {
		return dto.TagResponse{}, &apperrors.ValidationError{Fields: fields, Message: "validation failed"}
	}

	t, err := s.repo.Create(ctx, req.Name, req.Color, req.ParentID)
	if err != nil {
		return dto.TagResponse{}, err
	}
	return tagToDTO(t), nil
}

func (s *TagService) Update(ctx context.Context, id int64, req dto.UpdateTagRequest) (dto.TagResponse, error) {
	fields := map[string]string{}
	if strings.TrimSpace(req.Name) == "" {
		fields["name"] = "required"
	}
	if req.Color == "" {
		req.Color = defaultTagColor
	}
	if len(fields) > 0 {
		return dto.TagResponse{}, &apperrors.ValidationError{Fields: fields, Message: "validation failed"}
	}

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return dto.TagResponse{}, err
	}

	color := req.Color
	if existing.ParentID != nil {
		parent, err := s.repo.GetByID(ctx, *existing.ParentID)
		if err != nil {
			return dto.TagResponse{}, err
		}
		color = colorutil.Lighten(parent.Color, childColorLighten)
	}

	t, err := s.repo.Update(ctx, id, req.Name, color)
	if err != nil {
		return dto.TagResponse{}, err
	}

	if existing.ParentID == nil {
		_ = s.repo.UpdateChildrenColor(ctx, id, colorutil.Lighten(color, childColorLighten))
	}

	return tagToDTO(t), nil
}

func (s *TagService) Delete(ctx context.Context, id int64, cascade bool) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return err
	}

	parentID, _ := s.repo.ParentID(ctx, id)
	reassignTo := parentID // nil для корневых тегов

	if cascade {
		// Для корневых тегов (reassignTo == nil) проверяем: если есть транзакции
		// у самого тега или у потомков — блокируем, чтобы не потерять данные.
		if reassignTo == nil {
			total, err := s.repo.CountUsageSubtree(ctx, id)
			if err != nil {
				return err
			}
			if total > 0 {
				return &apperrors.ValidationError{
					Message: "root tag has transactions; reassign them before deleting",
					Fields:  map[string]string{"transactions": "exists"},
				}
			}
		}
		return s.repo.DeleteCascade(ctx, id, reassignTo)
	}

	count, err := s.repo.CountUsage(ctx, id)
	if err != nil {
		return err
	}
	descendants, err := s.repo.ListDescendantIDs(ctx, id)
	if err != nil {
		return err
	}
	if len(descendants) > 0 {
		return &apperrors.ValidationError{
			Message: "tag has children; use cascade delete",
			Fields:  map[string]string{"children": "exists"},
		}
	}

	if count > 0 {
		if err := s.repo.ReassignTransactions(ctx, id, reassignTo); err != nil {
			return err
		}
	}
	return s.repo.Delete(ctx, id)
}

func (s *TagService) GetUsage(ctx context.Context, id int64) (dto.TagUsageResponse, error) {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return dto.TagUsageResponse{}, err
	}
	count, err := s.repo.CountUsage(ctx, id)
	if err != nil {
		return dto.TagUsageResponse{}, err
	}
	return dto.TagUsageResponse{Count: count}, nil
}

func (s *TagService) GetByID(ctx context.Context, id int64) (domain.Tag, error) {
	return s.repo.GetByID(ctx, id)
}

func tagToDTO(t domain.Tag) dto.TagResponse {
	resp := dto.TagResponse{ID: t.ID, Name: t.Name, Color: t.Color, Children: []dto.TagResponse{}}
	if t.ParentID != nil {
		resp.ParentID = t.ParentID
	}
	return resp
}

func (s *TagService) TagWithParent(ctx context.Context, tagID int64) (dto.TransactionTag, error) {
	t, err := s.repo.GetByID(ctx, tagID)
	if err != nil {
		return dto.TransactionTag{}, err
	}
	result := dto.TransactionTag{ID: t.ID, Name: t.Name, Color: t.Color}
	if t.ParentID != nil {
		p, err := s.repo.GetByID(ctx, *t.ParentID)
		if err == nil {
			result.Parent = &dto.TagParent{ID: p.ID, Name: p.Name, Color: p.Color}
		}
	}
	return result, nil
}

// TagsWithParentBatch загружает теги для группы транзакций одним батчем (устраняет N+1).
func (s *TagService) TagsWithParentBatch(ctx context.Context, tagIDs []int64) (map[int64]dto.TransactionTag, error) {
	if len(tagIDs) == 0 {
		return map[int64]dto.TransactionTag{}, nil
	}

	// Дедупликация
	unique := make(map[int64]struct{}, len(tagIDs))
	for _, id := range tagIDs {
		unique[id] = struct{}{}
	}
	deduped := make([]int64, 0, len(unique))
	for id := range unique {
		deduped = append(deduped, id)
	}

	tags, err := s.repo.GetManyByIDs(ctx, deduped)
	if err != nil {
		return nil, err
	}

	byID := make(map[int64]domain.Tag, len(tags))
	parentIDs := make([]int64, 0)
	for _, t := range tags {
		byID[t.ID] = t
		if t.ParentID != nil {
			parentIDs = append(parentIDs, *t.ParentID)
		}
	}

	parentTags, err := s.repo.GetManyByIDs(ctx, parentIDs)
	if err != nil {
		return nil, err
	}
	parentByID := make(map[int64]domain.Tag, len(parentTags))
	for _, p := range parentTags {
		parentByID[p.ID] = p
	}

	result := make(map[int64]dto.TransactionTag, len(tagIDs))
	for _, id := range tagIDs {
		t, ok := byID[id]
		if !ok {
			continue
		}
		tt := dto.TransactionTag{ID: t.ID, Name: t.Name, Color: t.Color}
		if t.ParentID != nil {
			if p, ok := parentByID[*t.ParentID]; ok {
				tt.Parent = &dto.TagParent{ID: p.ID, Name: p.Name, Color: p.Color}
			}
		}
		result[id] = tt
	}
	return result, nil
}

func ValidateTagExists(ctx context.Context, repo *repository.TagRepository, tagID int64) error {
	exists, err := repo.Exists(ctx, tagID)
	if err != nil {
		return err
	}
	if !exists {
		return &apperrors.ValidationError{Fields: map[string]string{"tag_id": "not found"}}
	}
	return nil
}

func isValidURL(raw string) bool {
	u, err := url.ParseRequestURI(raw)
	return err == nil && u.Scheme != "" && u.Host != ""
}
