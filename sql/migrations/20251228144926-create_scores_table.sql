-- +migrate Up
CREATE TABLE scores (
    id TEXT PRIMARY KEY,
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    metadata_updated_at DATETIME NOT NULL,
    file_updated_at DATETIME NOT NULL DEFAULT 0,
    file_type TEXT NOT NULL DEFAULT 'none',
    title TEXT NOT NULL,
    metadata_json BLOB NOT NULL,

    changed DATETIME NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE tags (
    id TEXT PRIMARY KEY,
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    updated_at DATETIME NOT NULL,
    name TEXT NOT NULL,
    color INTEGER NOT NULL,

    changed DATETIME NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE score_tags(
    score_id TEXT NOT NULL REFERENCES scores(id) ON UPDATE CASCADE ON DELETE CASCADE,
    tag_id TEXT NOT NULL REFERENCES tags(id) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE TABLE deleted_scores (
    score_id TEXT PRIMARY KEY,
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    deleted_at DATETIME NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE deleted_tags (
    tag_id TEXT PRIMARY KEY,
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    deleted_at DATETIME NOT NULL DEFAULT (unixepoch())
);

-- +migrate Down
DROP TABLE deleted_tags;
DROP TABLE deleted_scores;
DROP TABLE score_tags;
DROP TABLE tags;
DROP TABLE scores;