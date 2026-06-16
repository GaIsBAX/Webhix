package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log/slog"
	"net/http"

	"github.com/GaIsBAX/Webhix/internal/domain"
	"github.com/GaIsBAX/Webhix/internal/notify"
)

const (
	DefaultMaxBodySize         int64 = 5 << 20 // 5MB
	maxConcurrentNotifications       = 64
)

type HookService interface {
	ListHooks(ctx context.Context) ([]domain.Hook, error)
	CreateHook(ctx context.Context, name string) (domain.Hook, error)
	ReceiveWebhook(ctx context.Context, token string, params domain.CreateWebhookRequestParams) (domain.WebhookRequest, domain.HookResponse, error)
	ListWebhookRequests(ctx context.Context, token string) ([]domain.WebhookRequest, error)
	GetHookResponse(ctx context.Context, token string) (domain.HookResponse, error)
	SetHookResponse(ctx context.Context, token string, params domain.UpsertHookResponseParams) (domain.HookResponse, error)
	ListChannels(ctx context.Context, token string) ([]domain.NotificationChannel, error)
	UpsertChannel(ctx context.Context, token, provider string, config map[string]string) (domain.NotificationChannel, error)
	DeleteChannel(ctx context.Context, token, provider string) error
	GetChannelsForHookID(ctx context.Context, hookID int64) ([]domain.NotificationChannel, error)
}

type EventBroker interface {
	Done() <-chan struct{}
	Subscribe(token string) (<-chan []byte, func())
	Publish(token string, data []byte)
}

type HookOptions struct {
	BaseURL     string
	MaxBodySize int64
	ReadOnly    bool
}

type HookDeps struct {
	Mux     *http.ServeMux
	Service HookService
	Hub     EventBroker
	Opts    HookOptions
}

type Hook struct {
	deps      *HookDeps
	notifySem chan struct{}
}

func NewHook(deps *HookDeps) *Hook {
	if deps.Opts.MaxBodySize <= 0 {
		deps.Opts.MaxBodySize = DefaultMaxBodySize
	}

	return &Hook{deps: deps, notifySem: make(chan struct{}, maxConcurrentNotifications)}
}

func (h *Hook) RegisterRoutes() {
	h.deps.Mux.HandleFunc("GET /api/endpoints", h.ListEndpoints)
	h.deps.Mux.HandleFunc("POST /api/endpoints", h.CreateEndpoint)
	h.deps.Mux.HandleFunc("GET /api/endpoints/{token}/requests", h.ListRequests)
	h.deps.Mux.HandleFunc("GET /api/endpoints/{token}/events", h.StreamEvents)
	h.deps.Mux.HandleFunc("GET /api/endpoints/{token}/response", h.GetResponse)
	h.deps.Mux.HandleFunc("PUT /api/endpoints/{token}/response", h.SetResponse)
	h.deps.Mux.HandleFunc("GET /api/endpoints/{token}/notifications", h.GetNotification)
	h.deps.Mux.HandleFunc("PUT /api/endpoints/{token}/notifications/{provider}", h.SetNotification)
	h.deps.Mux.HandleFunc("DELETE /api/endpoints/{token}/notifications/{provider}", h.DeleteNotification)
	h.deps.Mux.HandleFunc("POST /api/endpoints/{token}/notifications/{provider}/test", h.TestNotification)
	h.deps.Mux.HandleFunc("/r/{token}", h.ReceiveWebhook)
}

func (h *Hook) ListEndpoints(w http.ResponseWriter, r *http.Request) {
	hooks, err := h.deps.Service.ListHooks(r.Context())
	if err != nil {
		slog.Error("list endpoints", "err", err)
		SendError(w, http.StatusInternalServerError, ErrInternal)
		return
	}

	contracts := make([]EndpointListItemContract, len(hooks))
	for i, hook := range hooks {
		contracts[i] = EndpointListItemContract{
			ID:        hook.ID,
			Token:     hook.Token,
			Name:      hook.Name,
			URL:       h.deps.Opts.BaseURL + "/r/" + hook.Token,
			CreatedAt: hook.CreatedAt,
		}
	}

	data, err := json.Marshal(contracts)
	if err != nil {
		slog.Error("marshal endpoints", "err", err)
		SendError(w, http.StatusInternalServerError, ErrInternal)
		return
	}

	SendSuccess(w, http.StatusOK, data)
}

