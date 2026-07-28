package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"
)

type deletedItem struct {
	ID        string    `json:"id"`
	DeletedAt time.Time `json:"deletedAt"`
}

func getUser(r *http.Request) string {
	user := r.Context().Value("user")
	if user == nil {
		panic(errors.New("user not found in context"))
	}
	userStr := user.(string)
	if len(userStr) == 0 {
		panic(errors.New("user is empty"))
	}
	return userStr
}

func respond[T any](w http.ResponseWriter, v T, status int) {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = encoder.Encode(v)
}

func respondOK(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
}

func respondConflict(w http.ResponseWriter) {
	respondStatus(w, http.StatusConflict)
}

func respondUnauthorized(w http.ResponseWriter) {
	respondStatus(w, http.StatusUnauthorized)
}

func respondErr(w http.ResponseWriter, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		respondNotFound(w)
		return
	}
	respondInternalServerError(w, err)
}

func respondInternalServerError(w http.ResponseWriter, err error) {
	log.Printf("Internal Server Error: %s", err)
	respondStatus(w, http.StatusInternalServerError)
}

func respondNotFound(w http.ResponseWriter) {
	respondStatus(w, http.StatusNotFound)
}

func respondBadRequest(w http.ResponseWriter) {
	respondStatus(w, http.StatusBadRequest)
}

func respondStatus(w http.ResponseWriter, status int) {
	w.WriteHeader(status)
	_, _ = w.Write([]byte(http.StatusText(status)))
}

func parseTime(w http.ResponseWriter, str string, def time.Time) (time.Time, bool) {
	if str != "" {
		var err error
		t, err := time.Parse(time.RFC3339, str)
		if err != nil {
			respondBadRequest(w)
			return time.Time{}, false
		}
		return t, true
	}
	return def, true
}

func decodeBody[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var obj T
	err := json.NewDecoder(r.Body).Decode(&obj)
	_ = r.Body.Close()
	if err != nil {
		respondBadRequest(w)
		return obj, false
	}
	return obj, true
}
