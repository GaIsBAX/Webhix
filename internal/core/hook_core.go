package core

import (
	"context"
	"errors"

	"github.com/GaIsBAX/Webhix/internal/domain"
)

const defaultHookResponseStatusCode int64 = 200

type TokenGenerator func() string

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
}

func NewHook(repo HookRepository, generateToken TokenGenerator) *Hook {
	if generateToken == nil {
		generateToken = func() string { return "" }
	}

	return &Hook{
		repo:          repo,
		generateToken: generateToken,
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

func defaultHookResponse() domain.HookResponse {
	return domain.HookResponse{
		StatusCode: defaultHookResponseStatusCode,
		Headers:    map[string]string{},
	}
}
