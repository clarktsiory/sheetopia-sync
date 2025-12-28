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

		r.Get("/api/score", h.handleGetScores)
		r.Get("/api/score/{id}", h.handleGetScore)
		r.Post("/api/score/{id}", h.handleUpdateScore)
		r.Delete("/api/score/{id}", h.handleDeleteScore)

		r.Get("/api/score/{id}/file", h.handleGetScoreFile)
		r.Post("/api/score/{id}/file", h.handleUpdateScoreFile)

		r.Get("/api/tag", h.handleGetTags)
		r.Get("/api/tag/{id}", h.handleGetTag)
		r.Post("/api/tag/{id}", h.handleUpdateTag)
		r.Delete("/api/tag/{id}", h.handleDeleteTag)
	})
}

// GET /api/ping
func (h *Handler) handlePing(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("sheetopia-sync"))
}
