package handlers

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/juho05/sheetopia-sync/database"
)

type exerciseCategoryRequest struct {
	Name      string     `json:"name"`
	Position  int        `json:"position"`
	UpdatedAt time.Time  `json:"updatedAt"`
	WrittenAt *time.Time `json:"writtenAt,omitempty"`
}

type exerciseRequest struct {
	Name       string          `json:"name"`
	CategoryID *string         `json:"categoryId"`
	TagIDs     []string        `json:"tagIds"`
	ScoreIDs   []string        `json:"scoreIds"`
	Metadata   json.RawMessage `json:"metadata"`
	UpdatedAt  time.Time       `json:"updatedAt"`
	WrittenAt  *time.Time      `json:"writtenAt,omitempty"`
}

type practiceRoutineRequest struct {
	Name      string                 `json:"name"`
	Metadata  json.RawMessage        `json:"metadata"`
	Entries   []practiceRoutineEntry `json:"entries"`
	UpdatedAt time.Time              `json:"updatedAt"`
	WrittenAt *time.Time             `json:"writtenAt,omitempty"`
}

type practiceSessionRequest struct {
	StartedAt time.Time              `json:"startedAt"`
	EndedAt   *time.Time             `json:"endedAt"`
	RoutineID *string                `json:"routineId"`
	Metadata  json.RawMessage        `json:"metadata"`
	Entries   []practiceSessionEntry `json:"entries"`
	UpdatedAt time.Time              `json:"updatedAt"`
	WrittenAt *time.Time             `json:"writtenAt,omitempty"`
}

type deletedResponse struct {
	Deleted []deletedItemResponse `json:"deleted"`
}

func newExerciseRequest(updatedAt time.Time) exerciseRequest {
	return exerciseRequest{
		Name:      "exercise",
		Metadata:  json.RawMessage(`{"description":"warm up"}`),
		UpdatedAt: updatedAt,
	}
}

func TestExerciseCategoryRoundTrip(t *testing.T) {
	_, _, h := setupHandler(t)

	w := request(t, h, http.MethodPost, "/api/practice/category/cat-1", exerciseCategoryRequest{
		Name:      "Scales",
		Position:  2,
		UpdatedAt: time.Unix(1000, 0),
	})
	if w.Code != http.StatusOK {
		t.Fatalf("upsert returned %d: %s", w.Code, w.Body.String())
	}

	w = request(t, h, http.MethodGet, "/api/practice/category", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get categories returned %d: %s", w.Code, w.Body.String())
	}
	body := decodeResponse[struct {
		Categories []exerciseCategoryResponse `json:"categories"`
	}](t, w)
	if len(body.Categories) != 1 {
		t.Fatalf("get categories returned %d rows, want 1", len(body.Categories))
	}
	if body.Categories[0].Name != "Scales" || body.Categories[0].Position != 2 {
		t.Errorf("get categories returned %+v", body.Categories[0])
	}

	// an update that is not newer than the stored one is rejected
	w = request(t, h, http.MethodPost, "/api/practice/category/cat-1", exerciseCategoryRequest{
		Name:      "Other",
		UpdatedAt: time.Unix(1000, 0),
	})
	if w.Code != http.StatusConflict {
		t.Fatalf("stale upsert returned %d, want 409", w.Code)
	}
}

