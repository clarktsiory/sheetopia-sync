-- name: CreateUser :exec
INSERT INTO users (name, password_hash) VALUES (?, ?);

-- name: FindUser :one
SELECT * FROM users WHERE name = ?;

-- name: FindUsers :many
SELECT * FROM users;

-- name: DeleteUser :execresult
DELETE FROM users WHERE name = ?;

-- name: UpdateUserName :execresult
UPDATE users SET name = ? WHERE name = ?;

-- name: UpdateUserPassword :execresult
UPDATE users SET password_hash = ? WHERE name = ?;

-- name: CreateAuthKey :exec
INSERT INTO auth_keys (key, user) VALUES (?, ?);

-- name: FindAuthKey :one
SELECT * FROM auth_keys WHERE user = ? AND key = ?;

-- name: DeleteAuthKey :exec
DELETE FROM auth_keys WHERE auth_keys.user = ? AND auth_keys.key = ?;