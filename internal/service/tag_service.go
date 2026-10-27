package service

import (
	"context"
	"net/url"
	"strings"
	"time"

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

func (s *TagService) ListTree(ctx context.Context, projectID int64, includeDeleted bool) ([]dto.TagResponse, error) {
	var (
		tags []domain.Tag
		err  error
	)
	if includeDeleted {
		tags, err = s.repo.ListWithDeleted(ctx, projectID)
	} else {
		tags, err = s.repo.List(ctx, projectID)
	}
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
		resp := tagToDTO(t)
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

func (s *TagService) Create(ctx context.Context, req dto.CreateTagRequest, projectID int64) (dto.TagResponse, error) {
	fields := map[string]string{}
	if strings.TrimSpace(req.Name) == "" {
		fields["name"] = "required"
	}
	if req.Color == "" {
		req.Color = defaultTagColor
	}
	kind := req.Kind
	if kind == "" {
		kind = domain.TagKindExpense
	}
	if !isValidTagKind(kind) {
		fields["kind"] = "must be expense or income"
	}
	if req.ParentID != nil {
		if err := s.repo.ValidateParent(ctx, *req.ParentID); err != nil {
			fields["parent_id"] = "not found"
		} else {
			parent, err := s.repo.GetByID(ctx, *req.ParentID)
			if err == nil {
				req.Color = colorutil.Lighten(parent.Color, childColorLighten)
				kind = parent.Kind
			}
		}
	}
	if len(fields) > 0 {
		return dto.TagResponse{}, &apperrors.ValidationError{Fields: fields, Message: "validation failed"}
	}

	t, err := s.repo.Create(ctx, req.Name, req.Color, req.ParentID, projectID, kind)
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
	if req.Kind != nil && !isValidTagKind(*req.Kind) {
		fields["kind"] = "must be expense or income"
	}
	if len(fields) > 0 {
		return dto.TagResponse{}, &apperrors.ValidationError{Fields: fields, Message: "validation failed"}
	}

	existing, err := s.getAlive(ctx, id)
	if err != nil {
		return dto.TagResponse{}, err
	}

	color := req.Color
	kind := existing.Kind
	if existing.ParentID != nil {
		parent, err := s.repo.GetByID(ctx, *existing.ParentID)
		if err != nil {
			return dto.TagResponse{}, err
		}
		color = colorutil.Lighten(parent.Color, childColorLighten)
	} else if req.Kind != nil {
		kind = *req.Kind
	}
	order := existing.Order
	if req.Order != nil {
		order = *req.Order
	}

	t, err := s.repo.Update(ctx, id, req.Name, color, kind, order)
	if err != nil {
		return dto.TagResponse{}, err
	}

	if existing.ParentID == nil {
		_ = s.repo.UpdateChildrenColor(ctx, id, colorutil.Lighten(color, childColorLighten))
		if kind != existing.Kind {
			if err := s.repo.UpdateChildrenKind(ctx, id, kind); err != nil {
				return dto.TagResponse{}, err
			}
		}
	}

	return tagToDTO(t), nil
}

// Delete мягко удаляет тег: операции сохраняют ссылку на него, лимиты тега
// удаляются. Без cascade тег с подтегами удалить нельзя.
func (s *TagService) Delete(ctx context.Context, id int64, cascade bool) error {
	if _, err := s.getAlive(ctx, id); err != nil {
		return err
	}
	if !cascade {
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
	}
	return s.repo.SoftDeleteSubtree(ctx, id)
}

// getAlive возвращает тег, если он не удалён; удалённый считается ненайденным.
func (s *TagService) getAlive(ctx context.Context, id int64) (domain.Tag, error) {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.Tag{}, err
	}
	if t.DeletedAt != nil {
		return domain.Tag{}, &apperrors.NotFoundError{Resource: "tag"}
	}
	return t, nil
}

func isValidTagKind(kind string) bool {
	return kind == domain.TagKindExpense || kind == domain.TagKindIncome
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
	resp := dto.TagResponse{
		ID:        t.ID,
		Name:      t.Name,
		Color:     t.Color,
		Kind:      t.Kind,
		Order:     t.Order,
		CreatedAt: t.CreatedAt.UTC().Format(time.RFC3339),
		Children:  []dto.TagResponse{},
	}
	if t.ParentID != nil {
		resp.ParentID = t.ParentID
	}
	if t.DeletedAt != nil {
		v := t.DeletedAt.UTC().Format(time.RFC3339)
		resp.DeletedAt = &v
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