func TestExerciseRoundTrip(t *testing.T) {
	_, _, h := setupHandler(t)

	w := request(t, h, http.MethodPost, "/api/tag/tag-1", tagRequest{
		Name:      "tag",
		UpdatedAt: time.Unix(1000, 0),
		Type:      "exercise",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("upsert tag returned %d: %s", w.Code, w.Body.String())
	}

	categoryID := "cat-1"
	req := newExerciseRequest(time.Unix(1000, 0))
	req.CategoryID = &categoryID
	req.TagIDs = []string{"tag-1"}
	req.ScoreIDs = []string{"score-2", "score-1"}
	w = request(t, h, http.MethodPost, "/api/practice/exercise/exercise-1", req)
	if w.Code != http.StatusOK {
		t.Fatalf("upsert exercise returned %d: %s", w.Code, w.Body.String())
	}

	w = request(t, h, http.MethodGet, "/api/practice/exercise", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get exercises returned %d: %s", w.Code, w.Body.String())
	}
	body := decodeResponse[struct {
		Exercises []exerciseResponse `json:"exercises"`
	}](t, w)
	if len(body.Exercises) != 1 {
		t.Fatalf("get exercises returned %d rows, want 1", len(body.Exercises))
	}
	exercise := body.Exercises[0]
	if exercise.CategoryID == nil || *exercise.CategoryID != categoryID {
		t.Errorf("exercise categoryId is %v, want %s", exercise.CategoryID, categoryID)
	}
	if !slices.Equal(exercise.TagIDs, []string{"tag-1"}) {
		t.Errorf("exercise tagIds are %v, want [tag-1]", exercise.TagIDs)
	}
	// the score order is part of the exercise, alternatives are picked by position
	if !slices.Equal(exercise.ScoreIDs, []string{"score-2", "score-1"}) {
		t.Errorf("exercise scoreIds are %v, want [score-2 score-1]", exercise.ScoreIDs)
	}
	if string(exercise.Metadata) != `{"description":"warm up"}` {
		t.Errorf("exercise metadata is %s", exercise.Metadata)
	}

	// a second write replaces the children instead of appending to them
	req = newExerciseRequest(time.Unix(2000, 0))
	req.ScoreIDs = []string{"score-3"}
	w = request(t, h, http.MethodPost, "/api/practice/exercise/exercise-1", req)
	if w.Code != http.StatusOK {
		t.Fatalf("second upsert returned %d: %s", w.Code, w.Body.String())
	}
	w = request(t, h, http.MethodGet, "/api/practice/exercise/exercise-1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get exercise returned %d: %s", w.Code, w.Body.String())
	}
	exercise = decodeResponse[exerciseResponse](t, w)
	if len(exercise.TagIDs) != 0 {
		t.Errorf("exercise tagIds are %v, want []", exercise.TagIDs)
	}
	if !slices.Equal(exercise.ScoreIDs, []string{"score-3"}) {
		t.Errorf("exercise scoreIds are %v, want [score-3]", exercise.ScoreIDs)
	}
	if exercise.CategoryID != nil {
		t.Errorf("exercise categoryId is %v, want null", *exercise.CategoryID)
	}
}

func TestExerciseWithUnknownTagIsRejected(t *testing.T) {
	_, _, h := setupHandler(t)

	req := newExerciseRequest(time.Unix(1000, 0))
	req.TagIDs = []string{"missing"}
	w := request(t, h, http.MethodPost, "/api/practice/exercise/exercise-1", req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("upsert with unknown tag returned %d, want 400", w.Code)
	}

	w = request(t, h, http.MethodGet, "/api/practice/exercise/exercise-1", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("the rejected exercise was stored anyway: %d", w.Code)
	}
}

func TestResurrectExercise(t *testing.T) {
	ctx, q, h := setupHandler(t)

	w := request(t, h, http.MethodPost, "/api/practice/exercise/exercise-1", newExerciseRequest(time.Unix(1000, 0)))
	if w.Code != http.StatusOK {
		t.Fatalf("upsert returned %d: %s", w.Code, w.Body.String())
	}
	w = request(t, h, http.MethodDelete, "/api/practice/exercise/exercise-1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete returned %d: %s", w.Code, w.Body.String())
	}

	w = request(t, h, http.MethodGet, "/api/practice/exercise/deleted", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("deleted exercises returned %d: %s", w.Code, w.Body.String())
	}
	deleted := decodeResponse[deletedResponse](t, w)
	if len(deleted.Deleted) != 1 || deleted.Deleted[0].ID != "exercise-1" {
		t.Fatalf("deleted exercises are %v, want one entry for exercise-1", deleted.Deleted)
	}
	if deleted.Deleted[0].DeletedAt.IsZero() {
		t.Error("deleted exercise reported a zero deletedAt")
	}

	// an ordinary write loses against the tombstone
	req := newExerciseRequest(time.Unix(2000, 0))
	w = request(t, h, http.MethodPost, "/api/practice/exercise/exercise-1", req)
	if w.Code != http.StatusConflict {
		t.Fatalf("upsert of a deleted exercise returned %d, want 409", w.Code)
	}

	// a write restored by an import after the deletion wins and clears the tombstone
	req.WrittenAt = ptr(time.Now().Add(time.Hour))
	w = request(t, h, http.MethodPost, "/api/practice/exercise/exercise-1", req)
	if w.Code != http.StatusOK {
		t.Fatalf("resurrect returned %d: %s", w.Code, w.Body.String())
	}
	markers, err := q.FindDeletedExercisesSince(ctx, database.FindDeletedExercisesSinceParams{
		User:      testUser,
		DeletedAt: time.Unix(0, 0),
	})
	if err != nil {
		t.Fatalf("find deleted exercises since: %s", err)
	}
	if len(markers) != 0 {
		t.Errorf("the tombstone of the resurrected exercise survived: %v", markers)
	}
}