func (h *Hook) CreateEndpoint(w http.ResponseWriter, r *http.Request) {
	if h.readOnly(w) {
		return
	}

	contract, err := DecodeRequest[CreateEndpointRequestContract](r)
	if err != nil {
		slog.Error("create endpoint", "err", err)
		SendError(w, http.StatusInternalServerError, ErrInternal)
		return
	}

	hook, err := h.deps.Service.CreateHook(r.Context(), contract.Name)
	if err != nil {
		slog.Error("create endpoint", "err", err)
		SendError(w, http.StatusInternalServerError, ErrInternal)
		return
	}

	data, err := json.Marshal(CreateEndpointResponseContract{
		ID:    hook.ID,
		Token: hook.Token,
		Name:  hook.Name,
		URL:   h.deps.Opts.BaseURL + "/r/" + hook.Token,
	})
	if err != nil {
		slog.Error("marshal endpoint", "err", err)
		SendError(w, http.StatusInternalServerError, ErrInternal)
		return
	}

	SendSuccess(w, http.StatusCreated, data)
}

func (h *Hook) ReceiveWebhook(w http.ResponseWriter, r *http.Request) {
	if h.readOnly(w) {
		return
	}

	token := r.PathValue("token")

	headersJSON, err := json.Marshal(r.Header)
	if err != nil {
		slog.Error("marshal request headers", "err", err)
		SendError(w, http.StatusInternalServerError, ErrInternal)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.deps.Opts.MaxBodySize)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			SendError(w, http.StatusRequestEntityTooLarge, WithDetails(ErrPayloadTooLarge, ErrorDetailContract{
				Field:   "body",
				Message: fmt.Sprintf("body exceeds %d bytes limit", h.deps.Opts.MaxBodySize),
			}))
			return
		}
		slog.Error("read webhook body", "err", err)
		SendError(w, http.StatusBadRequest, WithDetails(ErrBadRequest, ErrorDetailContract{
			Field:   "body",
			Message: "failed to read body",
		}))
		return
	}

	req, customResp, err := h.deps.Service.ReceiveWebhook(r.Context(), token, domain.CreateWebhookRequestParams{
		Method:      r.Method,
		Path:        r.URL.Path,
		Query:       r.URL.RawQuery,
		Headers:     string(headersJSON),
		Body:        body,
		RemoteAddr:  r.RemoteAddr,
		ContentType: r.Header.Get("Content-Type"),
		BodySize:    int64(len(body)),
	})
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, ErrNotFound)
			return
		}
		slog.Error("receive webhook", "err", err)
		SendError(w, http.StatusInternalServerError, ErrInternal)
		return
	}

	data, err := json.Marshal(toWebhookRequestContract(req))
	if err != nil {
		slog.Error("marshal webhook request", "err", err)
		SendError(w, http.StatusInternalServerError, ErrInternal)
		return
	}

	h.deps.Hub.Publish(token, data)
	go func() {
		select {
		case h.notifySem <- struct{}{}:
			defer func() { <-h.notifySem }()
			h.sendNotifications(req, token, context.WithoutCancel(r.Context()))
		default:
			slog.Warn("notification queue full, dropping", "token", token)
		}
	}()

	if customResp.StatusCode > 0 {
		for k, v := range customResp.Headers {
			w.Header().Set(k, v)
		}
		w.WriteHeader(int(customResp.StatusCode))
		if len(customResp.Body) > 0 {
			if _, err := w.Write(customResp.Body); err != nil {
				slog.Error("write custom response body", "err", err)
			}
		}
		return
	}

	SendSuccess(w, http.StatusOK, data)
}

