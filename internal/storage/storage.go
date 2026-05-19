// Package storage содержит абстракцию для хранилища файлов.
// Без переменной окружения STORAGE_TYPE=s3 используется LocalStorage —
// файлы сохраняются на локальном диске в /data/uploads (как раньше).
// При STORAGE_TYPE=s3 — S3Storage (Yandex Object Storage / R2 / любой S3-совместимый).
package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	urlPkg "net/url"
	"os"
	"path/filepath"
	"strings"
)

// Storage — абстракция хранилища файлов.
type Storage interface {
	// Put сохраняет файл по ключу key (например "2026/05/abc123.jpg").
	// Возвращает публичный URL для клиента и фактический размер.
	Put(ctx context.Context, key string, r io.Reader, contentType string) (publicURL string, size int64, err error)

	// Delete удаляет файл по ключу.
	Delete(ctx context.Context, key string) error

	// Serve обслуживает HTTP-запрос на чтение файла.
	// Для локального — обычный FileServer. Для S3 — может быть redirect.
	Serve(w http.ResponseWriter, r *http.Request)

	// PublicURL возвращает URL для уже сохранённого ключа.
	PublicURL(key string) string

	// Type возвращает тип хранилища ("local" / "s3") — для логов.
	Type() string
}

// New возвращает Storage на основе env-переменных:
//
//	STORAGE_TYPE=s3 — включает S3Storage (нужен также S3_BUCKET, S3_ENDPOINT, S3_REGION, S3_ACCESS_KEY, S3_SECRET_KEY)
//	иначе — LocalStorage с базовой директорией /data/uploads (или из STORAGE_LOCAL_DIR).
func New() Storage {
	if strings.ToLower(strings.TrimSpace(os.Getenv("STORAGE_TYPE"))) == "s3" {
		s, err := newS3Storage()
		if err != nil {
			log.Printf("[storage] не удалось инициализировать S3, падаем на LocalStorage: %v", err)
		} else {
			log.Printf("[storage] инициализирован S3Storage: bucket=%s endpoint=%s", s.bucket, s.endpoint)
			return s
		}
	}
	dir := strings.TrimSpace(os.Getenv("STORAGE_LOCAL_DIR"))
	if dir == "" {
		dir = "/data/uploads"
	}
	log.Printf("[storage] инициализирован LocalStorage: %s", dir)
	return &LocalStorage{baseDir: dir}
}

// ─────────────────────────────────────────────────────────────
// LocalStorage — файлы на локальном диске
// ─────────────────────────────────────────────────────────────

type LocalStorage struct {
	baseDir string
}

func (s *LocalStorage) Type() string { return "local" }

func (s *LocalStorage) Put(ctx context.Context, key string, r io.Reader, contentType string) (string, int64, error) {
	full := filepath.Join(s.baseDir, key)
	dir := filepath.Dir(full)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", 0, fmt.Errorf("mkdir %s: %w", dir, err)
	}
	out, err := os.Create(full)
	if err != nil {
		return "", 0, fmt.Errorf("create %s: %w", full, err)
	}
	defer out.Close()
	written, err := io.Copy(out, r)
	if err != nil {
		_ = os.Remove(full)
		return "", 0, fmt.Errorf("copy: %w", err)
	}
	return s.PublicURL(key), written, nil
}

func (s *LocalStorage) Delete(ctx context.Context, key string) error {
	full := filepath.Join(s.baseDir, key)
	err := os.Remove(full)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *LocalStorage) Serve(w http.ResponseWriter, r *http.Request) {
	// Файлы в /uploads/ иммутабельные (имя содержит уникальный хеш).
	// Кешируем на сутки + immutable.
	w.Header().Set("Cache-Control", "public, max-age=86400, immutable")

	// SEC (Phase 4): SVG и XML отдаём как attachment — защита от XSS через
	// inline-рендеринг скриптов в SVG. Новые SVG больше не загружаются (заблокированы
	// в /api/upload), но если в БД остались старые — браузер не выполнит их JS.
	pathLower := strings.ToLower(r.URL.Path)
	if strings.HasSuffix(pathLower, ".svg") || strings.HasSuffix(pathLower, ".svgz") ||
		strings.HasSuffix(pathLower, ".xml") || strings.HasSuffix(pathLower, ".xhtml") {
		w.Header().Set("Content-Disposition", "attachment")
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("X-Content-Type-Options", "nosniff")
	}

	// Если есть ?download=имя — отдаём как attachment с оригинальным именем.
	// Имя в URL должно быть URL-encoded (encodeURIComponent на фронте).
	if dl := r.URL.Query().Get("download"); dl != "" {
		// Очистка от потенциально опасных символов
		safe := strings.ReplaceAll(dl, "\"", "")
		safe = strings.ReplaceAll(safe, "\n", "")
		safe = strings.ReplaceAll(safe, "\r", "")
		safe = strings.ReplaceAll(safe, "\\", "")
		// Используем filename*=UTF-8'' для корректной поддержки кириллицы и спецсимволов.
		// urlPathEscape кодирует имя в percent-encoding.
		encoded := urlPathEscapeName(safe)
		w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+encoded)
	}

	http.StripPrefix("/uploads/", http.FileServer(http.Dir(s.baseDir))).ServeHTTP(w, r)
}

// urlPathEscapeName кодирует строку для filename*=UTF-8'' заголовка.
// Использует url.PathEscape (заменяет специальные символы на %XX).
func urlPathEscapeName(s string) string {
	return urlPkg.PathEscape(s)
}

func (s *LocalStorage) PublicURL(key string) string {
	return "/uploads/" + strings.TrimLeft(key, "/")
}
