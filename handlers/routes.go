package handlers

import "net/http"

func (h *Handler) registerRoutes() {
	h.router.Get("/api/ping", h.handlePing)
}

func (h *Handler) handlePing(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("sheetopia-sync"))
}
