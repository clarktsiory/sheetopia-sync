package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

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

func respondUnauthorized(w http.ResponseWriter) {
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(http.StatusText(http.StatusUnauthorized)))
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
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
}

func respondNotFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(http.StatusText(http.StatusNotFound)))
}

func respondBadRequest(w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
	_, _ = w.Write([]byte(http.StatusText(http.StatusBadRequest)))
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
