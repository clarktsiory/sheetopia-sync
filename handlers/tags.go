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

type tagResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Color     int       `json:"color"`
	UpdatedAt time.Time `json:"updatedAt"`

	Type *string `json:"type,omitempty"`
}

// GET /api/tag?changedAfter=<time>
func (h *Handler) handleGetTags(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)

	changedAfter, ok := parseTime(w, r.URL.Query().Get("changedAfter"), time.Unix(0, 0))
	if !ok {
		return
	}

	tags, err := h.Queries.FindTagsChangedAfter(r.Context(), database.FindTagsChangedAfterParams{
		User:    user,
		Changed: changedAfter,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("find tags changed after time: %w", err))
		return
	}

	responseTags := make([]tagResponse, 0, len(tags))
	for _, tag := range tags {
		responseTags = append(responseTags, tagResponse{
			ID:        tag.ID,
			Name:      tag.Name,
			Color:     int(tag.Color),
			Type:      nullString(tag.Type),
			UpdatedAt: tag.UpdatedAt,
		})
	}

	type response struct {
		Tags []tagResponse `json:"tags"`
	}
	respond(w, response{
		Tags: responseTags,
	}, http.StatusOK)
}

// GET /api/tag/deleted?since=<time>
func (h *Handler) handleGetDeletedTags(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	since, ok := parseTime(w, r.URL.Query().Get("since"), time.Unix(0, 0))
	if !ok {
		return
	}

	rows, err := h.Queries.FindDeletedTagsSince(r.Context(), database.FindDeletedTagsSinceParams{
		User:      user,
		DeletedAt: since,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("find deleted tags since time: %w", err))
		return
	}

	ids := make([]string, 0, len(rows))
	deleted := make([]deletedItem, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.TagID)
		deleted = append(deleted, deletedItem{ID: row.TagID, DeletedAt: row.DeletedAt})
	}

	type response struct {
		TagIDs      []string      `json:"tagIds"`
		DeletedTags []deletedItem `json:"deletedTags"`
	}
	respond(w, response{
		TagIDs:      ids,
		DeletedTags: deleted,
	}, http.StatusOK)
}

// GET /api/tag/:id
func (h *Handler) handleGetTag(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	id := chi.URLParam(r, "id")

	tag, err := h.Queries.FindTag(r.Context(), database.FindTagParams{
		User: user,
		ID:   id,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("find tag: %w", err))
		return
	}

	respond(w, tagResponse{
		ID:        tag.ID,
		Name:      tag.Name,
		Color:     int(tag.Color),
		Type:      nullString(tag.Type),
		UpdatedAt: tag.UpdatedAt,
	}, http.StatusOK)
}

// POST /api/tag/:id
func (h *Handler) handleUpdateTag(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	id := chi.URLParam(r, "id")

	type request struct {
		Name      string    `json:"name"`
		Color     int       `json:"color"`
		Type      string    `json:"type"`
		UpdatedAt time.Time `json:"updatedAt"`
		WrittenAt time.Time `json:"writtenAt"`
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

	// a client that does not know about types must not reset the type of an existing tag
	var tagType sql.NullString

	tag, err := q.FindTag(r.Context(), database.FindTagParams{
		User: user,
		ID:   id,
	})
	if err == nil {
		tagType = tag.Type
		if staleUpdate(w, "updatedAt", tag.UpdatedAt, params.UpdatedAt) {
			return
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		respondErr(w, fmt.Errorf("find tag: %w", err))
		return
	}

	ok = resolveTombstone(w, params.WrittenAt, tombstone{
		find: func() (time.Time, error) {
			marker, err := q.FindDeletedTagMarker(r.Context(), database.FindDeletedTagMarkerParams{
				User:  user,
				TagID: id,
			})
			return marker.DeletedAt, err
		},
		remove: func() error {
			return q.DeleteDeletedTagMarker(r.Context(), database.DeleteDeletedTagMarkerParams{
				User:  user,
				TagID: id,
			})
		},
	})
	if !ok {
		return
	}

	if params.Type != "" {
		tagType = sql.NullString{String: params.Type, Valid: true}
	}

	err = q.UpsertTag(r.Context(), database.UpsertTagParams{
		ID:        id,
		User:      user,
		UpdatedAt: params.UpdatedAt,
		Name:      params.Name,
		Color:     int64(params.Color),
		Type:      tagType,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("upsert tag: %w", err))
		return
	}

	err = tx.Commit()
	if err != nil {
		respondErr(w, fmt.Errorf("commit tx: %w", err))
		return
	}

	respondOK(w)
}

// DELETE /api/tag/:id
func (h *Handler) handleDeleteTag(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	id := chi.URLParam(r, "id")

	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		respondErr(w, fmt.Errorf("begin tx: %w", err))
		return
	}
	defer tx.Rollback()

	q := h.Queries.WithTx(tx)
	result, err := q.DeleteTag(r.Context(), database.DeleteTagParams{
		ID:   id,
		User: user,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("delete tag: %w", err))
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

	err = q.CreateDeletedTagMarker(r.Context(), database.CreateDeletedTagMarkerParams{
		TagID: id,
		User:  user,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("create deleted tag marker: %w", err))
		return
	}

	err = tx.Commit()
	if err != nil {
		respondErr(w, fmt.Errorf("commit: %w", err))
		return
	}

	respondOK(w)
}
