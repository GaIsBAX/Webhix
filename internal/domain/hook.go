package domain

import (
	"fmt"
	"time"
)

type Hook struct {
	ID        int64
	Token     string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type WebhookRequest struct {
	ID          int64
	HookID      int64
	Method      string
	Path        string
	Query       string
	Headers     string
	Body        []byte
	RemoteAddr  string
	ContentType string
	BodySize    int64
	ReceivedAt  time.Time
}

type HookResponse struct {
	ID         int64
	HookID     int64
	StatusCode int64
	Headers    map[string]string
	Body       []byte
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type CreateWebhookRequestParams struct {
	HookID      int64
	Method      string
	Path        string
	Query       string
	Headers     string
	Body        []byte
	RemoteAddr  string
	ContentType string
	BodySize    int64
}

type UpsertHookResponseParams struct {
	StatusCode int64
	Headers    map[string]string
	Body       []byte
}

func (p UpsertHookResponseParams) Validate() error {
	if p.StatusCode < 100 || p.StatusCode > 599 {
		return fmt.Errorf("statusCode must be between 100 and 599")
	}
	return nil
}

type NotificationChannel struct {
	ID        int64
	HookID    int64
	Provider  string
	Config    map[string]string
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