func (h *Hook) ListRequests(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")

	reqs, err := h.deps.Service.ListWebhookRequests(r.Context(), token)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, ErrNotFound)
			return
		}
		slog.Error("list requests", "err", err)
		SendError(w, http.StatusInternalServerError, ErrInternal)
		return
	}

	contracts := make([]WebhookRequestContract, len(reqs))
	for i, req := range reqs {
		contracts[i] = toWebhookRequestContract(req)
	}

	data, err := json.Marshal(contracts)
	if err != nil {
		slog.Error("marshal requests", "err", err)
		SendError(w, http.StatusInternalServerError, ErrInternal)
		return
	}

	SendSuccess(w, http.StatusOK, data)
}

func (h *Hook) StreamEvents(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	ch, unsubscribe := h.deps.Hub.Subscribe(token)
	defer unsubscribe()

	flusher, canFlush := w.(http.Flusher)

	for {
		select {
		case <-r.Context().Done():
			return

		case <-h.deps.Hub.Done():
			return

		case data := <-ch:
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				return
			}
			if canFlush {
				flusher.Flush()
			}
		}
	}
}

func (h *Hook) GetResponse(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")

	resp, err := h.deps.Service.GetHookResponse(r.Context(), token)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, ErrNotFound)
			return
		}
		slog.Error("get hook response", "err", err)
		SendError(w, http.StatusInternalServerError, ErrInternal)
		return
	}

	data, err := json.Marshal(toHookResponseContract(resp))
	if err != nil {
		SendError(w, http.StatusInternalServerError, ErrInternal)
		return
	}

	SendSuccess(w, http.StatusOK, data)
}

func (h *Hook) SetResponse(w http.ResponseWriter, r *http.Request) {
	if h.readOnly(w) {
		return
	}

	token := r.PathValue("token")

	contract, err := DecodeRequest[SetHookResponseRequestContract](r)
	if err != nil {
		SendError(w, http.StatusBadRequest, ErrBadRequest)
		return
	}

	params := domain.UpsertHookResponseParams{
		StatusCode: int64(contract.StatusCode),
		Headers:    contract.Headers,
		Body:       []byte(contract.Body),
	}
	if err := params.Validate(); err != nil {
		SendError(w, http.StatusBadRequest, WithDetails(ErrBadRequest, ErrorDetailContract{
			Field:   "statusCode",
			Message: err.Error(),
		}))
		return
	}

	resp, err := h.deps.Service.SetHookResponse(r.Context(), token, params)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, ErrNotFound)
			return
		}
		slog.Error("set hook response", "err", err)
		SendError(w, http.StatusInternalServerError, ErrInternal)
		return
	}

	data, err := json.Marshal(toHookResponseContract(resp))
	if err != nil {
		SendError(w, http.StatusInternalServerError, ErrInternal)
		return
	}

	SendSuccess(w, http.StatusOK, data)
}

func (h *Hook) GetNotification(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")

	channels, err := h.deps.Service.ListChannels(r.Context(), token)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, ErrNotFound)
			return
		}
		slog.Error("list notification channels", "err", err)
		SendError(w, http.StatusInternalServerError, ErrInternal)
		return
	}

	contracts := make([]NotificationContract, len(channels))
	for i, ch := range channels {
		contracts[i] = toNotificationContract(ch, notify.SecretKeys(ch.Provider))
	}

	data, err := json.Marshal(contracts)
	if err != nil {
		SendError(w, http.StatusInternalServerError, ErrInternal)
		return
	}

	SendSuccess(w, http.StatusOK, data)
}