func TestPracticeRoutineEntriesKeepTheirOrder(t *testing.T) {
	_, _, h := setupHandler(t)

	entries := []practiceRoutineEntry{
		{ID: "entry-2", ExerciseID: "exercise-2", Metadata: json.RawMessage(`{"targetDuration":60000}`)},
		{ID: "entry-1", ExerciseID: "exercise-1", Metadata: json.RawMessage(`{}`)},
	}
	w := request(t, h, http.MethodPost, "/api/practice/routine/routine-1", practiceRoutineRequest{
		Name:      "Morning",
		Metadata:  json.RawMessage(`{"description":"before breakfast"}`),
		Entries:   entries,
		UpdatedAt: time.Unix(1000, 0),
	})
	if w.Code != http.StatusOK {
		t.Fatalf("upsert returned %d: %s", w.Code, w.Body.String())
	}

	w = request(t, h, http.MethodGet, "/api/practice/routine", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get routines returned %d: %s", w.Code, w.Body.String())
	}
	body := decodeResponse[struct {
		Routines []practiceRoutineResponse `json:"routines"`
	}](t, w)
	if len(body.Routines) != 1 {
		t.Fatalf("get routines returned %d rows, want 1", len(body.Routines))
	}
	got := body.Routines[0].Entries
	if len(got) != 2 || got[0].ID != "entry-2" || got[1].ID != "entry-1" {
		t.Fatalf("routine entries are %+v, want entry-2 before entry-1", got)
	}
	if string(got[0].Metadata) != `{"targetDuration":60000}` {
		t.Errorf("entry metadata is %s", got[0].Metadata)
	}

	// removing an entry removes it from the server as well
	w = request(t, h, http.MethodPost, "/api/practice/routine/routine-1", practiceRoutineRequest{
		Name:      "Morning",
		Metadata:  json.RawMessage(`{}`),
		Entries:   entries[1:],
		UpdatedAt: time.Unix(2000, 0),
	})
	if w.Code != http.StatusOK {
		t.Fatalf("second upsert returned %d: %s", w.Code, w.Body.String())
	}
	w = request(t, h, http.MethodGet, "/api/practice/routine/routine-1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get routine returned %d: %s", w.Code, w.Body.String())
	}
	routine := decodeResponse[practiceRoutineResponse](t, w)
	if len(routine.Entries) != 1 || routine.Entries[0].ID != "entry-1" {
		t.Errorf("routine entries are %+v, want only entry-1", routine.Entries)
	}
}

