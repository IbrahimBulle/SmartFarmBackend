-- +goose Up
CREATE TABLE farms (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,

    name TEXT NOT NULL,
    location TEXT NOT NULL,
    crop_type TEXT NOT NULL,
    size_acres REAL,

    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_farms_user_id ON farms(user_id);

-- +goose Down
DROP TABLE farms;