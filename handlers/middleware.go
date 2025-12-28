package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/juho05/sheetopia-sync/database"
)

func (h *Handler) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			respondUnauthorized(w)
			return
		}
		authKey := strings.TrimPrefix(authHeader, "Bearer ")
		user, err := h.Queries.VerifyAuthKey(r.Context(), database.HashAuthKey(authKey))
		if err != nil {
			respondUnauthorized(w)
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), "user", user))
		next.ServeHTTP(w, r)
	})
}
