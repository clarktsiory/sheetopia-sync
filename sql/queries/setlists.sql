-- name: FindSetlist :one
SELECT * FROM setlists WHERE id = ?;

-- name: FindSetlistByUser :one
SELECT * FROM setlists WHERE user = ? AND id = ?;

-- name: FindSetlistsChangedAfterWithScoreIds :many
SELECT * FROM setlists LEFT JOIN setlist_entries ON setlists.id = setlist_entries.setlist_id
WHERE user = ? AND changed > ? ORDER BY setlists.id, setlist_entries.position;

-- name: UpsertSetlist :exec
INSERT INTO setlists (id, user, updated_at, name, changed) VALUES (?, ?, ?, ?, unixepoch())
ON CONFLICT (id) DO UPDATE SET updated_at = excluded.updated_at, name = excluded.name, changed = excluded.changed;

-- name: DeleteSetlist :execresult
DELETE FROM setlists WHERE user = ? AND id = ?;

-- name: FindDeletedSetlistMarker :one
SELECT * FROM deleted_setlists WHERE setlist_id = ?;

-- name: FindDeletedSetlistIDsSince :many
SELECT setlist_id FROM deleted_setlists WHERE user = ? AND deleted_at > ?;

-- name: CreateDeletedSetlistMarker :exec
INSERT INTO deleted_setlists (setlist_id, user, deleted_at) VALUES (?, ?, unixepoch());

-- name: RemoveAllSetlistEntries :exec
DELETE FROM setlist_entries WHERE setlist_id = ?;

-- name: AddSetlistEntry :exec
INSERT INTO setlist_entries (setlist_id, position, score_id) VALUES (?, ?, ?);

-- name: GetSetlistScoreIDs :many
SELECT score_id FROM setlist_entries WHERE setlist_id = ? ORDER BY position;
