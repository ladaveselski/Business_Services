package handler

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func (h *Handler) saveUpload(ctx context.Context, fh *multipart.FileHeader, prefix string) (string, error) {
	if fh == nil {
		return "", nil
	}

	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if !allowedExt(ext) {
		return "", fmt.Errorf("недопустимое расширение: %s", ext)
	}

	key := prefix + "/" + uuid.NewString() + ext

	f, err := fh.Open()
	if err != nil {
		return "", fmt.Errorf("открытие файла: %w", err)
	}
	defer f.Close()

	if !h.Storage.IsAvailable() {
		if err := h.saveLocal(key, f); err != nil {
			return "", err
		}
		return key, nil
	}

	contentType := fh.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	if err := h.Storage.Upload(ctx, key, f, fh.Size, contentType); err != nil {
		return "", fmt.Errorf("загрузка в MinIO: %w", err)
	}
	return key, nil
}

func allowedExt(ext string) bool {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif",
		".mp4", ".webm", ".mov":
		return true
	}
	return false
}
