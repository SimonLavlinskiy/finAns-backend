package service

import (
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"

	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/internal/repository"
	"github.com/google/uuid"
)

var allowedMIMETypes = map[string]bool{
	"image/jpeg":         true,
	"image/png":          true,
	"image/webp":         true,
	"application/pdf":    true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
}

const maxFileSize = 10 * 1024 * 1024

type FileService struct {
	uploadDir string
	txRepo    *repository.TransactionRepository
}

func NewFileService(uploadDir string, txRepo *repository.TransactionRepository) *FileService {
	return &FileService{uploadDir: uploadDir, txRepo: txRepo}
}

func (s *FileService) Upload(ctx context.Context, txID int64, filename string, contentType string, r io.Reader, size int64) error {
	if size > maxFileSize {
		return &apperrors.ValidationError{Fields: map[string]string{"file": "max 10MB"}, Message: "file too large"}
	}
	ct := contentType
	if ct == "" {
		ct = mime.TypeByExtension(filepath.Ext(filename))
	}
	if !allowedMIMETypes[ct] {
		return &apperrors.ValidationError{Fields: map[string]string{"file": "unsupported type"}, Message: "invalid file type"}
	}

	tx, err := s.txRepo.GetByID(ctx, txID)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(s.uploadDir, 0o755); err != nil {
		return err
	}

	ext := filepath.Ext(filename)
	storedName := fmt.Sprintf("%s%s", uuid.New().String(), ext)
	storedPath := filepath.Join(s.uploadDir, storedName)

	f, err := os.Create(storedPath)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		_ = os.Remove(storedPath)
		return err
	}

	if tx.FilePath != nil && *tx.FilePath != "" {
		_ = os.Remove(*tx.FilePath)
	}

	relPath := filepath.Join("uploads", storedName)

	return s.txRepo.UpdateFile(ctx, txID, &relPath, &filename, &ct)
}

func (s *FileService) Delete(ctx context.Context, txID int64) error {
	tx, err := s.txRepo.GetByID(ctx, txID)
	if err != nil {
		return err
	}
	if tx.FilePath != nil && *tx.FilePath != "" {
		_ = os.Remove(*tx.FilePath)
	}
	return s.txRepo.ClearFile(ctx, txID)
}
