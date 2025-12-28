package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (h *Handler) registerRoutes() {
	h.router.Use(middleware.Recoverer)
	h.router.Use(middleware.StripSlashes)

	h.router.Get("/api/ping", h.handlePing)
	h.router.Post("/api/login", h.handleLogin)

	h.router.Group(func(r chi.Router) {
		r.Use(h.auth)

		r.Post("/api/logout", h.handleLogout)
		r.Get("/api/user", h.handleUser)
	})
}

// GET /api/ping
func (h *Handler) handlePing(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("sheetopia-sync"))
}
