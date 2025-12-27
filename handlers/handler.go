package handlers

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/juho05/sheetopia-sync/database"
)

type Handler struct {
	DB      *sql.DB
	Queries *database.Queries

	router *chi.Mux
}

func NewHandler(db *sql.DB, queries *database.Queries) *Handler {
	h := &Handler{
		DB:      db,
		Queries: queries,
		router:  chi.NewMux(),
	}
	h.registerRoutes()
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}
