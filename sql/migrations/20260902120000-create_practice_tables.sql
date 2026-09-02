-- +migrate Up

ALTER TABLE scores ADD COLUMN type TEXT;
ALTER TABLE tags ADD COLUMN type TEXT;

CREATE TABLE exercise_categories (
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    id TEXT NOT NULL,
    updated_at DATETIME NOT NULL,
    name TEXT NOT NULL,
    position INTEGER NOT NULL,

    changed DATETIME NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (user, id)
);

CREATE TABLE deleted_exercise_categories (
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    category_id TEXT NOT NULL,
    deleted_at DATETIME NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (user, category_id)
);

-- category_id has no foreign key on purpose because an exercise may reference a category that this server hasn't received yet
CREATE TABLE exercises (
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    id TEXT NOT NULL,
    updated_at DATETIME NOT NULL,
    name TEXT NOT NULL,
    category_id TEXT,
    metadata_json BLOB NOT NULL,

    changed DATETIME NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (user, id)
);

CREATE TABLE exercise_tags (
    user TEXT NOT NULL,
    exercise_id TEXT NOT NULL,
    tag_id TEXT NOT NULL,
    PRIMARY KEY (user, exercise_id, tag_id),
    FOREIGN KEY (user, exercise_id) REFERENCES exercises(user, id) ON UPDATE CASCADE ON DELETE CASCADE,
    FOREIGN KEY (user, tag_id) REFERENCES tags(user, id) ON UPDATE CASCADE ON DELETE CASCADE
);

-- score_id has no foreign key on purpose because an exercise may reference a score that this server hasn't received yet
CREATE TABLE exercise_scores (
    user TEXT NOT NULL,
    exercise_id TEXT NOT NULL,
    position INTEGER NOT NULL,
    score_id TEXT NOT NULL,
    PRIMARY KEY (user, exercise_id, position),
    FOREIGN KEY (user, exercise_id) REFERENCES exercises(user, id) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE TABLE deleted_exercises (
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    exercise_id TEXT NOT NULL,
    deleted_at DATETIME NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (user, exercise_id)
);

CREATE TABLE practice_routines (
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    id TEXT NOT NULL,
    updated_at DATETIME NOT NULL,
    name TEXT NOT NULL,
    metadata_json BLOB NOT NULL,

    changed DATETIME NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (user, id)
);

-- exercise_id has no foreign key on purpose because a routine may reference an exercise that this server hasn't received yet
CREATE TABLE practice_routine_entries (
    user TEXT NOT NULL,
    id TEXT NOT NULL,
    routine_id TEXT NOT NULL,
    position INTEGER NOT NULL,
    exercise_id TEXT NOT NULL,
    metadata_json BLOB NOT NULL,
    PRIMARY KEY (user, id),
    FOREIGN KEY (user, routine_id) REFERENCES practice_routines(user, id) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE TABLE deleted_practice_routines (
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    routine_id TEXT NOT NULL,
    deleted_at DATETIME NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (user, routine_id)
);

-- routine_id has no foreign key on purpose because the log has to outlive the routine it was practiced from
CREATE TABLE practice_sessions (
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    id TEXT NOT NULL,
    updated_at DATETIME NOT NULL,
    started_at DATETIME NOT NULL,
    ended_at DATETIME,
    routine_id TEXT,
    metadata_json BLOB NOT NULL,

    changed DATETIME NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (user, id)
);

-- exercise_id and routine_entry_id have no foreign keys on purpose because a practice record must survive what it points at
CREATE TABLE practice_session_entries (
    user TEXT NOT NULL,
    id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    exercise_id TEXT NOT NULL,
    routine_entry_id TEXT,
    metadata_json BLOB NOT NULL,
    PRIMARY KEY (user, id),
    FOREIGN KEY (user, session_id) REFERENCES practice_sessions(user, id) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE TABLE deleted_practice_sessions (
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    session_id TEXT NOT NULL,
    deleted_at DATETIME NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (user, session_id)
);

CREATE INDEX exercise_tags_tag_index ON exercise_tags (user, tag_id);
CREATE INDEX exercise_scores_score_index ON exercise_scores (user, score_id);
CREATE INDEX practice_routine_entries_routine_index ON practice_routine_entries (user, routine_id);
CREATE INDEX practice_session_entries_session_index ON practice_session_entries (user, session_id);

-- +migrate Down

DROP TABLE deleted_practice_sessions;
DROP TABLE practice_session_entries;
DROP TABLE practice_sessions;
DROP TABLE deleted_practice_routines;
DROP TABLE practice_routine_entries;
DROP TABLE practice_routines;
DROP TABLE deleted_exercises;
DROP TABLE exercise_scores;
DROP TABLE exercise_tags;
DROP TABLE exercises;
DROP TABLE deleted_exercise_categories;
DROP TABLE exercise_categories;

ALTER TABLE tags DROP COLUMN type;
ALTER TABLE scores DROP COLUMN type;
