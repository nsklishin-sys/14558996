package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Storage — реализация на S3-совместимом хранилище.
// Поддерживает Yandex Object Storage, Cloudflare R2, Backblaze B2, AWS S3.
type S3Storage struct {
	client    *s3.Client
	bucket    string
	endpoint  string
	region    string
	publicURL string // префикс для публичных URL: https://<bucket>.storage.yandexcloud.net (без слэша на конце)
}

// newS3Storage читает env-переменные:
//
//	S3_BUCKET            — имя bucket (обязательно)
//	S3_ENDPOINT          — endpoint URL (например https://storage.yandexcloud.net)
//	S3_REGION            — регион (например ru-central1; по умолчанию us-east-1)
//	S3_ACCESS_KEY        — Access Key ID (обязательно)
//	S3_SECRET_KEY        — Secret Access Key (обязательно)
//	S3_PUBLIC_URL_BASE   — публичный URL-префикс (например https://lastop-uploads.storage.yandexcloud.net).
//	                       Если пусто — собирается автоматически из endpoint + bucket.
//	S3_FORCE_PATH_STYLE  — "true" чтобы использовать path-style URLs (нужно для некоторых провайдеров)
func newS3Storage() (*S3Storage, error) {
	bucket := strings.TrimSpace(os.Getenv("S3_BUCKET"))
	endpoint := strings.TrimSpace(os.Getenv("S3_ENDPOINT"))
	region := strings.TrimSpace(os.Getenv("S3_REGION"))
	accessKey := strings.TrimSpace(os.Getenv("S3_ACCESS_KEY"))
	secretKey := os.Getenv("S3_SECRET_KEY")
	publicURL := strings.TrimRight(strings.TrimSpace(os.Getenv("S3_PUBLIC_URL_BASE")), "/")
	forcePathStyle := strings.ToLower(strings.TrimSpace(os.Getenv("S3_FORCE_PATH_STYLE"))) == "true"

	if bucket == "" {
		return nil, fmt.Errorf("S3_BUCKET не задан")
	}
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("S3_ACCESS_KEY или S3_SECRET_KEY не заданы")
	}
	if region == "" {
		region = "us-east-1" // S3 SDK требует регион, для не-AWS значение не критично
	}

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("aws config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
		o.UsePathStyle = forcePathStyle
	})

	// Если public URL не задан — пробуем собрать
	if publicURL == "" && endpoint != "" {
		// Например: endpoint=https://storage.yandexcloud.net + bucket=lastop-uploads
		//   → https://lastop-uploads.storage.yandexcloud.net (virtual-hosted)
		// Если forcePathStyle — оставляем endpoint + bucket в пути:
		//   → https://storage.yandexcloud.net/lastop-uploads
		if forcePathStyle {
			publicURL = strings.TrimRight(endpoint, "/") + "/" + bucket
		} else {
			publicURL = injectBucketSubdomain(endpoint, bucket)
		}
	}

	return &S3Storage{
		client:    client,
		bucket:    bucket,
		endpoint:  endpoint,
		region:    region,
		publicURL: publicURL,
	}, nil
}

// injectBucketSubdomain превращает https://storage.yandexcloud.net + bucket=foo в https://foo.storage.yandexcloud.net
func injectBucketSubdomain(endpoint, bucket string) string {
	idx := strings.Index(endpoint, "://")
	if idx == -1 {
		return endpoint + "/" + bucket
	}
	scheme := endpoint[:idx+3]
	host := endpoint[idx+3:]
	host = strings.TrimRight(host, "/")
	return scheme + bucket + "." + host
}

func (s *S3Storage) Type() string { return "s3" }

func (s *S3Storage) Put(ctx context.Context, key string, r io.Reader, contentType string) (string, int64, error) {
	// Читаем в буфер для подсчёта размера и для S3 PutObject (требует ContentLength или seeker).
	// Для больших файлов это неоптимально; в будущем можно перейти на multipart upload.
	buf, err := io.ReadAll(r)
	if err != nil {
		return "", 0, fmt.Errorf("read body: %w", err)
	}
	size := int64(len(buf))

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        strings.NewReader(string(buf)),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", 0, fmt.Errorf("s3 put: %w", err)
	}
	return s.PublicURL(key), size, nil
}

func (s *S3Storage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

// Serve делает HTTP 302 редирект на публичный URL объекта в S3.
// Это правильный паттерн: трафик идёт мимо нашего сервера, файлы кэшируются браузерами и CDN.
func (s *S3Storage) Serve(w http.ResponseWriter, r *http.Request) {
	// Извлекаем ключ из URL: /uploads/2026/05/abc.jpg → 2026/05/abc.jpg
	key := strings.TrimPrefix(r.URL.Path, "/uploads/")
	if key == "" || key == r.URL.Path {
		http.NotFound(w, r)
		return
	}
	target := s.PublicURL(key)
	if target == "" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=86400, immutable")
	http.Redirect(w, r, target, http.StatusFound)
}

func (s *S3Storage) PublicURL(key string) string {
	if s.publicURL == "" {
		return ""
	}
	return s.publicURL + "/" + strings.TrimLeft(key, "/")
}
