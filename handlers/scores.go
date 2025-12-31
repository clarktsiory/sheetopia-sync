package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/juho05/sheetopia-sync/database"
)

type scoreResponse struct {
	ID                string            `json:"id"`
	Title             string            `json:"title"`
	MetadataUpdatedAt time.Time         `json:"metadataUpdatedAt"`
	FileUpdatedAt     time.Time         `json:"fileUpdatedAt"`
	FileType          database.FileType `json:"fileType"`
	TagIDs            []string          `json:"tagIds"`
	Metadata          json.RawMessage   `json:"metadata"`
}

// GET /api/scores?changedAfter=<time>
func (h *Handler) handleGetScores(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)

	changedAfter, ok := parseTime(w, r.URL.Query().Get("changedAfter"), time.Unix(0, 0))
	if !ok {
		return
	}

	results, err := h.Queries.FindScoresChangedAfterWithTagIds(r.Context(), database.FindScoresChangedAfterWithTagIdsParams{
		User:    user,
		Changed: changedAfter,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("find scores changed after time with tag ids: %w", err))
		return
	}

	scores := make([]scoreResponse, 0, len(results)/2)
	for _, result := range results {
		if len(scores) == 0 || scores[len(scores)-1].ID != result.ID {
			initialTagIdsCapacity := 0
			if result.TagID.Valid {
				initialTagIdsCapacity = 1
			}
			scores = append(scores, scoreResponse{
				ID:                result.ID,
				Title:             result.Title,
				MetadataUpdatedAt: result.MetadataUpdatedAt,
				FileUpdatedAt:     result.FileUpdatedAt,
				FileType:          database.FileType(result.FileType),
				Metadata:          result.MetadataJson,
				TagIDs:            make([]string, 0, initialTagIdsCapacity),
			})
		}
		if result.TagID.Valid {
			scores[len(scores)-1].TagIDs = append(scores[len(scores)-1].TagIDs, result.TagID.String)
		}
	}

	type response struct {
		Scores []scoreResponse `json:"scores"`
	}
	respond(w, response{
		Scores: scores,
	}, http.StatusOK)
}

// GET /api/score/deleted?since=<time>
func (h *Handler) handleGetDeletedScores(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	since, ok := parseTime(w, r.URL.Query().Get("since"), time.Unix(0, 0))
	if !ok {
		return
	}

	ids, err := h.Queries.FindDeletedScoreIDsSince(r.Context(), database.FindDeletedScoreIDsSinceParams{
		User:      user,
		DeletedAt: since,
	})
	if err != nil {
		respondErr(w, err)
		return
	}

	type response struct {
		ScoreIDs []string `json:"scoreIds"`
	}
	respond(w, response{
		ScoreIDs: ids,
	}, http.StatusOK)
}

// GET /api/score/:id
func (h *Handler) handleGetScore(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	id := chi.URLParam(r, "id")

	score, err := h.Queries.FindScoreByUser(r.Context(), database.FindScoreByUserParams{
		User: user,
		ID:   id,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("find score: %w", err))
		return
	}
	if score.FileType == "none" {
		respondConflict(w)
		return
	}

	tagIDs, err := h.Queries.GetAssignedTagIDs(r.Context(), id)
	if err != nil {
		respondErr(w, fmt.Errorf("get assigned tag ids: %w", err))
		return
	}

	respond(w, scoreResponse{
		ID:                score.ID,
		Title:             score.Title,
		MetadataUpdatedAt: score.MetadataUpdatedAt,
		FileUpdatedAt:     score.FileUpdatedAt,
		FileType:          database.FileType(score.FileType),
		TagIDs:            tagIDs,
		Metadata:          score.MetadataJson,
	}, http.StatusOK)
}

// POST /api/score/:id
func (h *Handler) handleUpdateScore(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)

	id := chi.URLParam(r, "id")
	if id == "" {
		respondBadRequest(w)
		return
	}

	type request struct {
		Title             string          `json:"title"`
		MetadataUpdatedAt time.Time       `json:"metadataUpdatedAt"`
		Metadata          json.RawMessage `json:"metadata"`
		TagIDs            []string        `json:"tagIds"`
	}
	params, ok := decodeBody[request](w, r)
	if !ok {
		return
	}

	if params.Title == "" || params.MetadataUpdatedAt.IsZero() || params.Metadata == nil {
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

	score, err := q.FindScore(r.Context(), id)
	if err == nil {
		if score.User != user {
			respondForbidden(w)
			return
		}
		if !score.MetadataUpdatedAt.Before(params.MetadataUpdatedAt) {
			type response struct {
				MetadataUpdatedAt time.Time `json:"metadataUpdatedAt"`
			}
			respond(w, response{
				MetadataUpdatedAt: score.MetadataUpdatedAt,
			}, http.StatusConflict)
			return
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		respondErr(w, fmt.Errorf("find score: %w", err))
		return
	}

	deletedScoreMarker, err := q.FindDeletedScoreMarker(r.Context(), id)
	if err == nil {
		type response struct {
			DeletedAt time.Time `json:"deletedAt"`
		}
		respond(w, response{
			DeletedAt: deletedScoreMarker.DeletedAt,
		}, http.StatusConflict)
		return
	}
	if !errors.Is(err, sql.ErrNoRows) {
		respondErr(w, fmt.Errorf("check if already deleted: %w", err))
		return
	}

	err = q.UpsertScore(r.Context(), database.UpsertScoreParams{
		ID:                id,
		User:              user,
		MetadataUpdatedAt: params.MetadataUpdatedAt,
		Title:             params.Title,
		MetadataJson:      params.Metadata,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("upsert score: %w", err))
		return
	}

	err = q.UnassignAllTags(r.Context(), id)
	if err != nil {
		respondErr(w, fmt.Errorf("unassign tags: %w", err))
		return
	}

	for _, tagID := range params.TagIDs {
		err = q.AssignTag(r.Context(), database.AssignTagParams{
			ScoreID: id,
			TagID:   tagID,
		})
		if err != nil {
			respondErr(w, fmt.Errorf("assign tag %s: %w", tagID, err))
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

// DELETE /api/score/:id
func (h *Handler) handleDeleteScore(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	id := chi.URLParam(r, "id")

	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		respondInternalServerError(w, fmt.Errorf("begin tx: %w", err))
		return
	}
	defer tx.Rollback()

	q := h.Queries.WithTx(tx)
	result, err := q.DeleteScore(r.Context(), database.DeleteScoreParams{
		User: user,
		ID:   id,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("delete score: %w", err))
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

	err = q.CreateDeletedScoreMarker(r.Context(), database.CreateDeletedScoreMarkerParams{
		ScoreID: id,
		User:    user,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("create deleted score marker: %w", err))
		return
	}

	err = tx.Commit()
	if err != nil {
		respondInternalServerError(w, fmt.Errorf("commit tx: %w", err))
		return
	}

	dir := scoreDirPath(id)
	err = os.RemoveAll(dir)
	if err != nil {
		log.Printf("failed to delete score file of deleted score: %s", err)
	}

	respondOK(w)
}
