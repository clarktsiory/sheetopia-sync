-- +migrate Up
CREATE TABLE setlists (
    id TEXT PRIMARY KEY,
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    updated_at DATETIME NOT NULL,
    name TEXT NOT NULL,

    changed DATETIME NOT NULL DEFAULT (unixepoch())
);

-- score_id has no foreign key on purpose because a setlist may reference a score that this server hasn't received yet
CREATE TABLE setlist_entries (
    setlist_id TEXT NOT NULL REFERENCES setlists(id) ON UPDATE CASCADE ON DELETE CASCADE,
    position INTEGER NOT NULL,
    score_id TEXT NOT NULL,
    PRIMARY KEY (setlist_id, position)
);

CREATE TABLE deleted_setlists (
    setlist_id TEXT PRIMARY KEY,
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    deleted_at DATETIME NOT NULL DEFAULT (unixepoch())
);

-- +migrate Down
DROP TABLE deleted_setlists;
DROP TABLE setlist_entries;
DROP TABLE setlists;
