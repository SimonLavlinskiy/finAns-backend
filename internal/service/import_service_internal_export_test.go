package service

import "github.com/SimonLavlinskiy/finAns-backend/internal/domain"

// Exported aliases of unexported import_service.go internals, for use only by
// import_service_test.go (package service_test). Test-only, never compiled into the binary.

type RawRecordForTest = rawRecord

func BuildAndValidateRowForTest(batchID int64, rowNumber int, rec RawRecordForTest, tags []domain.Tag) domain.ModerationRow {
	return buildAndValidateRow(batchID, rowNumber, rec, tags)
}

func ResolveTagPathForTest(tags []domain.Tag, path string) (int64, bool) {
	return resolveTagPath(tags, path)
}
