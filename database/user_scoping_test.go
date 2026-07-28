package database

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

const (
	userA = "alice"
	userB = "bob"
	// the same ids are deliberately used for both users
	scoreID   = "score-1"
	tagID     = "tag-1"
	setlistID = "setlist-1"
)

func setupDB(t *testing.T) (context.Context, *Queries) {
	t.Helper()
	ctx := context.Background()

	db, queries, err := Open(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %s", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	for _, user := range []string{userA, userB} {
		err = queries.CreateUser(ctx, CreateUserParams{Name: user, PasswordHash: "hash"})
		if err != nil {
			t.Fatalf("create user %s: %s", user, err)
		}
	}
	return ctx, queries
}

// seedUser creates a score, a tag and a setlist under the shared ids with content identifying the
// owner, assigns the tag to the score and puts one entry into the setlist.
func seedUser(t *testing.T, ctx context.Context, q *Queries, user string) {
	t.Helper()
	updatedAt := time.Unix(1000, 0)

	err := q.UpsertScore(ctx, UpsertScoreParams{
		ID:                scoreID,
		User:              user,
		MetadataUpdatedAt: updatedAt,
		Title:             user + " score",
		MetadataJson:      []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("upsert score for %s: %s", user, err)
	}

	err = q.UpsertTag(ctx, UpsertTagParams{
		ID:        tagID,
		User:      user,
		UpdatedAt: updatedAt,
		Name:      user + " tag",
		Color:     1,
	})
	if err != nil {
		t.Fatalf("upsert tag for %s: %s", user, err)
	}

	err = q.AssignTag(ctx, AssignTagParams{User: user, ScoreID: scoreID, TagID: tagID})
	if err != nil {
		t.Fatalf("assign tag for %s: %s", user, err)
	}

	err = q.UpsertSetlist(ctx, UpsertSetlistParams{
		ID:        setlistID,
		User:      user,
		UpdatedAt: updatedAt,
		Name:      user + " setlist",
	})
	if err != nil {
		t.Fatalf("upsert setlist for %s: %s", user, err)
	}

	err = q.AddSetlistEntry(ctx, AddSetlistEntryParams{
		User:      user,
		SetlistID: setlistID,
		Position:  0,
		ScoreID:   user + " entry",
	})
	if err != nil {
		t.Fatalf("add setlist entry for %s: %s", user, err)
	}

	// the sync poll query skips scores without a file, so give each score one
	_, err = q.UpdateFileInfo(ctx, UpdateFileInfoParams{
		FileUpdatedAt:   updatedAt,
		FileUpdatedAt_2: updatedAt,
		FileType:        "pdf",
		User:            user,
		ID:              scoreID,
	})
	if err != nil {
		t.Fatalf("update file info for %s: %s", user, err)
	}
}

func TestSameIDsAreIndependentPerUser(t *testing.T) {
	ctx, q := setupDB(t)
	seedUser(t, ctx, q, userA)
	seedUser(t, ctx, q, userB)

	for _, user := range []string{userA, userB} {
		score, err := q.FindScore(ctx, FindScoreParams{User: user, ID: scoreID})
		if err != nil {
			t.Fatalf("find score of %s: %s", user, err)
		}
		if score.Title != user+" score" {
			t.Errorf("FindScore returned title %q for %s", score.Title, user)
		}

		tag, err := q.FindTag(ctx, FindTagParams{User: user, ID: tagID})
		if err != nil {
			t.Fatalf("find tag of %s: %s", user, err)
		}
		if tag.Name != user+" tag" {
			t.Errorf("FindTag returned name %q for %s", tag.Name, user)
		}

		setlist, err := q.FindSetlist(ctx, FindSetlistParams{User: user, ID: setlistID})
		if err != nil {
			t.Fatalf("find setlist of %s: %s", user, err)
		}
		if setlist.Name != user+" setlist" {
			t.Errorf("FindSetlist returned name %q for %s", setlist.Name, user)
		}

		entries, err := q.GetSetlistScoreIDs(ctx, GetSetlistScoreIDsParams{User: user, SetlistID: setlistID})
		if err != nil {
			t.Fatalf("get setlist score ids of %s: %s", user, err)
		}
		if !slices.Equal(entries, []string{user + " entry"}) {
			t.Errorf("GetSetlistScoreIDs returned %v for %s", entries, user)
		}

		tagIDs, err := q.GetAssignedTagIDs(ctx, GetAssignedTagIDsParams{User: user, ScoreID: scoreID})
		if err != nil {
			t.Fatalf("get assigned tag ids of %s: %s", user, err)
		}
		if !slices.Equal(tagIDs, []string{tagID}) {
			t.Errorf("GetAssignedTagIDs returned %v for %s", tagIDs, user)
		}

		tags, err := q.FindTagsChangedAfter(ctx, FindTagsChangedAfterParams{
			User:    user,
			Changed: time.Unix(0, 0),
		})
		if err != nil {
			t.Fatalf("find tags changed after for %s: %s", user, err)
		}
		if len(tags) != 1 {
			t.Fatalf("FindTagsChangedAfter returned %d rows for %s, want 1", len(tags), user)
		}
		if tags[0].User != user || tags[0].Name != user+" tag" {
			t.Errorf("FindTagsChangedAfter returned %q owned by %q for %s", tags[0].Name, tags[0].User, user)
		}

		scoreRows, err := q.FindScoresChangedAfterWithTagIds(ctx, FindScoresChangedAfterWithTagIdsParams{
			User:    user,
			Changed: time.Unix(0, 0),
		})
		if err != nil {
			t.Fatalf("find scores changed after for %s: %s", user, err)
		}
		if len(scoreRows) != 1 {
			t.Fatalf("FindScoresChangedAfterWithTagIds returned %d rows for %s, want 1", len(scoreRows), user)
		}
		if scoreRows[0].User != user || scoreRows[0].Title != user+" score" {
			t.Errorf("FindScoresChangedAfterWithTagIds returned %q owned by %q for %s", scoreRows[0].Title, scoreRows[0].User, user)
		}

		setlistRows, err := q.FindSetlistsChangedAfterWithScoreIds(ctx, FindSetlistsChangedAfterWithScoreIdsParams{
			User:    user,
			Changed: time.Unix(0, 0),
		})
		if err != nil {
			t.Fatalf("find setlists changed after for %s: %s", user, err)
		}
		if len(setlistRows) != 1 {
			t.Fatalf("FindSetlistsChangedAfterWithScoreIds returned %d rows for %s, want 1", len(setlistRows), user)
		}
		if setlistRows[0].User != user || setlistRows[0].ScoreID.String != user+" entry" {
			t.Errorf("FindSetlistsChangedAfterWithScoreIds returned entry %q owned by %q for %s", setlistRows[0].ScoreID.String, setlistRows[0].User, user)
		}
	}
}

func TestUpdateFileInfoOnlyTouchesOwnScore(t *testing.T) {
	ctx, q := setupDB(t)
	seedUser(t, ctx, q, userA)
	seedUser(t, ctx, q, userB)

	newTime := time.Unix(2000, 0)
	_, err := q.UpdateFileInfo(ctx, UpdateFileInfoParams{
		FileUpdatedAt:   newTime,
		FileUpdatedAt_2: newTime,
		FileType:        "pdf",
		User:            userA,
		ID:              scoreID,
	})
	if err != nil {
		t.Fatalf("update file info: %s", err)
	}

	score, err := q.FindScore(ctx, FindScoreParams{User: userB, ID: scoreID})
	if err != nil {
		t.Fatalf("find score of %s: %s", userB, err)
	}
	if !score.FileUpdatedAt.Equal(time.Unix(1000, 0)) {
		t.Errorf("UpdateFileInfo as %s changed the file info of %s to %s", userA, userB, score.FileUpdatedAt)
	}
}

func TestDeletesLeaveTheOtherUserIntact(t *testing.T) {
	ctx, q := setupDB(t)
	seedUser(t, ctx, q, userA)
	seedUser(t, ctx, q, userB)

	err := q.UnassignAllTags(ctx, UnassignAllTagsParams{User: userA, ScoreID: scoreID})
	if err != nil {
		t.Fatalf("unassign all tags: %s", err)
	}
	tagIDs, err := q.GetAssignedTagIDs(ctx, GetAssignedTagIDsParams{User: userB, ScoreID: scoreID})
	if err != nil {
		t.Fatalf("get assigned tag ids: %s", err)
	}
	if !slices.Equal(tagIDs, []string{tagID}) {
		t.Errorf("UnassignAllTags as %s removed the assignments of %s", userA, userB)
	}

	err = q.RemoveAllSetlistEntries(ctx, RemoveAllSetlistEntriesParams{User: userA, SetlistID: setlistID})
	if err != nil {
		t.Fatalf("remove all setlist entries: %s", err)
	}
	entries, err := q.GetSetlistScoreIDs(ctx, GetSetlistScoreIDsParams{User: userB, SetlistID: setlistID})
	if err != nil {
		t.Fatalf("get setlist score ids: %s", err)
	}
	if !slices.Equal(entries, []string{userB + " entry"}) {
		t.Errorf("RemoveAllSetlistEntries as %s removed the entries of %s", userA, userB)
	}

	result, err := q.DeleteScore(ctx, DeleteScoreParams{User: userA, ID: scoreID})
	if err != nil {
		t.Fatalf("delete score: %s", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("rows affected: %s", err)
	}
	if affected != 1 {
		t.Errorf("DeleteScore affected %d rows, want 1", affected)
	}
	_, err = q.FindScore(ctx, FindScoreParams{User: userB, ID: scoreID})
	if err != nil {
		t.Errorf("DeleteScore as %s removed the score of %s: %s", userA, userB, err)
	}

	_, err = q.DeleteTag(ctx, DeleteTagParams{User: userA, ID: tagID})
	if err != nil {
		t.Fatalf("delete tag: %s", err)
	}
	_, err = q.FindTag(ctx, FindTagParams{User: userB, ID: tagID})
	if err != nil {
		t.Errorf("DeleteTag as %s removed the tag of %s: %s", userA, userB, err)
	}

	_, err = q.DeleteSetlist(ctx, DeleteSetlistParams{User: userA, ID: setlistID})
	if err != nil {
		t.Fatalf("delete setlist: %s", err)
	}
	_, err = q.FindSetlist(ctx, FindSetlistParams{User: userB, ID: setlistID})
	if err != nil {
		t.Errorf("DeleteSetlist as %s removed the setlist of %s: %s", userA, userB, err)
	}
}

func TestTombstoneOfOneUserDoesNotBlockAnother(t *testing.T) {
	ctx, q := setupDB(t)
	seedUser(t, ctx, q, userA)

	err := q.CreateDeletedScoreMarker(ctx, CreateDeletedScoreMarkerParams{User: userA, ScoreID: scoreID})
	if err != nil {
		t.Fatalf("create deleted score marker: %s", err)
	}
	err = q.CreateDeletedTagMarker(ctx, CreateDeletedTagMarkerParams{User: userA, TagID: tagID})
	if err != nil {
		t.Fatalf("create deleted tag marker: %s", err)
	}
	err = q.CreateDeletedSetlistMarker(ctx, CreateDeletedSetlistMarkerParams{User: userA, SetlistID: setlistID})
	if err != nil {
		t.Fatalf("create deleted setlist marker: %s", err)
	}

	_, err = q.FindDeletedScoreMarker(ctx, FindDeletedScoreMarkerParams{User: userB, ScoreID: scoreID})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("FindDeletedScoreMarker found the tombstone of %s for %s: %v", userA, userB, err)
	}
	_, err = q.FindDeletedTagMarker(ctx, FindDeletedTagMarkerParams{User: userB, TagID: tagID})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("FindDeletedTagMarker found the tombstone of %s for %s: %v", userA, userB, err)
	}
	_, err = q.FindDeletedSetlistMarker(ctx, FindDeletedSetlistMarkerParams{User: userB, SetlistID: setlistID})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("FindDeletedSetlistMarker found the tombstone of %s for %s: %v", userA, userB, err)
	}

	epoch := time.Unix(0, 0)
	deletedScores, err := q.FindDeletedScoresSince(ctx, FindDeletedScoresSinceParams{User: userB, DeletedAt: epoch})
	if err != nil {
		t.Fatalf("find deleted scores since: %s", err)
	}
	if len(deletedScores) != 0 {
		t.Errorf("FindDeletedScoresSince returned %v of %s for %s", deletedScores, userA, userB)
	}
	deletedTags, err := q.FindDeletedTagsSince(ctx, FindDeletedTagsSinceParams{User: userB, DeletedAt: epoch})
	if err != nil {
		t.Fatalf("find deleted tags since: %s", err)
	}
	if len(deletedTags) != 0 {
		t.Errorf("FindDeletedTagsSince returned %v of %s for %s", deletedTags, userA, userB)
	}
	deletedSetlists, err := q.FindDeletedSetlistsSince(ctx, FindDeletedSetlistsSinceParams{User: userB, DeletedAt: epoch})
	if err != nil {
		t.Fatalf("find deleted setlists since: %s", err)
	}
	if len(deletedSetlists) != 0 {
		t.Errorf("FindDeletedSetlistsSince returned %v of %s for %s", deletedSetlists, userA, userB)
	}

	// the tombstones of A must not stop B from uploading the same ids
	seedUser(t, ctx, q, userB)
}

