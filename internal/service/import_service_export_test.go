package service_test

import (
	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/SimonLavlinskiy/finAns-backend/internal/service"
)

// Test-only re-exports of unexported import_service.go internals, defined via the
// service package's exported test shims (see import_service_internal_export_test.go).
type exportRawRecord = service.RawRecordForTest

func exportBuildAndValidateRow(batchID int64, rowNumber int, rec exportRawRecord, tags []domain.Tag) domain.ModerationRow {
	return service.BuildAndValidateRowForTest(batchID, rowNumber, rec, tags)
}

func exportResolveTagPath(tags []domain.Tag, path string) (int64, bool) {
	return service.ResolveTagPathForTest(tags, path)
}
