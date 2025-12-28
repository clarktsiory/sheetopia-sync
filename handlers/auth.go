package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/juho05/sheetopia-sync/database"
)

// POST /api/login
func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	type params struct {
		Username string `json:"user"`
		Password string `json:"password"`
	}
	body, ok := decodeBody[params](w, r)
	if !ok {
		return
	}

	user, err := h.Queries.FindUser(r.Context(), body.Username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondUnauthorized(w)
		} else {
			respondInternalServerError(w, err)
		}
		return
	}

	valid, err := database.VerifyPassword(user.PasswordHash, body.Password)
	if err != nil {
		respondInternalServerError(w, err)
		return
	}
	if !valid {
		respondUnauthorized(w)
		return
	}

	authKey, err := database.GenerateAuthKey()
	if err != nil {
		respondInternalServerError(w, err)
		return
	}

	err = h.Queries.CreateAuthKey(r.Context(), database.CreateAuthKeyParams{
		KeyHash: database.HashAuthKey(authKey),
		User:    user.Name,
	})
	if err != nil {
		respondInternalServerError(w, err)
		return
	}

	type response struct {
		AuthKey string `json:"auth_key"`
	}
	respond(w, response{
		AuthKey: authKey,
	}, http.StatusOK)
}

// POST /api/logout
func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	authKey := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	err := h.Queries.DeleteAuthKey(r.Context(), database.HashAuthKey(authKey))
	if err != nil {
		respondInternalServerError(w, err)
		return
	}
	respondOK(w)
}

// GET /api/user
func (h *Handler) handleUser(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	type response struct {
		User string `json:"user"`
	}
	respond(w, response{
		User: user,
	}, http.StatusOK)
}