func (h *Hook) SetNotification(w http.ResponseWriter, r *http.Request) {
	if h.readOnly(w) {
		return
	}

	token := r.PathValue("token")
	provider := r.PathValue("provider")

	contract, err := DecodeRequest[NotificationContract](r)
	if err != nil {
		SendError(w, http.StatusBadRequest, ErrBadRequest)
		return
	}

	if contract.Config == nil {
		contract.Config = make(map[string]string)
	}

	if secrets := notify.SecretKeys(provider); len(secrets) > 0 {
		if existing, err := h.deps.Service.ListChannels(r.Context(), token); err == nil {
			for _, exc := range existing {
				if exc.Provider != provider {
					continue
				}
				for _, key := range secrets {
					if contract.Config[key] == "" && exc.Config[key] != "" {
						contract.Config[key] = exc.Config[key]
					}
				}
				break
			}
		}
	}

	if err := notify.ValidateConfig(provider, notify.Config(contract.Config)); err != nil {
		SendError(w, http.StatusBadRequest, WithDetails(ErrBadRequest, ErrorDetailContract{Message: err.Error()}))
		return
	}

	ch, err := h.deps.Service.UpsertChannel(r.Context(), token, provider, contract.Config)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, ErrNotFound)
			return
		}
		slog.Error("upsert notification channel", "err", err)
		SendError(w, http.StatusInternalServerError, ErrInternal)
		return
	}

	data, err := json.Marshal(toNotificationContract(ch, notify.SecretKeys(ch.Provider)))
	if err != nil {
		SendError(w, http.StatusInternalServerError, ErrInternal)
		return
	}

	SendSuccess(w, http.StatusOK, data)
}

func (h *Hook) DeleteNotification(w http.ResponseWriter, r *http.Request) {
	if h.readOnly(w) {
		return
	}

	token := r.PathValue("token")
	provider := r.PathValue("provider")

	if err := h.deps.Service.DeleteChannel(r.Context(), token, provider); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, ErrNotFound)
			return
		}
		slog.Error("delete notification channel", "err", err)
		SendError(w, http.StatusInternalServerError, ErrInternal)
		return
	}

	SendSuccess(w, http.StatusOK, []byte(`{}`))
}

func (h *Hook) TestNotification(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	provider := r.PathValue("provider")

	channels, err := h.deps.Service.ListChannels(r.Context(), token)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, ErrNotFound)
			return
		}
		slog.Error("list channels for test", "err", err)
		SendError(w, http.StatusInternalServerError, ErrInternal)
		return
	}

	for _, ch := range channels {
		if ch.Provider != provider {
			continue
		}
		msg := fmt.Sprintf("✅ Webhix test notification for endpoint <code>/r/%s</code>", html.EscapeString(token))
		if err := notify.Send(r.Context(), ch.Provider, notify.Config(ch.Config), msg); err != nil {
			slog.Error("test notification", "provider", provider, "err", err)
			SendError(w, http.StatusBadGateway, WithDetails(ErrInternal, ErrorDetailContract{
				Field:   provider,
				Message: err.Error(),
			}))
			return
		}
		SendSuccess(w, http.StatusOK, []byte(`{"sent":true}`))
		return
	}

	SendError(w, http.StatusNotFound, WithDetails(ErrNotFound, ErrorDetailContract{
		Field:   "provider",
		Message: provider + " is not configured for this endpoint",
	}))
}

func (h *Hook) sendNotifications(req domain.WebhookRequest, token string, ctx context.Context) {
	channels, err := h.deps.Service.GetChannelsForHookID(ctx, req.HookID)
	if err != nil || len(channels) == 0 {
		return
	}

	msg := fmt.Sprintf(
		"📨 <b>New webhook</b>\nEndpoint: <code>/r/%s</code>\nMethod: <b>%s</b>\nPath: <code>%s</code>",
		html.EscapeString(token), html.EscapeString(req.Method), html.EscapeString(req.Path),
	)

	for _, ch := range channels {
		if !ch.Enabled {
			continue
		}
		if err := notify.Send(ctx, ch.Provider, notify.Config(ch.Config), msg); err != nil {
			slog.Warn("notification failed", "provider", ch.Provider, "token", token, "err", err)
		}
	}
}

func (h *Hook) readOnly(w http.ResponseWriter) bool {
	if !h.deps.Opts.ReadOnly {
		return false
	}

	SendError(w, http.StatusForbidden, ErrReadOnly)
	return true
}
