// Package pushnotify — отправка web-push уведомлений всем подпискам пользователя.
// Использует VAPID-ключи из env (VAPID_PUBLIC_KEY, VAPID_PRIVATE_KEY, VAPID_SUBJECT).
// На 410/404 от push-сервиса удаляет мёртвую подписку из БД.
package pushnotify

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
)

// Payload — данные одного push-уведомления.
// URL формируется выше по стеку (роутинг по типу уведомления), сюда
// приходит уже готовый.
type Payload struct {
	Title     string `json:"title"`
	Body      string `json:"body"`
	Icon      string `json:"icon,omitempty"`
	Badge     string `json:"badge,omitempty"`
	Tag       string `json:"tag,omitempty"`
	URL       string `json:"-"` // не сериализуется напрямую, кладётся в data ниже
	NotifType string `json:"-"`
}

// payloadJSON — то, что реально отправляется в браузер.
// Соответствует структуре, которую обрабатывает self.addEventListener('push') в /sw.js (P3).
type payloadJSON struct {
	Title string         `json:"title"`
	Body  string         `json:"body"`
	Icon  string         `json:"icon,omitempty"`
	Badge string         `json:"badge,omitempty"`
	Tag   string         `json:"tag,omitempty"`
	Data  map[string]any `json:"data,omitempty"`
}

// URLForNotification возвращает URL для notification.data.url
// на основе типа уведомления и source_public_id.
// Маршрутизация совпадает с обработчиком notificationclick в /sw.js.
func URLForNotification(notifType, sourceType, sourcePublicID string) string {
	switch sourceType {
	case "chat":
		if sourcePublicID != "" {
			return "/chat.html?conv=" + sourcePublicID
		}
		return "/chat.html"
	case "post":
		if sourcePublicID != "" {
			return "/dashboard.html?post=" + sourcePublicID
		}
	case "comment":
		if sourcePublicID != "" {
			return "/dashboard.html?post=" + sourcePublicID
		}
	case "event":
		if sourcePublicID != "" {
			return "/event-detail.html?id=" + sourcePublicID
		}
	case "company":
		if sourcePublicID != "" {
			return "/company-detail.html?id=" + sourcePublicID
		}
	case "exhibition":
		if sourcePublicID != "" {
			return "/exhibition-detail.html?id=" + sourcePublicID
		}
	case "project":
		if sourcePublicID != "" {
			return "/project-detail.html?id=" + sourcePublicID
		}
	}
	// По типу уведомления (если source_type не информативен)
	switch notifType {
	case "friend_request", "friend_accepted":
		return "/friends.html"
	}
	return "/notifications.html"
}

// SendToUser отправляет web-push всем активным подпискам пользователя.
// Вызывается асинхронно (go SendToUser(...)) — не возвращает ошибки наверх.
// Все ошибки логирует через log.Printf.
//
// На 410 Gone / 404 Not Found — удаляет подписку (endpoint умер).
// На другие ошибки — пишет last_error_at/last_error_msg для последующего анализа.
func SendToUser(ctx context.Context, db *sql.DB, userID int64, p Payload) {
	publicKey := strings.TrimSpace(os.Getenv("VAPID_PUBLIC_KEY"))
	privateKey := strings.TrimSpace(os.Getenv("VAPID_PRIVATE_KEY"))
	subject := strings.TrimSpace(os.Getenv("VAPID_SUBJECT"))
	if publicKey == "" || privateKey == "" || subject == "" {
		return // push не сконфигурирован — тихо выходим
	}

	// Тайм-аут на весь процесс отправки одному юзеру — на случай зависших http-запросов.
	sendCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rows, err := db.QueryContext(sendCtx, `
		SELECT id, endpoint, p256dh_key, auth_key
		FROM push_subscriptions
		WHERE user_id = $1
	`, userID)
	if err != nil {
		log.Printf("pushnotify: list subscriptions for user=%d: %v", userID, err)
		return
	}
	type sub struct {
		id       int64
		endpoint string
		p256dh   string
		auth     string
	}
	var subs []sub
	for rows.Next() {
		var s sub
		if err := rows.Scan(&s.id, &s.endpoint, &s.p256dh, &s.auth); err == nil {
			subs = append(subs, s)
		}
	}
	rows.Close()
	if len(subs) == 0 {
		return
	}

	body := payloadJSON{
		Title: p.Title,
		Body:  p.Body,
		Icon:  defaultIfEmpty(p.Icon, "/assets/icon-192.png"),
		Badge: defaultIfEmpty(p.Badge, "/assets/icon-192.png"),
		Tag:   p.Tag,
		Data: map[string]any{
			"url":  p.URL,
			"type": p.NotifType,
		},
	}
	msg, err := json.Marshal(body)
	if err != nil {
		log.Printf("pushnotify: marshal payload: %v", err)
		return
	}

	// webpush-go v1.4.0 в vapid.go:76 безусловно добавляет "mailto:" если
	// subject не начинается с "https:" — даже если "mailto:" там уже есть.
	// Из-за этого Apple отбивает push как BadJwtToken. Срезаем префикс
	// если он есть — библиотека сама его допишет.
	normalizedSubject := subject
	if strings.HasPrefix(normalizedSubject, "mailto:") {
		normalizedSubject = strings.TrimPrefix(normalizedSubject, "mailto:")
	}
	opts := &webpush.Options{
		Subscriber:      normalizedSubject,
		VAPIDPublicKey:  publicKey,
		VAPIDPrivateKey: privateKey,
		TTL:             3600, // 1 час — push-сервер хранит сообщение если устройство офлайн
	}

	for _, s := range subs {
		wpSub := &webpush.Subscription{
			Endpoint: s.endpoint,
			Keys: webpush.Keys{
				P256dh: s.p256dh,
				Auth:   s.auth,
			},
		}
		resp, err := webpush.SendNotificationWithContext(sendCtx, msg, wpSub, opts)
		if err != nil {
			markError(db, s.id, "send: "+err.Error())
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent:
			// success
		case http.StatusGone, http.StatusNotFound:
			// Подписка мертва — удаляем
			_, _ = db.ExecContext(sendCtx, `DELETE FROM push_subscriptions WHERE id = $1`, s.id)
			log.Printf("pushnotify: removed dead subscription id=%d status=%d", s.id, resp.StatusCode)
		default:
			markError(db, s.id, "status: "+resp.Status)
		}
	}
}

func markError(db *sql.DB, subID int64, msg string) {
	if len(msg) > 500 {
		msg = msg[:500]
	}
	_, _ = db.Exec(`
		UPDATE push_subscriptions
		SET last_error_at = NOW(), last_error_msg = $1
		WHERE id = $2
	`, msg, subID)
	log.Printf("pushnotify: subscription id=%d error: %s", subID, msg)
}

func defaultIfEmpty(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}
