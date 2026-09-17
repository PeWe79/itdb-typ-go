package service

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"itdb-backend/internal/common/primitives"
	"itdb-backend/internal/repository"
)

var floorplanExtensions = map[string]struct{}{".jpg": {}, ".jpeg": {}, ".png": {}, ".gif": {}, ".bmp": {}, ".webp": {}, ".svg": {}, ".avif": {}}
var invoiceExtensions = map[string]struct{}{".jpg": {}, ".jpeg": {}, ".png": {}, ".gif": {}, ".bmp": {}, ".webp": {}, ".svg": {}, ".avif": {}, ".pdf": {}}

type FileStorageService struct {
	repo      repository.Repository
	uploadDir string
}

func NewFileStorageService(repo repository.Repository, uploadDir string) *FileStorageService {
	return &FileStorageService{repo: repo, uploadDir: uploadDir}
}
func (s *FileStorageService) StoreUploadedFile(h *multipart.FileHeader, fileTypeID int64, title string) (string, error) {
	if h == nil {
		return "", errors.New("missing file")
	}
	ext := strings.ToLower(filepath.Ext(h.Filename))
	if ext == "" {
		ext = ".bin"
	}
	var typeName sql.NullString
	_ = s.repo.QueryRow(`SELECT typedesc FROM filetypes WHERE id = ?`, fileTypeID).Scan(&typeName)
	if fileTypeID == 3 || strings.EqualFold(typeName.String, "invoice") || typeName.String == "发票" {
		if _, ok := invoiceExtensions[ext]; !ok {
			return "", errors.New("invoice file must be an image or pdf")
		}
	}
	name := fmt.Sprintf("%s-%s-%s%s", primitives.BuildFilenameSegment(typeName.String), primitives.BuildFilenameSegment(title), primitives.ShortUUID(), ext)
	return s.copy(h, name)
}
func (s *FileStorageService) StoreFloorplanFile(h *multipart.FileHeader, locationName string) (string, error) {
	if h == nil {
		return "", errors.New("missing file")
	}
	ext := strings.ToLower(filepath.Ext(h.Filename))
	if _, ok := floorplanExtensions[ext]; !ok {
		return "", errors.New("floorplan file must be an image")
	}
	return s.copy(h, fmt.Sprintf("floorplan-%s-%s%s", primitives.BuildFilenameSegment(locationName), primitives.ShortUUID(), ext))
}
func (s *FileStorageService) copy(h *multipart.FileHeader, name string) (string, error) {
	path := filepath.Join(s.uploadDir, strings.ToLower(name))
	src, err := h.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()
	dst, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}
	return strings.ToLower(name), nil
}
