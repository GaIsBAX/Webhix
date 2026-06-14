package server

import (
	"encoding/json"
	"time"

	"github.com/GaIsBAX/Webhix/internal/domain"
	"github.com/GaIsBAX/Webhix/pkg"
)

type CreateEndpointRequestContract struct {
	Name string `json:"name"`
}

type CreateEndpointResponseContract struct {
	ID    int64  `json:"id"`
	Token string `json:"token"`
	Name  string `json:"name,omitempty"`
	URL   string `json:"url"`
}

type EndpointListItemContract struct {
	ID        int64     `json:"id"`
	Token     string    `json:"token"`
	Name      string    `json:"name,omitempty"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"createdAt"`
}

type WebhookRequestContract struct {
	ID          int64     `json:"id"`
	HookID      int64     `json:"hookId"`
	Method      string    `json:"method"`
	Path        string    `json:"path"`
	Query       string    `json:"query,omitempty"`
	Headers     string    `json:"headers"`
	Body        []byte    `json:"body,omitempty"`
	RemoteAddr  string    `json:"remoteAddr,omitempty"`
	ContentType string    `json:"contentType,omitempty"`
	BodySize    int64     `json:"bodySize"`
	ReceivedAt  time.Time `json:"receivedAt"`
}

type HookResponseContract struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

type NotificationContract struct {
	Provider string            `json:"provider"`
	Config   map[string]string `json:"config"`
	Redacted []string          `json:"redacted,omitempty"`
}

type SetHookResponseRequestContract struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

type ResponseContract struct {
	Success   bool            `json:"success"`
	Data      json.RawMessage `json:"body,omitempty"`
	Error     *ErrorContract  `json:"error,omitempty"`
	RequestID string          `json:"requestID"`
	Timestamp time.Time       `json:"timestamp"`
}

type ErrorContract struct {
	Code    string                `json:"code"`
	Message string                `json:"message"`
	Details []ErrorDetailContract `json:"details,omitempty"`
}

type ErrorDetailContract struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

func NewSuccessResponseContract(data []byte) *ResponseContract {
	return &ResponseContract{
		Success:   true,
		Data:      json.RawMessage(data),
		RequestID: pkg.GeneratePrefixedString("re"),
		Timestamp: time.Now().UTC(),
	}
}

func NewErrorResponseContract(apiErr ErrorContract) *ResponseContract {
	return &ResponseContract{
		Success:   false,
		Error:     &apiErr,
		RequestID: pkg.GeneratePrefixedString("re"),
		Timestamp: time.Now().UTC(),
	}
}

func WithDetails(err ErrorContract, details ...ErrorDetailContract) ErrorContract {
	err.Details = details
	return err
}

func toHookResponseContract(resp domain.HookResponse) HookResponseContract {
	headers := resp.Headers
	if headers == nil {
		headers = map[string]string{}
	}
	return HookResponseContract{
		StatusCode: int(resp.StatusCode),
		Headers:    headers,
		Body:       string(resp.Body),
	}
}

func toWebhookRequestContract(req domain.WebhookRequest) WebhookRequestContract {
	return WebhookRequestContract{
		ID:          req.ID,
		HookID:      req.HookID,
		Method:      req.Method,
		Path:        req.Path,
		Query:       req.Query,
		Headers:     req.Headers,
		Body:        req.Body,
		RemoteAddr:  req.RemoteAddr,
		ContentType: req.ContentType,
		BodySize:    req.BodySize,
		ReceivedAt:  req.ReceivedAt,
	}
}

func toNotificationContract(ch domain.NotificationChannel, secretKeys []string) NotificationContract {
	secretSet := make(map[string]bool, len(secretKeys))
	for _, s := range secretKeys {
		secretSet[s] = true
	}

	cfg := make(map[string]string, len(ch.Config))
	var redacted []string
	for k, v := range ch.Config {
		if secretSet[k] && v != "" {
			redacted = append(redacted, k)
		} else {
			cfg[k] = v
		}
	}

	return NotificationContract{Provider: ch.Provider, Config: cfg, Redacted: redacted}
}
