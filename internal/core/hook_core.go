package core

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"sync"

	"github.com/GaIsBAX/Webhix/internal/domain"
)

const (
	defaultHookResponseStatusCode int64 = 200
	maxConcurrentNotifications    int   = 64
)

type TokenGenerator func() string

type NotificationSender interface {
	Send(ctx context.Context, provider string, config map[string]string, message string) error
}

type HookRepository interface {
	CreateHook(ctx context.Context, token, name string) (domain.Hook, error)
	GetHookByToken(ctx context.Context, token string) (domain.Hook, error)
	ListHooks(ctx context.Context) ([]domain.Hook, error)
	CreateWebhookRequest(ctx context.Context, params domain.CreateWebhookRequestParams) (domain.WebhookRequest, error)
	ListWebhookRequests(ctx context.Context, hookID int64) ([]domain.WebhookRequest, error)
	GetHookResponse(ctx context.Context, hookID int64) (domain.HookResponse, error)
	UpsertHookResponse(ctx context.Context, hookID int64, params domain.UpsertHookResponseParams) (domain.HookResponse, error)
	ListNotificationChannels(ctx context.Context, hookID int64) ([]domain.NotificationChannel, error)
	UpsertNotificationChannel(ctx context.Context, hookID int64, provider string, config map[string]string) (domain.NotificationChannel, error)
	DeleteNotificationChannel(ctx context.Context, hookID int64, provider string) error
}

type Hook struct {
	repo          HookRepository
	generateToken TokenGenerator
	sender        NotificationSender
	notifySem     chan struct{}
}

func NewHook(repo HookRepository, generateToken TokenGenerator, sender NotificationSender) *Hook {
	if generateToken == nil {
		generateToken = func() string { return "" }
	}

	return &Hook{
		repo:          repo,
		generateToken: generateToken,
		sender:        sender,
		notifySem:     make(chan struct{}, maxConcurrentNotifications),
	}
}

func (s *Hook) ListHooks(ctx context.Context) ([]domain.Hook, error) {
	return s.repo.ListHooks(ctx)
}

func (s *Hook) CreateHook(ctx context.Context, name string) (domain.Hook, error) {
	return s.repo.CreateHook(ctx, s.generateToken(), name)
}

func (s *Hook) ReceiveWebhook(ctx context.Context, token string, params domain.CreateWebhookRequestParams) (domain.WebhookRequest, domain.HookResponse, error) {
	hook, err := s.repo.GetHookByToken(ctx, token)
	if err != nil {
		return domain.WebhookRequest{}, domain.HookResponse{}, err
	}

	params.HookID = hook.ID

	req, err := s.repo.CreateWebhookRequest(ctx, params)
	if err != nil {
		return domain.WebhookRequest{}, domain.HookResponse{}, err
	}

	resp, err := s.repo.GetHookResponse(ctx, hook.ID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return req, defaultHookResponse(), nil
		}
		return domain.WebhookRequest{}, domain.HookResponse{}, err
	}

	return req, resp, nil
}

func (s *Hook) ListWebhookRequests(ctx context.Context, token string) ([]domain.WebhookRequest, error) {
	hook, err := s.repo.GetHookByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	return s.repo.ListWebhookRequests(ctx, hook.ID)
}

func (s *Hook) GetHookResponse(ctx context.Context, token string) (domain.HookResponse, error) {
	hook, err := s.repo.GetHookByToken(ctx, token)
	if err != nil {
		return domain.HookResponse{}, err
	}

	resp, err := s.repo.GetHookResponse(ctx, hook.ID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return defaultHookResponse(), nil
		}
		return domain.HookResponse{}, err
	}

	return resp, nil
}

func (s *Hook) SetHookResponse(ctx context.Context, token string, params domain.UpsertHookResponseParams) (domain.HookResponse, error) {
	hook, err := s.repo.GetHookByToken(ctx, token)
	if err != nil {
		return domain.HookResponse{}, err
	}

	return s.repo.UpsertHookResponse(ctx, hook.ID, params)
}

func (s *Hook) ListChannels(ctx context.Context, token string) ([]domain.NotificationChannel, error) {
	hook, err := s.repo.GetHookByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	return s.repo.ListNotificationChannels(ctx, hook.ID)
}

func (s *Hook) UpsertChannel(ctx context.Context, token, provider string, config map[string]string) (domain.NotificationChannel, error) {
	hook, err := s.repo.GetHookByToken(ctx, token)
	if err != nil {
		return domain.NotificationChannel{}, err
	}
	return s.repo.UpsertNotificationChannel(ctx, hook.ID, provider, config)
}

func (s *Hook) DeleteChannel(ctx context.Context, token, provider string) error {
	hook, err := s.repo.GetHookByToken(ctx, token)
	if err != nil {
		return err
	}
	return s.repo.DeleteNotificationChannel(ctx, hook.ID, provider)
}

func (s *Hook) GetChannelsForHookID(ctx context.Context, hookID int64) ([]domain.NotificationChannel, error) {
	return s.repo.ListNotificationChannels(ctx, hookID)
}

func (s *Hook) DispatchNotifications(ctx context.Context, req domain.WebhookRequest, token string) {
	ctx = context.WithoutCancel(ctx)
	go func() {
		select {
		case s.notifySem <- struct{}{}:
			defer func() { <-s.notifySem }()
			s.sendNotifications(ctx, req, token)
		default:
			slog.Warn("notification queue full, dropping", "token", token)
		}
	}()
}

func (s *Hook) sendNotifications(ctx context.Context, req domain.WebhookRequest, token string) {
	channels, err := s.repo.ListNotificationChannels(ctx, req.HookID)
	if err != nil {
		slog.Warn("fetch notification channels", "hookID", req.HookID, "err", err)
		return
	}
	if len(channels) == 0 {
		return
	}

	msg := fmt.Sprintf(
		"📨 <b>New webhook</b>\nEndpoint: <code>/r/%s</code>\nMethod: <b>%s</b>\nPath: <code>%s</code>",
		html.EscapeString(token), html.EscapeString(req.Method), html.EscapeString(req.Path),
	)

	var wg sync.WaitGroup
	for _, ch := range channels {
		if !ch.Enabled {
			continue
		}
		wg.Add(1)
		go func(ch domain.NotificationChannel) {
			defer wg.Done()
			if err := s.sender.Send(ctx, ch.Provider, ch.Config, msg); err != nil {
				slog.Warn("notification failed", "provider", ch.Provider, "token", token, "err", err)
			}
		}(ch)
	}
	wg.Wait()
}

func defaultHookResponse() domain.HookResponse {
	return domain.HookResponse{
		StatusCode: defaultHookResponseStatusCode,
		Headers:    map[string]string{},
	}
}