func TestPracticeRoutineWithDuplicateEntryIDsIsRejected(t *testing.T) {
	_, _, h := setupHandler(t)

	w := request(t, h, http.MethodPost, "/api/practice/routine/routine-1", practiceRoutineRequest{
		Name:     "Morning",
		Metadata: json.RawMessage(`{}`),
		Entries: []practiceRoutineEntry{
			{ID: "entry-1", ExerciseID: "exercise-1", Metadata: json.RawMessage(`{}`)},
			{ID: "entry-1", ExerciseID: "exercise-2", Metadata: json.RawMessage(`{}`)},
		},
		UpdatedAt: time.Unix(1000, 0),
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("upsert with duplicate entry ids returned %d, want 400", w.Code)
	}
}

func TestPracticeRoutineEntryOfAnotherRoutineIsRejected(t *testing.T) {
	_, _, h := setupHandler(t)

	entry := practiceRoutineEntry{ID: "entry-1", ExerciseID: "exercise-1", Metadata: json.RawMessage(`{}`)}
	w := request(t, h, http.MethodPost, "/api/practice/routine/routine-1", practiceRoutineRequest{
		Name:      "Morning",
		Metadata:  json.RawMessage(`{}`),
		Entries:   []practiceRoutineEntry{entry},
		UpdatedAt: time.Unix(1000, 0),
	})
	if w.Code != http.StatusOK {
		t.Fatalf("upsert returned %d: %s", w.Code, w.Body.String())
	}

	w = request(t, h, http.MethodPost, "/api/practice/routine/routine-2", practiceRoutineRequest{
		Name:      "Evening",
		Metadata:  json.RawMessage(`{}`),
		Entries:   []practiceRoutineEntry{entry},
		UpdatedAt: time.Unix(1000, 0),
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("stealing an entry returned %d, want 400", w.Code)
	}

	// the entry stayed where it was
	w = request(t, h, http.MethodGet, "/api/practice/routine/routine-1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get routine returned %d: %s", w.Code, w.Body.String())
	}
	routine := decodeResponse[practiceRoutineResponse](t, w)
	if len(routine.Entries) != 1 || routine.Entries[0].ID != "entry-1" {
		t.Errorf("routine-1 entries are %+v, want entry-1", routine.Entries)
	}
}

func TestPracticeSessionEntryOfAnotherSessionIsRejected(t *testing.T) {
	_, _, h := setupHandler(t)

	entry := practiceSessionEntry{ID: "entry-1", ExerciseID: "exercise-1", Metadata: json.RawMessage(`{}`)}
	w := request(t, h, http.MethodPost, "/api/practice/session/session-1", practiceSessionRequest{
		StartedAt: time.Unix(1000, 0),
		Metadata:  json.RawMessage(`{}`),
		Entries:   []practiceSessionEntry{entry},
		UpdatedAt: time.Unix(1000, 0),
	})
	if w.Code != http.StatusOK {
		t.Fatalf("upsert returned %d: %s", w.Code, w.Body.String())
	}

	w = request(t, h, http.MethodPost, "/api/practice/session/session-2", practiceSessionRequest{
		StartedAt: time.Unix(2000, 0),
		Metadata:  json.RawMessage(`{}`),
		Entries:   []practiceSessionEntry{entry},
		UpdatedAt: time.Unix(2000, 0),
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("stealing an entry returned %d, want 400", w.Code)
	}

	w = request(t, h, http.MethodGet, "/api/practice/session/session-1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get session returned %d: %s", w.Code, w.Body.String())
	}
	session := decodeResponse[practiceSessionResponse](t, w)
	if len(session.Entries) != 1 || session.Entries[0].ID != "entry-1" {
		t.Errorf("session-1 entries are %+v, want entry-1", session.Entries)
	}
}

func TestPracticeSessionRoundTrip(t *testing.T) {
	_, _, h := setupHandler(t)

	endedAt := time.Unix(3000, 0).UTC()
	routineID := "routine-1"
	entryID := "routine-entry-1"
	w := request(t, h, http.MethodPost, "/api/practice/session/session-1", practiceSessionRequest{
		StartedAt: time.Unix(2000, 0).UTC(),
		EndedAt:   &endedAt,
		RoutineID: &routineID,
		Metadata:  json.RawMessage(`{"description":"evening"}`),
		Entries: []practiceSessionEntry{
			{ID: "session-entry-1", ExerciseID: "exercise-1", RoutineEntryID: &entryID, Metadata: json.RawMessage(`{"duration":1000}`)},
		},
		UpdatedAt: time.Unix(3000, 0),
	})
	if w.Code != http.StatusOK {
		t.Fatalf("upsert returned %d: %s", w.Code, w.Body.String())
	}

	w = request(t, h, http.MethodGet, "/api/practice/session/session-1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get session returned %d: %s", w.Code, w.Body.String())
	}
	session := decodeResponse[practiceSessionResponse](t, w)
	if !session.StartedAt.Equal(time.Unix(2000, 0)) {
		t.Errorf("session startedAt is %s", session.StartedAt)
	}
	if session.EndedAt == nil || !session.EndedAt.Equal(endedAt) {
		t.Errorf("session endedAt is %v, want %s", session.EndedAt, endedAt)
	}
	if session.RoutineID == nil || *session.RoutineID != routineID {
		t.Errorf("session routineId is %v, want %s", session.RoutineID, routineID)
	}
	if len(session.Entries) != 1 {
		t.Fatalf("session entries are %+v, want one entry", session.Entries)
	}
	if session.Entries[0].RoutineEntryID == nil || *session.Entries[0].RoutineEntryID != entryID {
		t.Errorf("session entry routineEntryId is %v, want %s", session.Entries[0].RoutineEntryID, entryID)
	}

	// a session that was never finished keeps its open end
	w = request(t, h, http.MethodPost, "/api/practice/session/session-2", practiceSessionRequest{
		StartedAt: time.Unix(2000, 0).UTC(),
		Metadata:  json.RawMessage(`{}`),
		UpdatedAt: time.Unix(3000, 0),
	})
	if w.Code != http.StatusOK {
		t.Fatalf("upsert of an open session returned %d: %s", w.Code, w.Body.String())
	}
	w = request(t, h, http.MethodGet, "/api/practice/session/session-2", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get open session returned %d: %s", w.Code, w.Body.String())
	}
	session = decodeResponse[practiceSessionResponse](t, w)
	if session.EndedAt != nil {
		t.Errorf("open session reported endedAt %s", session.EndedAt)
	}
	if session.RoutineID != nil {
		t.Errorf("session without a routine reported routineId %s", *session.RoutineID)
	}
	if len(session.Entries) != 0 {
		t.Errorf("session without entries reported %+v", session.Entries)
	}
}

