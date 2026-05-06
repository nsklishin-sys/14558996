// migrate-to-s3 — одноразовый CLI-скрипт для переноса файлов из локального диска в S3.
//
// Использование (все env переменные S3_* должны быть заданы):
//
//	go run ./scripts/migrate-to-s3 [--dir /data/uploads] [--dry-run]
//
// Скрипт обходит локальный каталог рекурсивно и заливает каждый файл в S3-bucket
// под тем же относительным путём. После успешной загрузки локальный файл НЕ удаляется
// (для безопасности — удалить руками после проверки).
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"greeting-site/internal/storage"
)

func main() {
	var (
		dir     = flag.String("dir", "/data/uploads", "локальная директория с файлами")
		dryRun  = flag.Bool("dry-run", false, "только показать что будет загружено, без реальной загрузки")
		verbose = flag.Bool("v", false, "подробный лог")
	)
	flag.Parse()

	// Принудительно включаем S3
	if os.Getenv("STORAGE_TYPE") != "s3" {
		log.Fatal("STORAGE_TYPE должен быть = s3 для миграции")
	}

	store := storage.New()
	if store.Type() != "s3" {
		log.Fatal("storage.New() вернул не S3 — проверь S3_* env-переменные")
	}

	if _, err := os.Stat(*dir); os.IsNotExist(err) {
		log.Fatalf("директория %s не существует", *dir)
	}

	var (
		uploaded int
		skipped  int
		failed   int
		bytes    int64
	)
	start := time.Now()

	err := filepath.Walk(*dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		// Относительный путь = ключ в S3
		rel, err := filepath.Rel(*dir, path)
		if err != nil {
			return err
		}
		key := filepath.ToSlash(rel)

		if *dryRun {
			fmt.Printf("[dry-run] would upload: %s (%d bytes)\n", key, info.Size())
			uploaded++
			bytes += info.Size()
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			log.Printf("FAIL open %s: %v", path, err)
			failed++
			return nil
		}
		defer f.Close()

		// Пытаемся определить content-type по расширению
		ct := contentTypeByExt(filepath.Ext(path))

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		_, _, err = store.Put(ctx, key, f, ct)
		cancel()
		if err != nil {
			log.Printf("FAIL upload %s: %v", key, err)
			failed++
			return nil
		}
		uploaded++
		bytes += info.Size()
		if *verbose {
			fmt.Printf("OK %s (%d bytes)\n", key, info.Size())
		}
		return nil
	})
	if err != nil {
		log.Fatalf("walk error: %v", err)
	}

	dur := time.Since(start)
	fmt.Printf("\n────────── миграция завершена за %s ──────────\n", dur)
	fmt.Printf("загружено: %d файлов, %.2f МБ\n", uploaded, float64(bytes)/(1024*1024))
	if skipped > 0 {
		fmt.Printf("пропущено: %d\n", skipped)
	}
	if failed > 0 {
		fmt.Printf("ОШИБОК: %d (см. лог выше)\n", failed)
		os.Exit(1)
	}
	if !*dryRun {
		fmt.Printf("\nЛокальные файлы НЕ удалены — проверьте загрузку и удалите %s вручную.\n", *dir)
	}
}

func contentTypeByExt(ext string) string {
	ext = strings.ToLower(ext)
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mp3":
		return "audio/mpeg"
	case ".ogg":
		return "audio/ogg"
	case ".pdf":
		return "application/pdf"
	case ".txt":
		return "text/plain; charset=utf-8"
	case ".json":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}
