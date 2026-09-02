-- name: FindExerciseCategory :one
SELECT * FROM exercise_categories WHERE user = ? AND id = ?;

-- name: FindExerciseCategoriesChangedAfter :many
SELECT * FROM exercise_categories WHERE user = ? AND changed > ? ORDER BY id;

-- name: UpsertExerciseCategory :exec
INSERT INTO exercise_categories (id, user, updated_at, name, position, changed) VALUES (?, ?, ?, ?, ?, unixepoch())
ON CONFLICT (user, id) DO UPDATE SET updated_at = excluded.updated_at, name = excluded.name, position = excluded.position, changed = excluded.changed;

-- name: DeleteExerciseCategory :execresult
DELETE FROM exercise_categories WHERE user = ? AND id = ?;

-- name: FindDeletedExerciseCategoryMarker :one
SELECT * FROM deleted_exercise_categories WHERE user = ? AND category_id = ?;

-- name: FindDeletedExerciseCategoriesSince :many
SELECT category_id, deleted_at FROM deleted_exercise_categories WHERE user = ? AND deleted_at > ?;

-- name: CreateDeletedExerciseCategoryMarker :exec
INSERT INTO deleted_exercise_categories (category_id, user, deleted_at) VALUES (?, ?, unixepoch());

-- name: DeleteDeletedExerciseCategoryMarker :exec
DELETE FROM deleted_exercise_categories WHERE user = ? AND category_id = ?;

-- name: FindExercise :one
SELECT * FROM exercises WHERE user = ? AND id = ?;

-- name: FindExercisesChangedAfter :many
SELECT * FROM exercises WHERE user = ? AND changed > ? ORDER BY id;

-- name: UpsertExercise :exec
INSERT INTO exercises (id, user, updated_at, name, category_id, metadata_json, changed) VALUES (?, ?, ?, ?, ?, ?, unixepoch())
ON CONFLICT (user, id) DO UPDATE SET updated_at = excluded.updated_at, name = excluded.name, category_id = excluded.category_id, metadata_json = excluded.metadata_json, changed = excluded.changed;

-- name: DeleteExercise :execresult
DELETE FROM exercises WHERE user = ? AND id = ?;

-- name: FindDeletedExerciseMarker :one
SELECT * FROM deleted_exercises WHERE user = ? AND exercise_id = ?;

-- name: FindDeletedExercisesSince :many
SELECT exercise_id, deleted_at FROM deleted_exercises WHERE user = ? AND deleted_at > ?;

-- name: CreateDeletedExerciseMarker :exec
INSERT INTO deleted_exercises (exercise_id, user, deleted_at) VALUES (?, ?, unixepoch());

-- name: DeleteDeletedExerciseMarker :exec
DELETE FROM deleted_exercises WHERE user = ? AND exercise_id = ?;

-- name: UnassignAllExerciseTags :exec
DELETE FROM exercise_tags WHERE user = ? AND exercise_id = ?;

-- name: AssignExerciseTag :exec
INSERT INTO exercise_tags (user, exercise_id, tag_id) VALUES (?, ?, ?) ON CONFLICT (user, exercise_id, tag_id) DO NOTHING;

-- name: GetExerciseTagIDs :many
SELECT tag_id FROM exercise_tags WHERE user = ? AND exercise_id = ?;

-- name: FindExerciseTagIDsChangedAfter :many
SELECT exercise_tags.exercise_id, exercise_tags.tag_id FROM exercise_tags
JOIN exercises ON exercises.user = exercise_tags.user AND exercises.id = exercise_tags.exercise_id
WHERE exercises.user = ? AND exercises.changed > ?;

-- name: RemoveAllExerciseScores :exec
DELETE FROM exercise_scores WHERE user = ? AND exercise_id = ?;

-- name: AddExerciseScore :exec
INSERT INTO exercise_scores (user, exercise_id, position, score_id) VALUES (?, ?, ?, ?);

-- name: GetExerciseScoreIDs :many
SELECT score_id FROM exercise_scores WHERE user = ? AND exercise_id = ? ORDER BY position;

-- name: FindExerciseScoreIDsChangedAfter :many
SELECT exercise_scores.exercise_id, exercise_scores.score_id FROM exercise_scores
JOIN exercises ON exercises.user = exercise_scores.user AND exercises.id = exercise_scores.exercise_id
WHERE exercises.user = ? AND exercises.changed > ? ORDER BY exercise_scores.position;

-- name: FindPracticeRoutine :one
SELECT * FROM practice_routines WHERE user = ? AND id = ?;

-- name: FindPracticeRoutinesChangedAfter :many
SELECT * FROM practice_routines WHERE user = ? AND changed > ? ORDER BY id;

