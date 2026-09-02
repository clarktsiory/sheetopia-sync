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

type practiceRoutineEntry struct {
	ID         string          `json:"id"`
	ExerciseID string          `json:"exerciseId"`
	Metadata   json.RawMessage `json:"metadata"`
}

type practiceRoutineResponse struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Metadata  json.RawMessage        `json:"metadata"`
	Entries   []practiceRoutineEntry `json:"entries"`
	UpdatedAt time.Time              `json:"updatedAt"`
}

func newPracticeRoutineResponse(routine database.PracticeRoutine, entries []practiceRoutineEntry) practiceRoutineResponse {
	if entries == nil {
		entries = make([]practiceRoutineEntry, 0)
	}
	return practiceRoutineResponse{
		ID:        routine.ID,
		Name:      routine.Name,
		Metadata:  routine.MetadataJson,
		Entries:   entries,
		UpdatedAt: routine.UpdatedAt,
	}
}

func newPracticeRoutineEntry(entry database.PracticeRoutineEntry) practiceRoutineEntry {
	return practiceRoutineEntry{
		ID:         entry.ID,
		ExerciseID: entry.ExerciseID,
		Metadata:   entry.MetadataJson,
	}
}

// GET /api/practice/routine?changedAfter=<time>
func (h *Handler) handleGetPracticeRoutines(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)

	changedAfter, ok := parseTime(w, r.URL.Query().Get("changedAfter"), time.Unix(0, 0))
	if !ok {
		return
	}

	rows, err := h.Queries.FindPracticeRoutinesChangedAfter(r.Context(), database.FindPracticeRoutinesChangedAfterParams{
		User:    user,
		Changed: changedAfter,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("find practice routines changed after time: %w", err))
		return
	}

	entryRows, err := h.Queries.FindPracticeRoutineEntriesChangedAfter(r.Context(), database.FindPracticeRoutineEntriesChangedAfterParams{
		User:    user,
		Changed: changedAfter,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("find entries of practice routines changed after time: %w", err))
		return
	}
	entries := make(map[string][]practiceRoutineEntry, len(rows))
	for _, row := range entryRows {
		entries[row.RoutineID] = append(entries[row.RoutineID], newPracticeRoutineEntry(row))
	}

	routines := make([]practiceRoutineResponse, 0, len(rows))
	for _, row := range rows {
		routines = append(routines, newPracticeRoutineResponse(row, entries[row.ID]))
	}

	type response struct {
		Routines []practiceRoutineResponse `json:"routines"`
	}
	respond(w, response{
		Routines: routines,
	}, http.StatusOK)
}

// GET /api/practice/routine/deleted?since=<time>
func (h *Handler) handleGetDeletedPracticeRoutines(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	since, ok := parseTime(w, r.URL.Query().Get("since"), time.Unix(0, 0))
	if !ok {
		return
	}

	rows, err := h.Queries.FindDeletedPracticeRoutinesSince(r.Context(), database.FindDeletedPracticeRoutinesSinceParams{
		User:      user,
		DeletedAt: since,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("find deleted practice routines since time: %w", err))
		return
	}

	deleted := make([]deletedItem, 0, len(rows))
	for _, row := range rows {
		deleted = append(deleted, deletedItem{ID: row.RoutineID, DeletedAt: row.DeletedAt})
	}

	type response struct {
		Deleted []deletedItem `json:"deleted"`
	}
	respond(w, response{
		Deleted: deleted,
	}, http.StatusOK)
}

// GET /api/practice/routine/:id
func (h *Handler) handleGetPracticeRoutine(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	id := chi.URLParam(r, "id")

	routine, err := h.Queries.FindPracticeRoutine(r.Context(), database.FindPracticeRoutineParams{
		User: user,
		ID:   id,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("find practice routine: %w", err))
		return
	}

	rows, err := h.Queries.GetPracticeRoutineEntries(r.Context(), database.GetPracticeRoutineEntriesParams{
		User:      user,
		RoutineID: id,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("get practice routine entries: %w", err))
		return
	}

	entries := make([]practiceRoutineEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, newPracticeRoutineEntry(row))
	}

	respond(w, newPracticeRoutineResponse(routine, entries), http.StatusOK)
}

