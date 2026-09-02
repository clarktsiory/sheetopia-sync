package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/juho05/sheetopia-sync/database"
)

type practiceSessionEntry struct {
	ID             string          `json:"id"`
	ExerciseID     string          `json:"exerciseId"`
	RoutineEntryID *string         `json:"routineEntryId"`
	Metadata       json.RawMessage `json:"metadata"`
}

type practiceSessionResponse struct {
	ID        string                 `json:"id"`
	StartedAt time.Time              `json:"startedAt"`
	EndedAt   *time.Time             `json:"endedAt"`
	RoutineID *string                `json:"routineId"`
	Metadata  json.RawMessage        `json:"metadata"`
	Entries   []practiceSessionEntry `json:"entries"`
	UpdatedAt time.Time              `json:"updatedAt"`
}

func newPracticeSessionResponse(session database.PracticeSession, entries []practiceSessionEntry) practiceSessionResponse {
	if entries == nil {
		entries = make([]practiceSessionEntry, 0)
	}
	return practiceSessionResponse{
		ID:        session.ID,
		StartedAt: session.StartedAt,
		EndedAt:   nullTime(session.EndedAt),
		RoutineID: nullString(session.RoutineID),
		Metadata:  session.MetadataJson,
		Entries:   entries,
		UpdatedAt: session.UpdatedAt,
	}
}

func newPracticeSessionEntry(entry database.PracticeSessionEntry) practiceSessionEntry {
	return practiceSessionEntry{
		ID:             entry.ID,
		ExerciseID:     entry.ExerciseID,
		RoutineEntryID: nullString(entry.RoutineEntryID),
		Metadata:       entry.MetadataJson,
	}
}

// GET /api/practice/session?changedAfter=<time>
func (h *Handler) handleGetPracticeSessions(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)

	changedAfter, ok := parseTime(w, r.URL.Query().Get("changedAfter"), time.Unix(0, 0))
	if !ok {
		return
	}

	rows, err := h.Queries.FindPracticeSessionsChangedAfter(r.Context(), database.FindPracticeSessionsChangedAfterParams{
		User:    user,
		Changed: changedAfter,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("find practice sessions changed after time: %w", err))
		return
	}

	entryRows, err := h.Queries.FindPracticeSessionEntriesChangedAfter(r.Context(), database.FindPracticeSessionEntriesChangedAfterParams{
		User:    user,
		Changed: changedAfter,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("find entries of practice sessions changed after time: %w", err))
		return
	}
	entries := make(map[string][]practiceSessionEntry, len(rows))
	for _, row := range entryRows {
		entries[row.SessionID] = append(entries[row.SessionID], newPracticeSessionEntry(row))
	}

	sessions := make([]practiceSessionResponse, 0, len(rows))
	for _, row := range rows {
		sessions = append(sessions, newPracticeSessionResponse(row, entries[row.ID]))
	}

	type response struct {
		Sessions []practiceSessionResponse `json:"sessions"`
	}
	respond(w, response{
		Sessions: sessions,
	}, http.StatusOK)
}

// GET /api/practice/session/deleted?since=<time>
func (h *Handler) handleGetDeletedPracticeSessions(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	since, ok := parseTime(w, r.URL.Query().Get("since"), time.Unix(0, 0))
	if !ok {
		return
	}

	rows, err := h.Queries.FindDeletedPracticeSessionsSince(r.Context(), database.FindDeletedPracticeSessionsSinceParams{
		User:      user,
		DeletedAt: since,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("find deleted practice sessions since time: %w", err))
		return
	}

	deleted := make([]deletedItem, 0, len(rows))
	for _, row := range rows {
		deleted = append(deleted, deletedItem{ID: row.SessionID, DeletedAt: row.DeletedAt})
	}

	type response struct {
		Deleted []deletedItem `json:"deleted"`
	}
	respond(w, response{
		Deleted: deleted,
	}, http.StatusOK)
}

// GET /api/practice/session/:id
func (h *Handler) handleGetPracticeSession(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	id := chi.URLParam(r, "id")

	session, err := h.Queries.FindPracticeSession(r.Context(), database.FindPracticeSessionParams{
		User: user,
		ID:   id,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("find practice session: %w", err))
		return
	}

	rows, err := h.Queries.GetPracticeSessionEntries(r.Context(), database.GetPracticeSessionEntriesParams{
		User:      user,
		SessionID: id,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("get practice session entries: %w", err))
		return
	}

	entries := make([]practiceSessionEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, newPracticeSessionEntry(row))
	}

	respond(w, newPracticeSessionResponse(session, entries), http.StatusOK)
}

