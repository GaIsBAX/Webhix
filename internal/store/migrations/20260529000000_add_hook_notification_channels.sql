-- +goose Up
CREATE TABLE hook_notification_channels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    hook_id INTEGER NOT NULL,
    provider TEXT NOT NULL,
    config TEXT NOT NULL DEFAULT '{}',
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (hook_id) REFERENCES hooks(id) ON DELETE CASCADE,
    UNIQUE(hook_id, provider)
);

-- +goose Down
DROP TABLE IF EXISTS hook_notification_channels;
