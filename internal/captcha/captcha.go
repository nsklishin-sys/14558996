// Package captcha — абстракция для проверки капчи.
// Без переменной окружения CAPTCHA_TYPE=yandex используется NoopCaptcha
// (всегда возвращает true). Это нормально для разработки и для текущей
// клиентской math-капчи (которая работает чисто на фронте).
//
// При CAPTCHA_TYPE=yandex включается YandexSmartCaptcha с проверкой токена
// через Yandex Cloud API.
package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Captcha — интерфейс проверки капчи.
type Captcha interface {
	Verify(ctx context.Context, token, ip string) (bool, error)
	Type() string
	SiteKey() string
}

// New возвращает реализацию по env:
//
//	CAPTCHA_TYPE=yandex     — Yandex SmartCaptcha
//	YANDEX_CAPTCHA_SITEKEY  — публичный ключ виджета (на фронте)
//	YANDEX_CAPTCHA_SECRET   — серверный ключ (для проверки)
func New() Captcha {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("CAPTCHA_TYPE")), "yandex") {
		siteKey := strings.TrimSpace(os.Getenv("YANDEX_CAPTCHA_SITEKEY"))
		secret := strings.TrimSpace(os.Getenv("YANDEX_CAPTCHA_SECRET"))
		if siteKey == "" || secret == "" {
			log.Printf("[captcha] CAPTCHA_TYPE=yandex, но YANDEX_CAPTCHA_SITEKEY/SECRET не заданы — используется Noop")
		} else {
			preview := siteKey
			if len(preview) > 8 {
				preview = preview[:8] + "..."
			}
			log.Printf("[captcha] инициализирован YandexSmartCaptcha (sitekey=%s)", preview)
			return &YandexSmartCaptcha{siteKey: siteKey, secret: secret}
		}
	}
	log.Printf("[captcha] инициализирован NoopCaptcha (всегда true)")
	return &NoopCaptcha{}
}

// ─────────── Noop ───────────

type NoopCaptcha struct{}

func (n *NoopCaptcha) Verify(ctx context.Context, token, ip string) (bool, error) {
	return true, nil
}
func (n *NoopCaptcha) Type() string    { return "noop" }
func (n *NoopCaptcha) SiteKey() string { return "" }

// ─────────── Yandex SmartCaptcha ───────────

type YandexSmartCaptcha struct {
	siteKey string
	secret  string
}

func (y *YandexSmartCaptcha) Type() string    { return "yandex" }
func (y *YandexSmartCaptcha) SiteKey() string { return y.siteKey }

// https://yandex.cloud/ru/docs/smartcaptcha/concepts/validation
func (y *YandexSmartCaptcha) Verify(ctx context.Context, token, ip string) (bool, error) {
	if token == "" {
		return false, fmt.Errorf("empty captcha token")
	}
	form := url.Values{}
	form.Set("secret", y.secret)
	form.Set("token", token)
	if ip != "" {
		form.Set("ip", ip)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", "https://smartcaptcha.yandexcloud.net/validate", strings.NewReader(form.Encode()))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("captcha api request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var r struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Host    string `json:"host"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return false, fmt.Errorf("captcha api parse: %w body=%s", err, string(body))
	}
	return r.Status == "ok", nil
}
