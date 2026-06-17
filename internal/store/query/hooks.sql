-- name: CreateHook :one
INSERT INTO hooks (token, name)
VALUES (?, ?)
RETURNING id, token, name, created_at, updated_at;

-- name: GetHookByToken :one
SELECT id, token, name, created_at, updated_at
FROM hooks
WHERE token = ?;

-- name: GetCountRequests :one
SELECT COUNT(*)
FROM webhook_requests;

-- name: CreateWebhookRequest :one
INSERT INTO webhook_requests (
    hook_id,
    method,
    path,
    query,
    headers,
    body,
    remote_addr,
    content_type,
    body_size
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, hook_id, method, path, query, headers, body, remote_addr, content_type, body_size, received_at;

-- name: ListWebhookRequestsByHookID :many
SELECT id, hook_id, method, path, query, headers, body, remote_addr, content_type, body_size, received_at
FROM webhook_requests
WHERE hook_id = ?
ORDER BY received_at DESC, id DESC;

-- name: ListWebhookRequestsByTime :many
SELECT
    wr.id,
    wr.hook_id,
    h.token,
    h.name,
    wr.method,
    wr.path,
    wr.query,
    wr.headers,
    wr.remote_addr,
    wr.content_type,
    wr.body_size,
    wr.received_at
FROM webhook_requests wr
JOIN hooks h ON h.id = wr.hook_id
WHERE wr.received_at <= datetime('now', ?)
ORDER BY wr.received_at DESC;

-- name: DeleteWebhookRequestsOlderThan :execresult
DELETE FROM webhook_requests
WHERE received_at < datetime('now', ?);

-- name: UpsertHookResponse :one
INSERT INTO hook_responses (hook_id, status_code, headers, body)
VALUES (?, ?, ?, ?)
ON CONFLICT (hook_id) DO UPDATE SET
    status_code = excluded.status_code,
    headers = excluded.headers,
    body = excluded.body,
    updated_at = CURRENT_TIMESTAMP
RETURNING id, hook_id, status_code, headers, body, created_at, updated_at;

-- name: ListHooks :many
SELECT id, token, name, created_at, updated_at
FROM hooks
ORDER BY created_at DESC;

-- name: GetHookResponseByHookID :one
SELECT id, hook_id, status_code, headers, body, created_at, updated_at
FROM hook_responses
WHERE hook_id = ?;

-- name: ListNotificationChannels :many
SELECT id, hook_id, provider, config, enabled, created_at, updated_at
FROM hook_notification_channels
WHERE hook_id = ?
ORDER BY provider;

-- name: GetNotificationChannel :one
SELECT id, hook_id, provider, config, enabled, created_at, updated_at
FROM hook_notification_channels
WHERE hook_id = ? AND provider = ?;

-- name: UpsertNotificationChannel :one
INSERT INTO hook_notification_channels (hook_id, provider, config)
VALUES (?, ?, ?)
ON CONFLICT (hook_id, provider) DO UPDATE SET
    config     = excluded.config,
    enabled    = 1,
    updated_at = CURRENT_TIMESTAMP
RETURNING id, hook_id, provider, config, enabled, created_at, updated_at;

-- name: DeleteNotificationChannel :execrows
DELETE FROM hook_notification_channels
WHERE hook_id = ? AND provider = ?;
