package handlers

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/juho05/sheetopia-sync/config"

	"github.com/juho05/sheetopia-sync/database"
)

// GET /api/score/:id/file?fileType=<type>
func (h *Handler) handleGetScoreFile(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	id := chi.URLParam(r, "id")

	fileType := database.FileType(chi.URLParam(r, "fileType"))
	if fileType != "" && !fileType.Valid() {
		respondBadRequest(w)
		return
	}

	score, err := h.Queries.FindScoreByUser(r.Context(), database.FindScoreByUserParams{
		User: user,
		ID:   id,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("find score: %w", err))
		return
	}
	if score.FileType == "none" || (fileType != "" && string(fileType) != score.FileType) {
		respondConflict(w)
		return
	}

	filePath := scoreFilePath(score.ID, database.FileType(score.FileType))

	http.ServeFile(w, r, filePath)
}

// POST /api/score/:id/file?updatedAt=<time>&fileType=<type>
func (h *Handler) handleUpdateScoreFile(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	id := chi.URLParam(r, "id")

	updatedAtStr := r.URL.Query().Get("updatedAt")
	if updatedAtStr == "" {
		respondBadRequest(w)
		return
	}

	updatedAt, err := time.Parse(time.RFC3339, updatedAtStr)
	if err != nil {
		respondBadRequest(w)
		return
	}

	fileType := database.FileType(r.URL.Query().Get("fileType"))
	if !fileType.Valid() {
		respondBadRequest(w)
		return
	}

	err = os.MkdirAll(scoreDirPath(id), 0o755)
	if err != nil {
		respondInternalServerError(w, fmt.Errorf("create score file dir: %w", err))
		return
	}

	score, err := h.Queries.FindScoreByUser(r.Context(), database.FindScoreByUserParams{
		User: user,
		ID:   id,
	})
	if err != nil {
		respondErr(w, err)
		return
	}

	if !score.FileUpdatedAt.Before(updatedAt) {
		respondConflict(w)
		return
	}

	filePath := scoreFilePath(id, fileType)

	partFile, err := os.Create(filePath + ".part")
	if err != nil {
		respondInternalServerError(w, fmt.Errorf("create part file: %w", err))
		return
	}
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			log.Printf("failed to delete temporary .part file: %s", err)
		}
	}(partFile.Name())

	// Limit file size to 1 GB because the client currently does not handle file sizes this must be
	// large enough that it is never reached in normal usage.
	_, err = io.Copy(partFile, http.MaxBytesReader(w, r.Body, 1000<<20))
	closeErr := partFile.Close()
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			respondBadRequest(w)
		} else {
			respondInternalServerError(w, fmt.Errorf("receive uploaded file: %w", err))
		}
		return
	}
	if closeErr != nil {
		respondInternalServerError(w, fmt.Errorf("close part file: %w", closeErr))
		return
	}

	result, err := h.Queries.UpdateFileInfo(r.Context(), database.UpdateFileInfoParams{
		FileUpdatedAt:   updatedAt,
		FileUpdatedAt_2: updatedAt,
		FileType:        string(fileType),
		ID:              id,
	})
	if err != nil {
		respondErr(w, err)
		return
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		respondErr(w, err)
		return
	}
	if rowsAffected == 0 {
		// file updated time has changed during the upload causing this version to not be latest anymore
		respondConflict(w)
		return
	}

	err = os.Rename(partFile.Name(), filePath)
	if err != nil {
		respondInternalServerError(w, fmt.Errorf("rename .part to final path: %w", err))
		return
	}

	if score.FileType != "none" && database.FileType(score.FileType) != fileType {
		p := scoreFilePath(score.ID, database.FileType(score.FileType))
		err = os.Remove(p)
		if err != nil {
			log.Printf("failed to remove old score file %s: %s", p, err)
		}
	}

	respondOK(w)
}

func scoreDirPath(scoreID string) string {
	encoded := base64.URLEncoding.EncodeToString([]byte(scoreID))
	return filepath.Join(config.DataDir, "scores", encoded)
}

func scoreFilePath(scoreID string, fileType database.FileType) string {
	var extension string
	switch fileType {
	case database.FileTypePDF:
		extension = ".pdf"
	}

	return filepath.Join(scoreDirPath(scoreID), "score"+extension)
}
