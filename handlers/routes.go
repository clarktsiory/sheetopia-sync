package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (h *Handler) registerRoutes() {
	h.router.Use(middleware.Recoverer)
	h.router.Use(middleware.StripSlashes)

	h.router.Get("/api/info", h.handleInfo)
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

// GET /api/info
func (h *Handler) handleInfo(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Server        string    `json:"server"`
		Time          time.Time `json:"time"`
		ServerVersion string    `json:"serverVersion"`
		APIVersion    string    `json:"apiVersion"`
	}
	respond(w, response{
		Server:        "sheetopia-sync",
		Time:          time.Now(),
		ServerVersion: "dev", // TODO
		APIVersion:    "0.0.1",
	}, http.StatusOK)
}
