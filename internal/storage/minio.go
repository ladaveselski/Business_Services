package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync/atomic"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Storage struct {
	client    *minio.Client
	bucket    string
	available atomic.Bool // флаг: доступен ли MinIO
}

type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

// NewStorage пытается подключиться к MinIO.
// Если не получается — возвращает Storage с available=false и nil-ошибкой,
// чтобы приложение могло работать в fallback-режиме.
func NewStorage(cfg Config) (*Storage, error) {
	s := &Storage{bucket: cfg.Bucket}

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		log.Printf("MinIO: клиент не создан (%v) — работаем в fallback-режиме", err)
		return s, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		log.Printf("MinIO: бакет недоступен (%v) — работаем в fallback-режиме", err)
		return s, nil
	}
	if !exists {
		log.Printf("MinIO: бакет %q не существует — работаем в fallback-режиме", cfg.Bucket)
		return s, nil
	}

	s.client = client
	s.available.Store(true)
	log.Printf("MinIO подключён: %s / %s", cfg.Endpoint, cfg.Bucket)
	return s, nil
}

// IsAvailable — true, если MinIO доступен.
func (s *Storage) IsAvailable() bool {
	return s.available.Load()
}

// Upload — загружает файл в MinIO. Если MinIO недоступен — возвращает ошибку.
func (s *Storage) Upload(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	if !s.IsAvailable() {
		return fmt.Errorf("MinIO недоступен")
	}
	_, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

// Get — возвращает объект из MinIO.
func (s *Storage) Get(ctx context.Context, key string) (*minio.Object, error) {
	if !s.IsAvailable() {
		return nil, fmt.Errorf("MinIO недоступен")
	}
	return s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
}

// Stat — метаданные объекта.
func (s *Storage) Stat(ctx context.Context, key string) (minio.ObjectInfo, error) {
	if !s.IsAvailable() {
		return minio.ObjectInfo{}, fmt.Errorf("MinIO недоступен")
	}
	return s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
}