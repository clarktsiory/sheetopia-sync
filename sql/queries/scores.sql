-- name: FindScore :one
SELECT * FROM scores WHERE id = ?;

-- name: FindScoreByUser :one
SELECT * FROM scores WHERE user = ? AND id = ?;

-- name: FindScoresChangedAfterWithTagIds :many
SELECT * FROM scores LEFT JOIN score_tags ON scores.id = score_tags.score_id
WHERE user = ? AND changed > ? AND file_type != 'none' ORDER BY scores.id;

-- name: UpsertScore :exec
INSERT INTO scores (id, user, metadata_updated_at, file_updated_at, file_type, title, metadata_json, changed)
VALUES (?,?,?,0,'none',?,?,unixepoch())
ON CONFLICT (id) DO UPDATE SET metadata_updated_at = excluded.metadata_updated_at, title = excluded.title, metadata_json = excluded.metadata_json, changed = excluded.changed;

-- name: DeleteScore :execresult
DELETE FROM scores WHERE user = ? AND id = ?;

-- name: CreateDeletedScoreMarker :exec
INSERT INTO deleted_scores (score_id, user, deleted_at) VALUES (?, ?, unixepoch());

-- name: FindDeletedScoreMarker :one
SELECT * FROM deleted_scores WHERE score_id = ?;

-- name: FindDeletedScoreIDsSince :many
SELECT score_id FROM deleted_scores WHERE user = ? AND deleted_at > ?;

-- name: UnassignAllTags :exec
DELETE FROM score_tags WHERE score_id = ?;

-- name: AssignTag :exec
INSERT INTO score_tags (score_id, tag_id) VALUES (?, ?) ON CONFLICT (score_id,tag_id) DO NOTHING;

-- name: FindTag :one
SELECT * FROM tags WHERE id = ?;

-- name: FindTagByUser :one
SELECT * FROM tags WHERE user = ? AND id = ?;

-- name: FindTagsChangedAfter :many
SELECT * FROM tags WHERE user = ? AND changed > ?;

-- name: UpsertTag :exec
INSERT INTO tags (id, user, updated_at, name, color, changed) VALUES (?, ?, ?, ?, ?, unixepoch())
ON CONFLICT (id) DO UPDATE SET updated_at = excluded.updated_at, name = excluded.name, color = excluded.color, changed = excluded.changed;

-- name: DeleteTag :execresult
DELETE FROM tags WHERE user = ? AND id = ?;

-- name: FindDeletedTagMarker :one
SELECT * FROM deleted_tags WHERE tag_id = ?;

-- name: FindDeletedTagIDsSince :many
SELECT tag_id FROM deleted_tags WHERE user = ? AND deleted_at > ?;

-- name: CreateDeletedTagMarker :exec
INSERT INTO deleted_tags (tag_id, user, deleted_at) VALUES (?, ?, unixepoch());

-- name: GetAssignedTagIDs :many
SELECT tag_id FROM score_tags WHERE score_id = ?;

-- name: UpdateFileInfo :execresult
UPDATE scores SET file_updated_at = ?, file_type = ?, changed = unixepoch() WHERE id = ? AND file_updated_at < ?;