package repos

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/GaIsBAX/Webhix/internal/domain"
	"github.com/GaIsBAX/Webhix/internal/store/sqlc"
)

type Hook struct {
	q *sqlc.Queries
}

func NewHook(db sqlc.DBTX) *Hook {
	return &Hook{
		q: sqlc.New(db),
	}
}

func (r *Hook) CreateHook(ctx context.Context, token, name string) (domain.Hook, error) {
	hook, err := r.q.CreateHook(ctx, sqlc.CreateHookParams{
		Token: token,
		Name:  sql.NullString{String: name, Valid: name != ""},
	})
	if err != nil {
		return domain.Hook{}, err
	}

	return toDomainHook(hook), nil
}

func (r *Hook) GetHookByToken(ctx context.Context, token string) (domain.Hook, error) {
	hook, err := r.q.GetHookByToken(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Hook{}, domain.ErrNotFound
		}
		return domain.Hook{}, err
	}

	return toDomainHook(hook), nil
}

func (r *Hook) CreateWebhookRequest(ctx context.Context, params domain.CreateWebhookRequestParams) (domain.WebhookRequest, error) {

	req, err := r.q.CreateWebhookRequest(ctx, sqlc.CreateWebhookRequestParams{
		HookID:      params.HookID,
		Method:      params.Method,
		Path:        params.Path,
		Query:       params.Query,
		Headers:     params.Headers,
		Body:        params.Body,
		RemoteAddr:  sql.NullString{String: params.RemoteAddr, Valid: params.RemoteAddr != ""},
		ContentType: sql.NullString{String: params.ContentType, Valid: params.ContentType != ""},
		BodySize:    params.BodySize,
	})
	if err != nil {
		return domain.WebhookRequest{}, err
	}

	return toDomainWebhookRequest(req), nil
}

func (r *Hook) ListWebhookRequests(ctx context.Context, hookID int64) ([]domain.WebhookRequest, error) {
	rows, err := r.q.ListWebhookRequestsByHookID(ctx, hookID)
	if err != nil {
		return nil, err
	}

	result := make([]domain.WebhookRequest, len(rows))
	for i, row := range rows {
		result[i] = toDomainWebhookRequest(row)
	}

	return result, nil
}

func (r *Hook) ListHooks(ctx context.Context) ([]domain.Hook, error) {
	rows, err := r.q.ListHooks(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]domain.Hook, len(rows))
	for i, row := range rows {
		result[i] = toDomainHook(row)
	}

	return result, nil
}

func (r *Hook) GetHookResponse(ctx context.Context, hookID int64) (domain.HookResponse, error) {
	row, err := r.q.GetHookResponseByHookID(ctx, hookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.HookResponse{}, domain.ErrNotFound
		}
		return domain.HookResponse{}, err
	}

	return toDomainHookResponse(row), nil
}

func (r *Hook) UpsertHookResponse(ctx context.Context, hookID int64, params domain.UpsertHookResponseParams) (domain.HookResponse, error) {
	headersJSON, err := json.Marshal(params.Headers)
	if err != nil {
		return domain.HookResponse{}, err
	}

	row, err := r.q.UpsertHookResponse(ctx, sqlc.UpsertHookResponseParams{
		HookID:     hookID,
		StatusCode: params.StatusCode,
		Headers:    string(headersJSON),
		Body:       params.Body,
	})
	if err != nil {
		return domain.HookResponse{}, err
	}

	return toDomainHookResponse(row), nil
}

func (r *Hook) ListNotificationChannels(ctx context.Context, hookID int64) ([]domain.NotificationChannel, error) {
	rows, err := r.q.ListNotificationChannels(ctx, hookID)
	if err != nil {
		return nil, err
	}

	result := make([]domain.NotificationChannel, len(rows))
	for i, row := range rows {
		result[i] = toDomainChannel(row)
	}

	return result, nil
}

func (r *Hook) GetNotificationChannel(ctx context.Context, hookID int64, provider string) (domain.NotificationChannel, error) {
	row, err := r.q.GetNotificationChannel(ctx, sqlc.GetNotificationChannelParams{
		HookID:   hookID,
		Provider: provider,
	})

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NotificationChannel{}, domain.ErrNotFound
		}
		return domain.NotificationChannel{}, err
	}

	return toDomainChannel(row), nil
}

func (r *Hook) UpsertNotificationChannel(ctx context.Context, hookID int64, provider string, config map[string]string) (domain.NotificationChannel, error) {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return domain.NotificationChannel{}, err
	}

	row, err := r.q.UpsertNotificationChannel(ctx, sqlc.UpsertNotificationChannelParams{
		HookID:   hookID,
		Provider: provider,
		Config:   string(configJSON),
	})

	if err != nil {
		return domain.NotificationChannel{}, err
	}

	return toDomainChannel(row), nil
}

func (r *Hook) DeleteNotificationChannel(ctx context.Context, hookID int64, provider string) error {
	n, err := r.q.DeleteNotificationChannel(ctx, sqlc.DeleteNotificationChannelParams{
		HookID:   hookID,
		Provider: provider,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func toDomainChannel(row sqlc.HookNotificationChannel) domain.NotificationChannel {
	cfg := map[string]string{}
	if err := json.Unmarshal([]byte(row.Config), &cfg); err != nil {
		slog.Warn("parse notification channel config", "err", err)
	}

	return domain.NotificationChannel{
		ID:        row.ID,
		HookID:    row.HookID,
		Provider:  row.Provider,
		Config:    cfg,
		Enabled:   row.Enabled != 0,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func toDomainHookResponse(row sqlc.HookResponse) domain.HookResponse {
	headers := map[string]string{}
	if err := json.Unmarshal([]byte(row.Headers), &headers); err != nil {
		headers = map[string]string{}
	}

	return domain.HookResponse{
		ID:         row.ID,
		HookID:     row.HookID,
		StatusCode: row.StatusCode,
		Headers:    headers,
		Body:       row.Body,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
}

func toDomainHook(hook sqlc.Hook) domain.Hook {
	return domain.Hook{
		ID:        hook.ID,
		Token:     hook.Token,
		Name:      hook.Name.String,
		CreatedAt: hook.CreatedAt,
		UpdatedAt: hook.UpdatedAt,
	}
}

func toDomainWebhookRequest(req sqlc.WebhookRequest) domain.WebhookRequest {
	return domain.WebhookRequest{
		ID:          req.ID,
		HookID:      req.HookID,
		Method:      req.Method,
		Path:        req.Path,
		Query:       req.Query,
		Headers:     req.Headers,
		Body:        req.Body,
		RemoteAddr:  req.RemoteAddr.String,
		ContentType: req.ContentType.String,
		BodySize:    req.BodySize,
		ReceivedAt:  req.ReceivedAt,
	}
}
