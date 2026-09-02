package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/juho05/sheetopia-sync/database"
)

type exerciseCategoryResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Position  int       `json:"position"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func newExerciseCategoryResponse(category database.ExerciseCategory) exerciseCategoryResponse {
	return exerciseCategoryResponse{
		ID:        category.ID,
		Name:      category.Name,
		Position:  int(category.Position),
		UpdatedAt: category.UpdatedAt,
	}
}

// GET /api/practice/category?changedAfter=<time>
func (h *Handler) handleGetExerciseCategories(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)

	changedAfter, ok := parseTime(w, r.URL.Query().Get("changedAfter"), time.Unix(0, 0))
	if !ok {
		return
	}

	rows, err := h.Queries.FindExerciseCategoriesChangedAfter(r.Context(), database.FindExerciseCategoriesChangedAfterParams{
		User:    user,
		Changed: changedAfter,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("find exercise categories changed after time: %w", err))
		return
	}

	categories := make([]exerciseCategoryResponse, 0, len(rows))
	for _, row := range rows {
		categories = append(categories, newExerciseCategoryResponse(row))
	}

	type response struct {
		Categories []exerciseCategoryResponse `json:"categories"`
	}
	respond(w, response{
		Categories: categories,
	}, http.StatusOK)
}

// GET /api/practice/category/deleted?since=<time>
func (h *Handler) handleGetDeletedExerciseCategories(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	since, ok := parseTime(w, r.URL.Query().Get("since"), time.Unix(0, 0))
	if !ok {
		return
	}

	rows, err := h.Queries.FindDeletedExerciseCategoriesSince(r.Context(), database.FindDeletedExerciseCategoriesSinceParams{
		User:      user,
		DeletedAt: since,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("find deleted exercise categories since time: %w", err))
		return
	}

	deleted := make([]deletedItem, 0, len(rows))
	for _, row := range rows {
		deleted = append(deleted, deletedItem{ID: row.CategoryID, DeletedAt: row.DeletedAt})
	}

	type response struct {
		Deleted []deletedItem `json:"deleted"`
	}
	respond(w, response{
		Deleted: deleted,
	}, http.StatusOK)
}

// GET /api/practice/category/:id
func (h *Handler) handleGetExerciseCategory(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	id := chi.URLParam(r, "id")

	category, err := h.Queries.FindExerciseCategory(r.Context(), database.FindExerciseCategoryParams{
		User: user,
		ID:   id,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("find exercise category: %w", err))
		return
	}

	respond(w, newExerciseCategoryResponse(category), http.StatusOK)
}

// POST /api/practice/category/:id
func (h *Handler) handleUpdateExerciseCategory(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	id := chi.URLParam(r, "id")
	if id == "" {
		respondBadRequest(w)
		return
	}

	type request struct {
		Name      string    `json:"name"`
		Position  int       `json:"position"`
		UpdatedAt time.Time `json:"updatedAt"`
		WrittenAt time.Time `json:"writtenAt"`
	}
	params, ok := decodeBody[request](w, r)
	if !ok {
		return
	}

	if params.Name == "" || params.Position < 0 || params.UpdatedAt.IsZero() {
		respondBadRequest(w)
		return
	}

	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		respondInternalServerError(w, fmt.Errorf("begin tx: %w", err))
		return
	}
	defer tx.Rollback()

	q := h.Queries.WithTx(tx)

	category, err := q.FindExerciseCategory(r.Context(), database.FindExerciseCategoryParams{
		User: user,
		ID:   id,
	})
	if err == nil {
		if staleUpdate(w, "updatedAt", category.UpdatedAt, params.UpdatedAt) {
			return
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		respondErr(w, fmt.Errorf("find exercise category: %w", err))
		return
	}

	ok = resolveTombstone(w, params.WrittenAt, tombstone{
		find: func() (time.Time, error) {
			marker, err := q.FindDeletedExerciseCategoryMarker(r.Context(), database.FindDeletedExerciseCategoryMarkerParams{
				User:       user,
				CategoryID: id,
			})
			return marker.DeletedAt, err
		},
		remove: func() error {
			return q.DeleteDeletedExerciseCategoryMarker(r.Context(), database.DeleteDeletedExerciseCategoryMarkerParams{
				User:       user,
				CategoryID: id,
			})
		},
	})
	if !ok {
		return
	}

	err = q.UpsertExerciseCategory(r.Context(), database.UpsertExerciseCategoryParams{
		ID:        id,
		User:      user,
		UpdatedAt: params.UpdatedAt,
		Name:      params.Name,
		Position:  int64(params.Position),
	})
	if err != nil {
		respondErr(w, fmt.Errorf("upsert exercise category: %w", err))
		return
	}

	err = tx.Commit()
	if err != nil {
		respondInternalServerError(w, fmt.Errorf("commit tx: %w", err))
		return
	}

	respondOK(w)
}

// DELETE /api/practice/category/:id
func (h *Handler) handleDeleteExerciseCategory(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	id := chi.URLParam(r, "id")

	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		respondInternalServerError(w, fmt.Errorf("begin tx: %w", err))
		return
	}
	defer tx.Rollback()

	q := h.Queries.WithTx(tx)
	result, err := q.DeleteExerciseCategory(r.Context(), database.DeleteExerciseCategoryParams{
		User: user,
		ID:   id,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("delete exercise category: %w", err))
		return
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		respondErr(w, fmt.Errorf("affected rows: %w", err))
		return
	}
	if rowsAffected == 0 {
		respondNotFound(w)
		return
	}

	err = q.CreateDeletedExerciseCategoryMarker(r.Context(), database.CreateDeletedExerciseCategoryMarkerParams{
		CategoryID: id,
		User:       user,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("create deleted exercise category marker: %w", err))
		return
	}

	err = tx.Commit()
	if err != nil {
		respondInternalServerError(w, fmt.Errorf("commit tx: %w", err))
		return
	}

	respondOK(w)
}