func TestChangedAfterOnlyReturnsNewerRows(t *testing.T) {
	_, _, h := setupHandler(t)

	w := request(t, h, http.MethodPost, "/api/practice/exercise/exercise-1", newExerciseRequest(time.Unix(1000, 0)))
	if w.Code != http.StatusOK {
		t.Fatalf("upsert returned %d: %s", w.Code, w.Body.String())
	}

	future := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	w = request(t, h, http.MethodGet, "/api/practice/exercise?changedAfter="+future, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get exercises returned %d: %s", w.Code, w.Body.String())
	}
	body := decodeResponse[struct {
		Exercises []exerciseResponse `json:"exercises"`
	}](t, w)
	if len(body.Exercises) != 0 {
		t.Errorf("get exercises returned %d rows for a future changedAfter, want 0", len(body.Exercises))
	}
}

func TestTagTypeSurvivesAnUpdateFromAClientWithoutTypes(t *testing.T) {
	_, _, h := setupHandler(t)

	w := request(t, h, http.MethodPost, "/api/tag/tag-1", tagRequest{
		Name:      "tag",
		UpdatedAt: time.Unix(1000, 0),
		Type:      "exercise",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("upsert returned %d: %s", w.Code, w.Body.String())
	}

	w = request(t, h, http.MethodPost, "/api/tag/tag-1", tagRequest{
		Name:      "renamed",
		UpdatedAt: time.Unix(2000, 0),
	})
	if w.Code != http.StatusOK {
		t.Fatalf("second upsert returned %d: %s", w.Code, w.Body.String())
	}

	w = request(t, h, http.MethodGet, "/api/tag/tag-1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get tag returned %d: %s", w.Code, w.Body.String())
	}
	tag := decodeResponse[tagResponse](t, w)
	if tag.Type == nil || *tag.Type != "exercise" {
		t.Errorf("tag type is %v, want exercise", tag.Type)
	}
}

func TestTagWithoutATypeOmitsIt(t *testing.T) {
	_, _, h := setupHandler(t)

	// a missing type tells a downloading client that the writer did not know about types
	w := request(t, h, http.MethodPost, "/api/tag/tag-1", tagRequest{
		Name:      "tag",
		UpdatedAt: time.Unix(1000, 0),
	})
	if w.Code != http.StatusOK {
		t.Fatalf("upsert returned %d: %s", w.Code, w.Body.String())
	}

	w = request(t, h, http.MethodGet, "/api/tag/tag-1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get tag returned %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "type") {
		t.Errorf("the response carries a type: %s", w.Body.String())
	}
	tag := decodeResponse[tagResponse](t, w)
	if tag.Type != nil {
		t.Errorf("tag type is %q, want none", *tag.Type)
	}
}

func TestUnknownTypeIsStoredAsItIs(t *testing.T) {
	_, _, h := setupHandler(t)

	// a type the server does not know about must survive so that a new client can introduce one
	w := request(t, h, http.MethodPost, "/api/tag/tag-1", tagRequest{
		Name:      "tag",
		UpdatedAt: time.Unix(1000, 0),
		Type:      "from-a-newer-client",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("upsert returned %d: %s", w.Code, w.Body.String())
	}

	w = request(t, h, http.MethodGet, "/api/tag/tag-1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get tag returned %d: %s", w.Code, w.Body.String())
	}
	tag := decodeResponse[tagResponse](t, w)
	if tag.Type == nil || *tag.Type != "from-a-newer-client" {
		t.Errorf("tag type is %v, want from-a-newer-client", tag.Type)
	}
}