// POST /api/practice/session/:id
func (h *Handler) handleUpdatePracticeSession(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	id := chi.URLParam(r, "id")
	if id == "" {
		respondBadRequest(w)
		return
	}

	type request struct {
		StartedAt time.Time              `json:"startedAt"`
		EndedAt   *time.Time             `json:"endedAt"`
		RoutineID *string                `json:"routineId"`
		Metadata  json.RawMessage        `json:"metadata"`
		Entries   []practiceSessionEntry `json:"entries"`
		UpdatedAt time.Time              `json:"updatedAt"`
		WrittenAt time.Time              `json:"writtenAt"`
	}
	params, ok := decodeBody[request](w, r)
	if !ok {
		return
	}

	if params.StartedAt.IsZero() || params.UpdatedAt.IsZero() || params.Metadata == nil {
		respondBadRequest(w)
		return
	}
	entryIDs := make(map[string]struct{}, len(params.Entries))
	for _, entry := range params.Entries {
		if entry.ID == "" || entry.ExerciseID == "" || entry.Metadata == nil {
			respondBadRequest(w)
			return
		}
		if _, duplicate := entryIDs[entry.ID]; duplicate {
			respondBadRequest(w)
			return
		}
		entryIDs[entry.ID] = struct{}{}
	}

	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		respondInternalServerError(w, fmt.Errorf("begin tx: %w", err))
		return
	}
	defer tx.Rollback()

	q := h.Queries.WithTx(tx)

	session, err := q.FindPracticeSession(r.Context(), database.FindPracticeSessionParams{
		User: user,
		ID:   id,
	})
	if err == nil {
		if staleUpdate(w, "updatedAt", session.UpdatedAt, params.UpdatedAt) {
			return
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		respondErr(w, fmt.Errorf("find practice session: %w", err))
		return
	}

	ok = resolveTombstone(w, params.WrittenAt, tombstone{
		find: func() (time.Time, error) {
			marker, err := q.FindDeletedPracticeSessionMarker(r.Context(), database.FindDeletedPracticeSessionMarkerParams{
				User:      user,
				SessionID: id,
			})
			return marker.DeletedAt, err
		},
		remove: func() error {
			return q.DeleteDeletedPracticeSessionMarker(r.Context(), database.DeleteDeletedPracticeSessionMarkerParams{
				User:      user,
				SessionID: id,
			})
		},
	})
	if !ok {
		return
	}

	endedAt := sql.NullTime{}
	if params.EndedAt != nil {
		endedAt = sql.NullTime{Time: *params.EndedAt, Valid: true}
	}
	routineID := sql.NullString{}
	if params.RoutineID != nil {
		routineID = sql.NullString{String: *params.RoutineID, Valid: true}
	}

	err = q.UpsertPracticeSession(r.Context(), database.UpsertPracticeSessionParams{
		ID:           id,
		User:         user,
		UpdatedAt:    params.UpdatedAt,
		StartedAt:    params.StartedAt,
		EndedAt:      endedAt,
		RoutineID:    routineID,
		MetadataJson: params.Metadata,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("upsert practice session: %w", err))
		return
	}

	err = q.RemoveAllPracticeSessionEntries(r.Context(), database.RemoveAllPracticeSessionEntriesParams{
		User:      user,
		SessionID: id,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("remove practice session entries: %w", err))
		return
	}

	for i, entry := range params.Entries {
		// the entries of this session are gone, so a surviving row means the id belongs to another
		// session. Moving it would drop it from that session without touching its changed timestamp,
		// leaving every other device unaware of the removal.
		_, err = q.FindPracticeSessionEntry(r.Context(), database.FindPracticeSessionEntryParams{
			User: user,
			ID:   entry.ID,
		})
		if err == nil {
			respondBadRequest(w)
			return
		} else if !errors.Is(err, sql.ErrNoRows) {
			respondErr(w, fmt.Errorf("find practice session entry %s: %w", entry.ID, err))
			return
		}

		routineEntryID := sql.NullString{}
		if entry.RoutineEntryID != nil {
			routineEntryID = sql.NullString{String: *entry.RoutineEntryID, Valid: true}
		}
		err = q.AddPracticeSessionEntry(r.Context(), database.AddPracticeSessionEntryParams{
			User:           user,
			ID:             entry.ID,
			SessionID:      id,
			ExerciseID:     entry.ExerciseID,
			RoutineEntryID: routineEntryID,
			MetadataJson:   entry.Metadata,
		})
		if err != nil {
			respondErr(w, fmt.Errorf("add practice session entry %d: %w", i, err))
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

// DELETE /api/practice/session/:id
func (h *Handler) handleDeletePracticeSession(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	id := chi.URLParam(r, "id")

	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		respondInternalServerError(w, fmt.Errorf("begin tx: %w", err))
		return
	}
	defer tx.Rollback()

	q := h.Queries.WithTx(tx)
	result, err := q.DeletePracticeSession(r.Context(), database.DeletePracticeSessionParams{
		User: user,
		ID:   id,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("delete practice session: %w", err))
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

	err = q.CreateDeletedPracticeSessionMarker(r.Context(), database.CreateDeletedPracticeSessionMarkerParams{
		SessionID: id,
		User:      user,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("create deleted practice session marker: %w", err))
		return
	}

	err = tx.Commit()
	if err != nil {
		respondInternalServerError(w, fmt.Errorf("commit tx: %w", err))
		return
	}

	respondOK(w)
}
