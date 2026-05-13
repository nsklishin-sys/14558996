// Package dadata — клиент для проверки ИНН через Dadata API.
// При отсутствии переменной DADATA_TOKEN используется NoopClient (всегда возвращает
// FoundFalse) — это позволяет запускать сервер локально без интеграции.
package dadata

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// Client — интерфейс проверки ИНН.
type Client interface {
	FindByINN(ctx context.Context, inn string) (*PartyInfo, error)
}

// PartyInfo — результат проверки ИНН в Dadata.
// Только нужные поля, остальные игнорируем.
type PartyInfo struct {
	Found     bool   `json:"found"`
	INN       string `json:"inn"`
	OGRN      string `json:"ogrn"`
	KPP       string `json:"kpp"`
	NameShort string `json:"name_short"`
	NameFull  string `json:"name_full"`
	OPF       string `json:"opf"`
	Address   string `json:"address"`
	Status    string `json:"status"` // ACTIVE / LIQUIDATING / LIQUIDATED / BANKRUPT / REORGANIZING
	Director  string `json:"director,omitempty"`
	RawJSON   string `json:"-"` // сохраняем полный ответ в БД
}

// New создаёт клиент. Если DADATA_TOKEN не задан — возвращает NoopClient.
func New() Client {
	token := strings.TrimSpace(os.Getenv("DADATA_TOKEN"))
	if token == "" {
		log.Printf("[dadata] DADATA_TOKEN не задан — используется NoopClient (ИНН не проверяется)")
		return &NoopClient{}
	}
	preview := token
	if len(preview) > 8 {
		preview = preview[:8] + "..."
	}
	log.Printf("[dadata] инициализирован клиент (token=%s)", preview)
	return &realClient{
		token: token,
		http: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

// NoopClient — заглушка, не делает реальных запросов.
type NoopClient struct{}

func (n *NoopClient) FindByINN(ctx context.Context, inn string) (*PartyInfo, error) {
	return &PartyInfo{Found: false}, nil
}

type realClient struct {
	token string
	http  *http.Client
}

const dadataURL = "https://suggestions.dadata.ru/suggestions/api/4_1/rs/findById/party"

func (c *realClient) FindByINN(ctx context.Context, inn string) (*PartyInfo, error) {
	inn = strings.TrimSpace(inn)
	if len(inn) != 10 && len(inn) != 12 {
		return &PartyInfo{Found: false}, nil
	}
	body := map[string]string{"query": inn}
	buf, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", dadataURL, bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Token "+c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dadata request: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dadata status %d: %s", resp.StatusCode, string(raw))
	}

	var parsed struct {
		Suggestions []struct {
			Value             string `json:"value"`
			UnrestrictedValue string `json:"unrestricted_value"`
			Data              struct {
				INN  string `json:"inn"`
				KPP  string `json:"kpp"`
				OGRN string `json:"ogrn"`
				OPF  struct {
					Short string `json:"short"`
					Full  string `json:"full"`
				} `json:"opf"`
				Name struct {
					ShortWithOPF string `json:"short_with_opf"`
					FullWithOPF  string `json:"full_with_opf"`
					Short        string `json:"short"`
					Full         string `json:"full"`
				} `json:"name"`
				Address struct {
					Value             string `json:"value"`
					UnrestrictedValue string `json:"unrestricted_value"`
				} `json:"address"`
				State struct {
					Status string `json:"status"`
				} `json:"state"`
				Management struct {
					Name string `json:"name"`
					Post string `json:"post"`
				} `json:"management"`
			} `json:"data"`
		} `json:"suggestions"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("parse dadata: %w", err)
	}
	if len(parsed.Suggestions) == 0 {
		return &PartyInfo{Found: false, RawJSON: string(raw)}, nil
	}
	s := parsed.Suggestions[0]
	name := s.Data.Name.ShortWithOPF
	if name == "" {
		name = s.Data.Name.FullWithOPF
	}
	if name == "" {
		name = s.Value
	}
	_ = name
	return &PartyInfo{
		Found:     true,
		INN:       s.Data.INN,
		OGRN:      s.Data.OGRN,
		KPP:       s.Data.KPP,
		NameShort: s.Data.Name.Short,
		NameFull:  s.Data.Name.Full,
		OPF:       s.Data.OPF.Short,
		Address:   s.Data.Address.UnrestrictedValue,
		Status:    s.Data.State.Status,
		Director:  s.Data.Management.Name,
		RawJSON:   string(raw),
	}, nil
}
