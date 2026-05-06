package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// S3Storage — реализация на S3-совместимом хранилище (Yandex Object Storage / R2 / Backblaze B2).
// Заглушка: будет реализована в следующем коммите. Пока возвращает ошибку при попытке Put,
// чтобы при STORAGE_TYPE=s3 без правильных ключей сервер падал на LocalStorage в New().
type S3Storage struct {
	bucket    string
	endpoint  string
	region    string
	accessKey string
	secretKey string
	publicURL string
}

func newS3Storage() (*S3Storage, error) {
	return nil, fmt.Errorf("S3 storage не реализован — будет добавлен в коммите 2.2")
}

func (s *S3Storage) Type() string { return "s3" }

func (s *S3Storage) Put(ctx context.Context, key string, r io.Reader, contentType string) (string, int64, error) {
	return "", 0, fmt.Errorf("not implemented")
}

func (s *S3Storage) Delete(ctx context.Context, key string) error {
	return fmt.Errorf("not implemented")
}

func (s *S3Storage) Serve(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (s *S3Storage) PublicURL(key string) string {
	return ""
}