func TestDeletingAUserLeavesTheOtherIntact(t *testing.T) {
	ctx, q := setupDB(t)
	seedUser(t, ctx, q, userA)
	seedUser(t, ctx, q, userB)

	_, err := q.DeleteUser(ctx, userA)
	if err != nil {
		t.Fatalf("delete user: %s", err)
	}

	_, err = q.FindScore(ctx, FindScoreParams{User: userB, ID: scoreID})
	if err != nil {
		t.Errorf("deleting %s removed the score of %s: %s", userA, userB, err)
	}
	tagIDs, err := q.GetAssignedTagIDs(ctx, GetAssignedTagIDsParams{User: userB, ScoreID: scoreID})
	if err != nil {
		t.Fatalf("get assigned tag ids: %s", err)
	}
	if !slices.Equal(tagIDs, []string{tagID}) {
		t.Errorf("deleting %s removed the tag assignments of %s", userA, userB)
	}
	entries, err := q.GetSetlistScoreIDs(ctx, GetSetlistScoreIDsParams{User: userB, SetlistID: setlistID})
	if err != nil {
		t.Fatalf("get setlist score ids: %s", err)
	}
	if !slices.Equal(entries, []string{userB + " entry"}) {
		t.Errorf("deleting %s removed the setlist entries of %s", userA, userB)
	}

	_, err = q.FindScore(ctx, FindScoreParams{User: userA, ID: scoreID})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("score of deleted user %s survived: %v", userA, err)
	}
}
