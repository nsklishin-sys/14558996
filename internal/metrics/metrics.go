// Package metrics — простой in-memory счётчик HTTP-запросов и ошибок.
// Не требует внешних зависимостей. Если нужны Prometheus-совместимые
// метрики — позже подключить prometheus/client_golang в эту же абстракцию.
package metrics

import (
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Registry хранит метрики платформы.
type Registry struct {
	startedAt time.Time

	// Глобальные счётчики
	totalRequests atomic.Int64
	totalErrors   atomic.Int64
	totalPanics   atomic.Int64

	// Счётчики по path-bucket: ключ — нормализованный путь (например /api/users/{id}).
	mu    sync.RWMutex
	paths map[string]*PathStat
}

// PathStat — метрики по одному endpoint.
type PathStat struct {
	Count      atomic.Int64
	Errors     atomic.Int64 // 5xx
	ClientErr  atomic.Int64 // 4xx
	DurationUs atomic.Int64 // суммарное время в микросекундах
}

// New создаёт пустой реестр.
func New() *Registry {
	return &Registry{
		startedAt: time.Now(),
		paths:     map[string]*PathStat{},
	}
}

// RecordRequest регистрирует завершённый HTTP-запрос.
func (r *Registry) RecordRequest(path string, status int, duration time.Duration) {
	r.totalRequests.Add(1)
	if status >= 500 {
		r.totalErrors.Add(1)
	}
	bucket := normalizePath(path)
	r.mu.RLock()
	stat, ok := r.paths[bucket]
	r.mu.RUnlock()
	if !ok {
		r.mu.Lock()
		stat, ok = r.paths[bucket]
		if !ok {
			stat = &PathStat{}
			r.paths[bucket] = stat
		}
		r.mu.Unlock()
	}
	stat.Count.Add(1)
	if status >= 500 {
		stat.Errors.Add(1)
	} else if status >= 400 {
		stat.ClientErr.Add(1)
	}
	stat.DurationUs.Add(duration.Microseconds())
}

// RecordPanic — отдельный счётчик паник (вне HTTP-цикла тоже учитывается).
func (r *Registry) RecordPanic() {
	r.totalPanics.Add(1)
}

// Snapshot возвращает текущее состояние для отдачи в /api/admin/metrics.
type Snapshot struct {
	UptimeSeconds int64          `json:"uptime_seconds"`
	StartedAt     time.Time      `json:"started_at"`
	TotalRequests int64          `json:"total_requests"`
	TotalErrors   int64          `json:"total_errors"`
	TotalPanics   int64          `json:"total_panics"`
	TopPaths      []SnapshotPath `json:"top_paths"`
}

type SnapshotPath struct {
	Path          string  `json:"path"`
	Count         int64   `json:"count"`
	Errors        int64   `json:"errors"`
	ClientErrors  int64   `json:"client_errors"`
	AvgDurationMs float64 `json:"avg_duration_ms"`
}

func (r *Registry) Snapshot() Snapshot {
	snap := Snapshot{
		UptimeSeconds: int64(time.Since(r.startedAt).Seconds()),
		StartedAt:     r.startedAt,
		TotalRequests: r.totalRequests.Load(),
		TotalErrors:   r.totalErrors.Load(),
		TotalPanics:   r.totalPanics.Load(),
	}
	r.mu.RLock()
	paths := make([]SnapshotPath, 0, len(r.paths))
	for path, stat := range r.paths {
		count := stat.Count.Load()
		var avgMs float64
		if count > 0 {
			avgMs = float64(stat.DurationUs.Load()) / float64(count) / 1000.0
		}
		paths = append(paths, SnapshotPath{
			Path:          path,
			Count:         count,
			Errors:        stat.Errors.Load(),
			ClientErrors:  stat.ClientErr.Load(),
			AvgDurationMs: avgMs,
		})
	}
	r.mu.RUnlock()
	// Топ по количеству вызовов
	sort.Slice(paths, func(i, j int) bool { return paths[i].Count > paths[j].Count })
	if len(paths) > 50 {
		paths = paths[:50]
	}
	snap.TopPaths = paths
	return snap
}

// normalizePath группирует похожие URL в один bucket.
// Например /api/users/123 → /api/users/{id}, /uploads/2026/05/abc.jpg → /uploads/{file}.
func normalizePath(p string) string {
	if strings.HasPrefix(p, "/uploads/") {
		return "/uploads/{file}"
	}
	if strings.HasPrefix(p, "/assets/") {
		return "/assets/{file}"
	}
	parts := strings.Split(strings.Trim(p, "/"), "/")
	out := make([]string, len(parts))
	for i, part := range parts {
		// Числовые ID
		if isAllDigits(part) {
			out[i] = "{id}"
			continue
		}
		// Хексовые/UUID-подобные
		if len(part) >= 16 && isHexLike(part) {
			out[i] = "{hex}"
			continue
		}
		out[i] = part
	}
	return "/" + strings.Join(out, "/")
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func isHexLike(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') || c == '-') {
			return false
		}
	}
	return true
}
