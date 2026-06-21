package service_test

import (
	"strings"
	"testing"

	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/stretchr/testify/require"
)

// buildAndValidateRow, resolveTagPath and recomputeStatus are unexported, so these tests
// exercise them indirectly through the exported parsing/validation pipeline by re-running
// the same CSV the service would parse, via a small in-package shim file (import_service_export_test.go).

func TestImportValidation_NonNumericAmount(t *testing.T) {
	row := exportBuildAndValidateRow(1, 1, exportRawRecord{
		Title: "Продукты", Amount: "тысяча", Date: "2026-06-15",
	}, nil)
	require.Equal(t, domain.ModerationRowStatusError, row.Status)
	require.Contains(t, row.FieldErrors["amount"], "нечисловое")
	require.Nil(t, row.Amount)
}

func TestImportValidation_InvalidDateFormat(t *testing.T) {
	row := exportBuildAndValidateRow(1, 1, exportRawRecord{
		Title: "Продукты", Amount: "1250.00", Date: "15.06.2026",
	}, nil)
	require.Equal(t, domain.ModerationRowStatusError, row.Status)
	require.Contains(t, row.FieldErrors["date"], "YYYY-MM-DD")
	require.Nil(t, row.Date)
}

func TestImportValidation_InvalidCategory(t *testing.T) {
	row := exportBuildAndValidateRow(1, 1, exportRawRecord{
		Title: "Продукты", Amount: "1250.00", Date: "2026-06-15", Category: "unknown",
	}, nil)
	require.Equal(t, domain.ModerationRowStatusError, row.Status)
	require.Contains(t, row.FieldErrors["category"], "expense")
}

func TestImportValidation_TagNotFound(t *testing.T) {
	row := exportBuildAndValidateRow(1, 1, exportRawRecord{
		Title: "Продукты", Amount: "1250.00", Date: "2026-06-15", Tag: "Неизвестная категория",
	}, nil)
	require.Equal(t, domain.ModerationRowStatusError, row.Status)
	require.Contains(t, row.FieldErrors["tag"], "не найден")
	require.Nil(t, row.TagID)
}

func TestImportValidation_MissingRequiredFields(t *testing.T) {
	row := exportBuildAndValidateRow(1, 1, exportRawRecord{}, nil)
	require.Equal(t, domain.ModerationRowStatusError, row.Status)
	require.Equal(t, "обязательное поле не заполнено", row.FieldErrors["title"])
	require.Equal(t, "обязательное поле не заполнено", row.FieldErrors["amount"])
	require.Equal(t, "обязательное поле не заполнено", row.FieldErrors["date"])
}

func TestImportValidation_ValidRowWithoutOptionalFields_IsPending(t *testing.T) {
	row := exportBuildAndValidateRow(1, 1, exportRawRecord{
		Title: "Зарплата", Amount: "120000.00", Date: "2026-06-10",
	}, nil)
	require.Empty(t, row.FieldErrors)
	require.Equal(t, domain.ModerationRowStatusPending, row.Status)
}

func TestImportValidation_FullyFilledRow_IsReady(t *testing.T) {
	tags := []domain.Tag{{ID: 1, Name: "Еда"}}
	row := exportBuildAndValidateRow(1, 1, exportRawRecord{
		Title: "Продукты", Amount: "1250.00", Date: "2026-06-15",
		Tag: "Еда", Category: "expense", Specificity: "simple",
	}, tags)
	require.Empty(t, row.FieldErrors)
	require.Equal(t, domain.ModerationRowStatusReady, row.Status)
	require.NotNil(t, row.Amount)
	require.Equal(t, int64(125000), *row.Amount) // kopecks
	require.NotNil(t, row.TagID)
	require.Equal(t, int64(1), *row.TagID)
}

func TestResolveTagPath_Hierarchy(t *testing.T) {
	food := domain.Tag{ID: 1, Name: "Еда"}
	cafe := domain.Tag{ID: 2, Name: "Кафе", ParentID: ptrInt64(1)}
	otherCafe := domain.Tag{ID: 3, Name: "Кафе"} // root-level tag with the same name, different parent
	tags := []domain.Tag{food, cafe, otherCafe}

	id, ok := exportResolveTagPath(tags, "Еда/Кафе")
	require.True(t, ok)
	require.Equal(t, int64(2), id)

	id, ok = exportResolveTagPath(tags, " еда / кафе ")
	require.True(t, ok)
	require.Equal(t, int64(2), id)

	id, ok = exportResolveTagPath(tags, "Кафе")
	require.True(t, ok)
	require.Equal(t, int64(3), id)

	_, ok = exportResolveTagPath(tags, "Еда/Несуществующий")
	require.False(t, ok)

	_, ok = exportResolveTagPath(tags, strings.Repeat("/", 3))
	require.False(t, ok)
}

func ptrInt64(v int64) *int64 { return &v }
