// Package errtrack — абстракция для отправки ошибок и паник во внешний трекер.
// Без переменной окружения SENTRY_DSN используется NoopTracker (ничего не делает).
// При SENTRY_DSN=https://...@sentry.io/... включается SentryTracker.
package errtrack

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
)

// Tracker — интерфейс трекера ошибок.
type Tracker interface {
	// Capture отправляет error в трекер. Безопасен для горутин.
	Capture(err error, tags map[string]string)
	// CaptureMessage отправляет произвольное сообщение (например warning).
	CaptureMessage(msg string, tags map[string]string)
	// Flush блокирует до отправки накопленного буфера или таймаута. Вызывается перед shutdown.
	Flush(timeout time.Duration)
	// Type возвращает имя реализации.
	Type() string
}

// New возвращает реализацию по env:
//
//	SENTRY_DSN=https://...   — включает SentryTracker
//	SENTRY_ENVIRONMENT=prod  — окружение (для фильтра в Sentry)
//	SENTRY_RELEASE=1.0.0     — версия (для группировки)
func New() Tracker {
	dsn := strings.TrimSpace(os.Getenv("SENTRY_DSN"))
	if dsn == "" {
		log.Printf("[errtrack] инициализирован NoopTracker (SENTRY_DSN не задан)")
		return &NoopTracker{}
	}
	env := strings.TrimSpace(os.Getenv("SENTRY_ENVIRONMENT"))
	if env == "" {
		env = "production"
	}
	release := strings.TrimSpace(os.Getenv("SENTRY_RELEASE"))
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      env,
		Release:          release,
		AttachStacktrace: true,
		// 0.1 = 10% запросов в performance-trace. Для критичных стартов оставим 1.0.
		TracesSampleRate: 0.1,
	})
	if err != nil {
		log.Printf("[errtrack] sentry.Init failed: %v — фолбэк на NoopTracker", err)
		return &NoopTracker{}
	}
	log.Printf("[errtrack] инициализирован SentryTracker (env=%s release=%s)", env, release)
	return &SentryTracker{}
}

// ─────────── Noop ───────────

type NoopTracker struct{}

func (n *NoopTracker) Capture(err error, tags map[string]string)         {}
func (n *NoopTracker) CaptureMessage(msg string, tags map[string]string) {}
func (n *NoopTracker) Flush(timeout time.Duration)                       {}
func (n *NoopTracker) Type() string                                      { return "noop" }

// ─────────── Sentry ───────────

type SentryTracker struct{}

func (s *SentryTracker) Type() string { return "sentry" }

func (s *SentryTracker) Capture(err error, tags map[string]string) {
	if err == nil {
		return
	}
	hub := sentry.CurrentHub().Clone()
	hub.WithScope(func(scope *sentry.Scope) {
		for k, v := range tags {
			scope.SetTag(k, v)
		}
		hub.CaptureException(err)
	})
}

func (s *SentryTracker) CaptureMessage(msg string, tags map[string]string) {
	if msg == "" {
		return
	}
	hub := sentry.CurrentHub().Clone()
	hub.WithScope(func(scope *sentry.Scope) {
		for k, v := range tags {
			scope.SetTag(k, v)
		}
		hub.CaptureMessage(msg)
	})
}

func (s *SentryTracker) Flush(timeout time.Duration) {
	sentry.Flush(timeout)
}

// CaptureFromContext — удобный хелпер для вызова из обычных мест.
// Не использует context.Context напрямую, оставлен для совместимости с возможным расширением.
func CaptureFromContext(ctx context.Context, t Tracker, err error, tags map[string]string) {
	t.Capture(err, tags)
}
