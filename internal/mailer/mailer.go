// Package mailer содержит абстракцию для отправки email.
// Без переменной окружения SMTP_HOST используется LogMailer, который пишет
// письма в stdout — удобно для разработки. С SMTP_HOST подключается реальный
// SMTPMailer.
package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"strings"
	"time"
)

// Mailer — интерфейс отправки писем.
type Mailer interface {
	Send(ctx context.Context, to, subject, htmlBody string) error
}

// New возвращает Mailer на основе env-переменных:
//
//	SMTP_HOST     — хост SMTP-сервера (если пусто → LogMailer)
//	SMTP_PORT     — порт (по умолчанию 587)
//	SMTP_USER     — логин
//	SMTP_PASSWORD — пароль (или app-password)
//	SMTP_FROM     — адрес отправителя (по умолчанию = SMTP_USER)
//	SMTP_FROM_NAME — отображаемое имя (по умолчанию "LASTOP GROUP")
func New() Mailer {
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	if host == "" {
		log.Printf("[mailer] SMTP_HOST не задан, используется LogMailer (письма пишутся в stdout)")
		return &LogMailer{}
	}
	port := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	if port == "" {
		port = "587"
	}
	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	if from == "" {
		from = strings.TrimSpace(os.Getenv("SMTP_USER"))
	}
	fromName := strings.TrimSpace(os.Getenv("SMTP_FROM_NAME"))
	if fromName == "" {
		fromName = "LASTOP GROUP"
	}
	return &SMTPMailer{
		host:     host,
		port:     port,
		user:     strings.TrimSpace(os.Getenv("SMTP_USER")),
		password: os.Getenv("SMTP_PASSWORD"),
		from:     from,
		fromName: fromName,
	}
}

// LogMailer пишет письма в stdout (для разработки и пока SMTP не подключён).
type LogMailer struct{}

func (m *LogMailer) Send(ctx context.Context, to, subject, htmlBody string) error {
	log.Printf("[mailer:log] ─── EMAIL ───\nTo: %s\nSubject: %s\n%s\n────────────", to, subject, htmlBody)
	return nil
}

// SMTPMailer отправляет письма через реальный SMTP-сервер.
type SMTPMailer struct {
	host     string
	port     string
	user     string
	password string
	from     string
	fromName string
}

func (m *SMTPMailer) Send(ctx context.Context, to, subject, htmlBody string) error {
	addr := m.host + ":" + m.port
	auth := smtp.PlainAuth("", m.user, m.password, m.host)

	headers := map[string]string{
		"From":         fmt.Sprintf("%s <%s>", m.fromName, m.from),
		"To":           to,
		"Subject":      mimeEncode(subject),
		"MIME-Version": "1.0",
		"Content-Type": "text/html; charset=UTF-8",
		"Date":         time.Now().Format(time.RFC1123Z),
	}
	var msg strings.Builder
	for k, v := range headers {
		msg.WriteString(k)
		msg.WriteString(": ")
		msg.WriteString(v)
		msg.WriteString("\r\n")
	}
	msg.WriteString("\r\n")
	msg.WriteString(htmlBody)

	// Отдельный канал чтобы уважать ctx
	errCh := make(chan error, 1)
	go func() {
		// Yandex 365/Mail требует TLS на 465 (SSL) или STARTTLS на 587.
		// Используем стандартный smtp.SendMail — он сам выбирает STARTTLS,
		// если сервер его поддерживает (587). Для 465 (SSL) нужен другой код.
		if m.port == "465" {
			errCh <- m.sendSSL(addr, auth, to, msg.String())
			return
		}
		errCh <- smtp.SendMail(addr, auth, m.from, []string{to}, []byte(msg.String()))
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("smtp send: %w", err)
		}
		return nil
	}
}

func (m *SMTPMailer) sendSSL(addr string, auth smtp.Auth, to, msg string) error {
	tlsCfg := &tls.Config{ServerName: m.host}
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	defer conn.Close()
	c, err := smtp.NewClient(conn, m.host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer c.Quit()
	if err := c.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err := c.Mail(m.from); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return err
	}
	return w.Close()
}

// mimeEncode кодирует тему письма в RFC 2047 (для не-ASCII).
func mimeEncode(s string) string {
	if isASCII(s) {
		return s
	}
	// "=?UTF-8?B?<base64>?="
	return "=?UTF-8?B?" + base64Encode([]byte(s)) + "?="
}

func isASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return false
		}
	}
	return true
}

func base64Encode(b []byte) string {
	const enc = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	out := make([]byte, 0, ((len(b)+2)/3)*4)
	for i := 0; i < len(b); i += 3 {
		var triple uint32
		n := 0
		for j := 0; j < 3 && i+j < len(b); j++ {
			triple |= uint32(b[i+j]) << uint((2-j)*8)
			n++
		}
		for j := 0; j < 4; j++ {
			if j <= n {
				out = append(out, enc[(triple>>uint((3-j)*6))&0x3F])
			} else {
				out = append(out, '=')
			}
		}
	}
	return string(out)
}
