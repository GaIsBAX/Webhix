package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"
)

var (
	defaultTelegramClient = &http.Client{Timeout: 10 * time.Second}
	proxyClients          sync.Map
)

func NewTelegramProvider() Provider {
	return telegramProvider{}
}

type telegramProvider struct{}

func (telegramProvider) Send(ctx context.Context, config Config, message string) error {
	return sendMessage(ctx, config["bot_token"], config["chat_id"], message, config["proxy_url"])
}

func (telegramProvider) ValidateConfig(config Config) error {
	for _, k := range []string{"bot_token", "chat_id"} {
		if config[k] == "" {
			return fmt.Errorf("telegram: %q is required", k)
		}
	}
	return nil
}

func (telegramProvider) SecretKeys() []string {
	return []string{"bot_token"}
}

func sendMessage(ctx context.Context, botToken, chatID, text, proxyURL string) error {
	if botToken == "" || chatID == "" {
		return fmt.Errorf("telegram: bot_token and chat_id are required")
	}

	client := defaultTelegramClient
	if proxyURL != "" {
		if v, ok := proxyClients.Load(proxyURL); ok {
			if c, ok := v.(*http.Client); ok {
				client = c
			}
		} else {
			proxy, err := validateProxy(proxyURL)
			if err != nil {
				return err
			}
			c := &http.Client{
				Timeout:   10 * time.Second,
				Transport: &http.Transport{Proxy: http.ProxyURL(proxy)},
			}
			proxyClients.Store(proxyURL, c)
			client = c
		}
	}

	// text already contains HTML from the caller (handler.go escapes user data before calling Send)
	payload, err := json.Marshal(map[string]string{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML",
	})
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("close response body", "err", err)
		}
	}()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	var tgErr struct {
		Description string `json:"description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tgErr); err == nil && tgErr.Description != "" {
		return fmt.Errorf("telegram: %s", tgErr.Description)
	}
	return fmt.Errorf("telegram API returned %d", resp.StatusCode)
}

// validateProxy parses and validates the proxy URL scheme.
// Returns the parsed URL so the caller doesn't parse it twice.
func validateProxy(rawURL string) (*url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL: %w", err)
	}
	switch u.Scheme {
	case "http", "https", "socks5":
		return u, nil
	default:
		return nil, fmt.Errorf("proxy scheme %q not allowed: use http, https, or socks5", u.Scheme)
	}
}
