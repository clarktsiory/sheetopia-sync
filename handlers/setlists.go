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

type setlistResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	ScoreIDs  []string  `json:"scoreIds"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// GET /api/setlist?changedAfter=<time>
func (h *Handler) handleGetSetlists(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)

	changedAfter, ok := parseTime(w, r.URL.Query().Get("changedAfter"), time.Unix(0, 0))
	if !ok {
		return
	}

	results, err := h.Queries.FindSetlistsChangedAfterWithScoreIds(r.Context(), database.FindSetlistsChangedAfterWithScoreIdsParams{
		User:    user,
		Changed: changedAfter,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("find setlists changed after time with score ids: %w", err))
		return
	}

	setlists := make([]setlistResponse, 0, len(results))
	for _, result := range results {
		if len(setlists) == 0 || setlists[len(setlists)-1].ID != result.ID {
			setlists = append(setlists, setlistResponse{
				ID:        result.ID,
				Name:      result.Name,
				UpdatedAt: result.UpdatedAt,
				ScoreIDs:  make([]string, 0),
			})
		}
		if result.ScoreID.Valid {
			setlists[len(setlists)-1].ScoreIDs = append(setlists[len(setlists)-1].ScoreIDs, result.ScoreID.String)
		}
	}

	type response struct {
		Setlists []setlistResponse `json:"setlists"`
	}
	respond(w, response{
		Setlists: setlists,
	}, http.StatusOK)
}

// GET /api/setlist/deleted?since=<time>
func (h *Handler) handleGetDeletedSetlists(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	since, ok := parseTime(w, r.URL.Query().Get("since"), time.Unix(0, 0))
	if !ok {
		return
	}

	ids, err := h.Queries.FindDeletedSetlistIDsSince(r.Context(), database.FindDeletedSetlistIDsSinceParams{
		User:      user,
		DeletedAt: since,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("find deleted setlists since time: %w", err))
		return
	}

	type response struct {
		SetlistIDs []string `json:"setlistIds"`
	}
	respond(w, response{
		SetlistIDs: ids,
	}, http.StatusOK)
}

// GET /api/setlist/:id
func (h *Handler) handleGetSetlist(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	id := chi.URLParam(r, "id")

	setlist, err := h.Queries.FindSetlistByUser(r.Context(), database.FindSetlistByUserParams{
		User: user,
		ID:   id,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("find setlist: %w", err))
		return
	}

	scoreIDs, err := h.Queries.GetSetlistScoreIDs(r.Context(), id)
	if err != nil {
		respondErr(w, fmt.Errorf("get setlist score ids: %w", err))
		return
	}

	respond(w, setlistResponse{
		ID:        setlist.ID,
		Name:      setlist.Name,
		ScoreIDs:  scoreIDs,
		UpdatedAt: setlist.UpdatedAt,
	}, http.StatusOK)
}

// POST /api/setlist/:id
func (h *Handler) handleUpdateSetlist(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)

	id := chi.URLParam(r, "id")
	if id == "" {
		respondBadRequest(w)
		return
	}

	type request struct {
		Name      string    `json:"name"`
		ScoreIDs  []string  `json:"scoreIds"`
		UpdatedAt time.Time `json:"updatedAt"`
	}
	params, ok := decodeBody[request](w, r)
	if !ok {
		return
	}

	if params.Name == "" || params.UpdatedAt.IsZero() {
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

	setlist, err := q.FindSetlist(r.Context(), id)
	if err == nil {
		if setlist.User != user {
			respondForbidden(w)
			return
		}
		if !setlist.UpdatedAt.Before(params.UpdatedAt) {
			type response struct {
				UpdatedAt time.Time `json:"updatedAt"`
			}
			respond(w, response{
				UpdatedAt: setlist.UpdatedAt,
			}, http.StatusConflict)
			return
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		respondErr(w, fmt.Errorf("find setlist: %w", err))
		return
	}

	deletedSetlistMarker, err := q.FindDeletedSetlistMarker(r.Context(), id)
	if err == nil {
		type response struct {
			DeletedAt time.Time `json:"deletedAt"`
		}
		respond(w, response{
			DeletedAt: deletedSetlistMarker.DeletedAt,
		}, http.StatusConflict)
		return
	}
	if !errors.Is(err, sql.ErrNoRows) {
		respondErr(w, fmt.Errorf("check if already deleted: %w", err))
		return
	}

	err = q.UpsertSetlist(r.Context(), database.UpsertSetlistParams{
		ID:        id,
		User:      user,
		UpdatedAt: params.UpdatedAt,
		Name:      params.Name,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("upsert setlist: %w", err))
		return
	}

	err = q.RemoveAllSetlistEntries(r.Context(), id)
	if err != nil {
		respondErr(w, fmt.Errorf("remove setlist entries: %w", err))
		return
	}

	for i, scoreID := range params.ScoreIDs {
		err = q.AddSetlistEntry(r.Context(), database.AddSetlistEntryParams{
			SetlistID: id,
			Position:  int64(i),
			ScoreID:   scoreID,
		})
		if err != nil {
			respondErr(w, fmt.Errorf("add setlist entry %d: %w", i, err))
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		respondInternalServerError(w, fmt.Errorf("commit tx: %w", err))
		return
	}
	respondOK(w)
}

// DELETE /api/setlist/:id
func (h *Handler) handleDeleteSetlist(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	id := chi.URLParam(r, "id")

	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		respondInternalServerError(w, fmt.Errorf("begin tx: %w", err))
		return
	}
	defer tx.Rollback()

	q := h.Queries.WithTx(tx)
	result, err := q.DeleteSetlist(r.Context(), database.DeleteSetlistParams{
		User: user,
		ID:   id,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("delete setlist: %w", err))
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

	err = q.CreateDeletedSetlistMarker(r.Context(), database.CreateDeletedSetlistMarkerParams{
		SetlistID: id,
		User:      user,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("create deleted setlist marker: %w", err))
		return
	}

	err = tx.Commit()
	if err != nil {
		respondInternalServerError(w, fmt.Errorf("commit tx: %w", err))
		return
	}

	respondOK(w)
}
