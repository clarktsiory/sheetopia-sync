-- +migrate Up

CREATE TABLE scores_new (
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    id TEXT NOT NULL,
    metadata_updated_at DATETIME NOT NULL,
    file_updated_at DATETIME NOT NULL DEFAULT 0,
    file_type TEXT NOT NULL DEFAULT 'none',
    title TEXT NOT NULL,
    metadata_json BLOB NOT NULL,

    changed DATETIME NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (user, id)
);

CREATE TABLE tags_new (
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    id TEXT NOT NULL,
    updated_at DATETIME NOT NULL,
    name TEXT NOT NULL,
    color INTEGER NOT NULL,

    changed DATETIME NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (user, id)
);

CREATE TABLE score_tags_new (
    user TEXT NOT NULL,
    score_id TEXT NOT NULL,
    tag_id TEXT NOT NULL,
    PRIMARY KEY (user, score_id, tag_id),
    FOREIGN KEY (user, score_id) REFERENCES scores_new(user, id) ON UPDATE CASCADE ON DELETE CASCADE,
    FOREIGN KEY (user, tag_id) REFERENCES tags_new(user, id) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE TABLE deleted_scores_new (
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    score_id TEXT NOT NULL,
    deleted_at DATETIME NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (user, score_id)
);

CREATE TABLE deleted_tags_new (
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    tag_id TEXT NOT NULL,
    deleted_at DATETIME NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (user, tag_id)
);

CREATE TABLE setlists_new (
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    id TEXT NOT NULL,
    updated_at DATETIME NOT NULL,
    name TEXT NOT NULL,

    changed DATETIME NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (user, id)
);

-- score_id has no foreign key on purpose because a setlist may reference a score that this server hasn't received yet
CREATE TABLE setlist_entries_new (
    user TEXT NOT NULL,
    setlist_id TEXT NOT NULL,
    position INTEGER NOT NULL,
    score_id TEXT NOT NULL,
    PRIMARY KEY (user, setlist_id, position),
    FOREIGN KEY (user, setlist_id) REFERENCES setlists_new(user, id) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE TABLE deleted_setlists_new (
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    setlist_id TEXT NOT NULL,
    deleted_at DATETIME NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (user, setlist_id)
);

INSERT INTO scores_new (user, id, metadata_updated_at, file_updated_at, file_type, title, metadata_json, changed)
SELECT user, id, metadata_updated_at, file_updated_at, file_type, title, metadata_json, changed FROM scores;

INSERT INTO tags_new (user, id, updated_at, name, color, changed)
SELECT user, id, updated_at, name, color, changed FROM tags;

-- The join on tags drops assignments of another user's tag. Those rows are reachable through the
-- API today because the old foreign key only required the tag to exist globally, and they would
-- violate the new (user, tag_id) foreign key.
INSERT INTO score_tags_new (user, score_id, tag_id)
SELECT s.user, st.score_id, st.tag_id FROM score_tags st
JOIN scores s ON s.id = st.score_id
JOIN tags t ON t.id = st.tag_id AND t.user = s.user;

INSERT INTO deleted_scores_new (user, score_id, deleted_at) SELECT user, score_id, deleted_at FROM deleted_scores;
INSERT INTO deleted_tags_new (user, tag_id, deleted_at) SELECT user, tag_id, deleted_at FROM deleted_tags;

INSERT INTO setlists_new (user, id, updated_at, name, changed)
SELECT user, id, updated_at, name, changed FROM setlists;

INSERT INTO setlist_entries_new (user, setlist_id, position, score_id)
SELECT sl.user, e.setlist_id, e.position, e.score_id FROM setlist_entries e
JOIN setlists sl ON sl.id = e.setlist_id;

INSERT INTO deleted_setlists_new (user, setlist_id, deleted_at) SELECT user, setlist_id, deleted_at FROM deleted_setlists;

DROP TABLE score_tags;
DROP TABLE setlist_entries;
DROP TABLE deleted_scores;
DROP TABLE deleted_tags;
DROP TABLE deleted_setlists;
DROP TABLE scores;
DROP TABLE tags;
DROP TABLE setlists;

ALTER TABLE scores_new RENAME TO scores;
ALTER TABLE tags_new RENAME TO tags;
ALTER TABLE score_tags_new RENAME TO score_tags;
ALTER TABLE deleted_scores_new RENAME TO deleted_scores;
ALTER TABLE deleted_tags_new RENAME TO deleted_tags;
ALTER TABLE setlists_new RENAME TO setlists;
ALTER TABLE setlist_entries_new RENAME TO setlist_entries;
ALTER TABLE deleted_setlists_new RENAME TO deleted_setlists;

-- +migrate Down

-- This direction is only lossless while every id is still unique across users. Once two users own
-- the same id it aborts on a primary key conflict, which is inherent to the change. Restore a
-- backup instead of downgrading.

CREATE TABLE scores_old (
    id TEXT PRIMARY KEY,
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    metadata_updated_at DATETIME NOT NULL,
    file_updated_at DATETIME NOT NULL DEFAULT 0,
    file_type TEXT NOT NULL DEFAULT 'none',
    title TEXT NOT NULL,
    metadata_json BLOB NOT NULL,

    changed DATETIME NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE tags_old (
    id TEXT PRIMARY KEY,
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    updated_at DATETIME NOT NULL,
    name TEXT NOT NULL,
    color INTEGER NOT NULL,

    changed DATETIME NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE score_tags_old(
    score_id TEXT NOT NULL REFERENCES scores_old(id) ON UPDATE CASCADE ON DELETE CASCADE,
    tag_id TEXT NOT NULL REFERENCES tags_old(id) ON UPDATE CASCADE ON DELETE CASCADE,
    PRIMARY KEY (score_id, tag_id)
);

CREATE TABLE deleted_scores_old (
    score_id TEXT PRIMARY KEY,
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    deleted_at DATETIME NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE deleted_tags_old (
    tag_id TEXT PRIMARY KEY,
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    deleted_at DATETIME NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE setlists_old (
    id TEXT PRIMARY KEY,
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    updated_at DATETIME NOT NULL,
    name TEXT NOT NULL,

    changed DATETIME NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE setlist_entries_old (
    setlist_id TEXT NOT NULL REFERENCES setlists_old(id) ON UPDATE CASCADE ON DELETE CASCADE,
    position INTEGER NOT NULL,
    score_id TEXT NOT NULL,
    PRIMARY KEY (setlist_id, position)
);

CREATE TABLE deleted_setlists_old (
    setlist_id TEXT PRIMARY KEY,
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    deleted_at DATETIME NOT NULL DEFAULT (unixepoch())
);

INSERT INTO scores_old (id, user, metadata_updated_at, file_updated_at, file_type, title, metadata_json, changed)
SELECT id, user, metadata_updated_at, file_updated_at, file_type, title, metadata_json, changed FROM scores;

INSERT INTO tags_old (id, user, updated_at, name, color, changed)
SELECT id, user, updated_at, name, color, changed FROM tags;

INSERT INTO score_tags_old (score_id, tag_id) SELECT score_id, tag_id FROM score_tags;

INSERT INTO deleted_scores_old (score_id, user, deleted_at) SELECT score_id, user, deleted_at FROM deleted_scores;
INSERT INTO deleted_tags_old (tag_id, user, deleted_at) SELECT tag_id, user, deleted_at FROM deleted_tags;

INSERT INTO setlists_old (id, user, updated_at, name, changed)
SELECT id, user, updated_at, name, changed FROM setlists;

INSERT INTO setlist_entries_old (setlist_id, position, score_id)
SELECT setlist_id, position, score_id FROM setlist_entries;

INSERT INTO deleted_setlists_old (setlist_id, user, deleted_at) SELECT setlist_id, user, deleted_at FROM deleted_setlists;

DROP TABLE score_tags;
DROP TABLE setlist_entries;
DROP TABLE deleted_scores;
DROP TABLE deleted_tags;
DROP TABLE deleted_setlists;
DROP TABLE scores;
DROP TABLE tags;
DROP TABLE setlists;

ALTER TABLE scores_old RENAME TO scores;
ALTER TABLE tags_old RENAME TO tags;
ALTER TABLE score_tags_old RENAME TO score_tags;
ALTER TABLE deleted_scores_old RENAME TO deleted_scores;
ALTER TABLE deleted_tags_old RENAME TO deleted_tags;
ALTER TABLE setlists_old RENAME TO setlists;
ALTER TABLE setlist_entries_old RENAME TO setlist_entries;
ALTER TABLE deleted_setlists_old RENAME TO deleted_setlists;