-- name: UpsertPracticeRoutine :exec
INSERT INTO practice_routines (id, user, updated_at, name, metadata_json, changed) VALUES (?, ?, ?, ?, ?, unixepoch())
ON CONFLICT (user, id) DO UPDATE SET updated_at = excluded.updated_at, name = excluded.name, metadata_json = excluded.metadata_json, changed = excluded.changed;

-- name: DeletePracticeRoutine :execresult
DELETE FROM practice_routines WHERE user = ? AND id = ?;

-- name: FindDeletedPracticeRoutineMarker :one
SELECT * FROM deleted_practice_routines WHERE user = ? AND routine_id = ?;

-- name: FindDeletedPracticeRoutinesSince :many
SELECT routine_id, deleted_at FROM deleted_practice_routines WHERE user = ? AND deleted_at > ?;

-- name: CreateDeletedPracticeRoutineMarker :exec
INSERT INTO deleted_practice_routines (routine_id, user, deleted_at) VALUES (?, ?, unixepoch());

-- name: DeleteDeletedPracticeRoutineMarker :exec
DELETE FROM deleted_practice_routines WHERE user = ? AND routine_id = ?;

-- name: RemoveAllPracticeRoutineEntries :exec
DELETE FROM practice_routine_entries WHERE user = ? AND routine_id = ?;

-- name: FindPracticeRoutineEntry :one
SELECT * FROM practice_routine_entries WHERE user = ? AND id = ?;

-- name: AddPracticeRoutineEntry :exec
INSERT INTO practice_routine_entries (user, id, routine_id, position, exercise_id, metadata_json) VALUES (?, ?, ?, ?, ?, ?);

-- name: GetPracticeRoutineEntries :many
SELECT * FROM practice_routine_entries WHERE user = ? AND routine_id = ? ORDER BY position;

-- name: FindPracticeRoutineEntriesChangedAfter :many
SELECT practice_routine_entries.* FROM practice_routine_entries
JOIN practice_routines ON practice_routines.user = practice_routine_entries.user AND practice_routines.id = practice_routine_entries.routine_id
WHERE practice_routines.user = ? AND practice_routines.changed > ? ORDER BY practice_routine_entries.position;

-- name: FindPracticeSession :one
SELECT * FROM practice_sessions WHERE user = ? AND id = ?;

-- name: FindPracticeSessionsChangedAfter :many
SELECT * FROM practice_sessions WHERE user = ? AND changed > ? ORDER BY id;

-- name: UpsertPracticeSession :exec
INSERT INTO practice_sessions (id, user, updated_at, started_at, ended_at, routine_id, metadata_json, changed) VALUES (?, ?, ?, ?, ?, ?, ?, unixepoch())
ON CONFLICT (user, id) DO UPDATE SET updated_at = excluded.updated_at, started_at = excluded.started_at, ended_at = excluded.ended_at, routine_id = excluded.routine_id, metadata_json = excluded.metadata_json, changed = excluded.changed;

-- name: DeletePracticeSession :execresult
DELETE FROM practice_sessions WHERE user = ? AND id = ?;

-- name: FindDeletedPracticeSessionMarker :one
SELECT * FROM deleted_practice_sessions WHERE user = ? AND session_id = ?;

-- name: FindDeletedPracticeSessionsSince :many
SELECT session_id, deleted_at FROM deleted_practice_sessions WHERE user = ? AND deleted_at > ?;

-- name: CreateDeletedPracticeSessionMarker :exec
INSERT INTO deleted_practice_sessions (session_id, user, deleted_at) VALUES (?, ?, unixepoch());

-- name: DeleteDeletedPracticeSessionMarker :exec
DELETE FROM deleted_practice_sessions WHERE user = ? AND session_id = ?;

-- name: RemoveAllPracticeSessionEntries :exec
DELETE FROM practice_session_entries WHERE user = ? AND session_id = ?;

-- name: FindPracticeSessionEntry :one
SELECT * FROM practice_session_entries WHERE user = ? AND id = ?;

-- name: AddPracticeSessionEntry :exec
INSERT INTO practice_session_entries (user, id, session_id, exercise_id, routine_entry_id, metadata_json) VALUES (?, ?, ?, ?, ?, ?);

-- name: GetPracticeSessionEntries :many
SELECT * FROM practice_session_entries WHERE user = ? AND session_id = ? ORDER BY id;

-- name: FindPracticeSessionEntriesChangedAfter :many
SELECT practice_session_entries.* FROM practice_session_entries
JOIN practice_sessions ON practice_sessions.user = practice_session_entries.user AND practice_sessions.id = practice_session_entries.session_id
WHERE practice_sessions.user = ? AND practice_sessions.changed > ? ORDER BY practice_session_entries.id;
