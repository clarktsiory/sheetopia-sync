-- +migrate Up
CREATE TABLE users (
    name TEXT PRIMARY KEY,
    password_hash TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE auth_keys (
    key_hash TEXT PRIMARY KEY,
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT (unixepoch()),
    last_used DATETIME NOT NULL DEFAULT (unixepoch())
);

-- +migrate Down
DROP TABLE auth_keys;
DROP TABLE users;