// POST /api/practice/routine/:id
func (h *Handler) handleUpdatePracticeRoutine(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	id := chi.URLParam(r, "id")
	if id == "" {
		respondBadRequest(w)
		return
	}

	type request struct {
		Name      string                 `json:"name"`
		Metadata  json.RawMessage        `json:"metadata"`
		Entries   []practiceRoutineEntry `json:"entries"`
		UpdatedAt time.Time              `json:"updatedAt"`
		WrittenAt time.Time              `json:"writtenAt"`
	}
	params, ok := decodeBody[request](w, r)
	if !ok {
		return
	}

	if params.Name == "" || params.UpdatedAt.IsZero() || params.Metadata == nil {
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

	routine, err := q.FindPracticeRoutine(r.Context(), database.FindPracticeRoutineParams{
		User: user,
		ID:   id,
	})
	if err == nil {
		if staleUpdate(w, "updatedAt", routine.UpdatedAt, params.UpdatedAt) {
			return
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		respondErr(w, fmt.Errorf("find practice routine: %w", err))
		return
	}

	ok = resolveTombstone(w, params.WrittenAt, tombstone{
		find: func() (time.Time, error) {
			marker, err := q.FindDeletedPracticeRoutineMarker(r.Context(), database.FindDeletedPracticeRoutineMarkerParams{
				User:      user,
				RoutineID: id,
			})
			return marker.DeletedAt, err
		},
		remove: func() error {
			return q.DeleteDeletedPracticeRoutineMarker(r.Context(), database.DeleteDeletedPracticeRoutineMarkerParams{
				User:      user,
				RoutineID: id,
			})
		},
	})
	if !ok {
		return
	}

	err = q.UpsertPracticeRoutine(r.Context(), database.UpsertPracticeRoutineParams{
		ID:           id,
		User:         user,
		UpdatedAt:    params.UpdatedAt,
		Name:         params.Name,
		MetadataJson: params.Metadata,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("upsert practice routine: %w", err))
		return
	}

	err = q.RemoveAllPracticeRoutineEntries(r.Context(), database.RemoveAllPracticeRoutineEntriesParams{
		User:      user,
		RoutineID: id,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("remove practice routine entries: %w", err))
		return
	}

	for i, entry := range params.Entries {
		// the entries of this routine are gone, so a surviving row means the id belongs to another
		// routine. Moving it would drop it from that routine without touching its changed timestamp,
		// leaving every other device unaware of the removal.
		_, err = q.FindPracticeRoutineEntry(r.Context(), database.FindPracticeRoutineEntryParams{
			User: user,
			ID:   entry.ID,
		})
		if err == nil {
			respondBadRequest(w)
			return
		} else if !errors.Is(err, sql.ErrNoRows) {
			respondErr(w, fmt.Errorf("find practice routine entry %s: %w", entry.ID, err))
			return
		}

		err = q.AddPracticeRoutineEntry(r.Context(), database.AddPracticeRoutineEntryParams{
			User:         user,
			ID:           entry.ID,
			RoutineID:    id,
			Position:     int64(i),
			ExerciseID:   entry.ExerciseID,
			MetadataJson: entry.Metadata,
		})
		if err != nil {
			respondErr(w, fmt.Errorf("add practice routine entry %d: %w", i, err))
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

// DELETE /api/practice/routine/:id
func (h *Handler) handleDeletePracticeRoutine(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	id := chi.URLParam(r, "id")

	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		respondInternalServerError(w, fmt.Errorf("begin tx: %w", err))
		return
	}
	defer tx.Rollback()

	q := h.Queries.WithTx(tx)
	result, err := q.DeletePracticeRoutine(r.Context(), database.DeletePracticeRoutineParams{
		User: user,
		ID:   id,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("delete practice routine: %w", err))
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

	err = q.CreateDeletedPracticeRoutineMarker(r.Context(), database.CreateDeletedPracticeRoutineMarkerParams{
		RoutineID: id,
		User:      user,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("create deleted practice routine marker: %w", err))
		return
	}

	err = tx.Commit()
	if err != nil {
		respondInternalServerError(w, fmt.Errorf("commit tx: %w", err))
		return
	}

	respondOK(w)
}
