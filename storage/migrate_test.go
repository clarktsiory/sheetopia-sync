package storage

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/juho05/sheetopia-sync/config"
	"github.com/juho05/sheetopia-sync/database"
)

func writeFile(t *testing.T, path string) {
	t.Helper()
	err := os.MkdirAll(filepath.Dir(path), 0o755)
	if err != nil {
		t.Fatalf("create dir of %s: %s", path, err)
	}
	err = os.WriteFile(path, []byte(path), 0o644)
	if err != nil {
		t.Fatalf("write %s: %s", path, err)
	}
}

func TestIsLegacyLayout(t *testing.T) {
	tests := []struct {
		name      string
		files     []string
		emptyDirs []string
		want      bool
	}{
		{
			name:  "legacy tree",
			files: []string{filepath.Join(encode("s1"), "score.pdf"), filepath.Join(encode("s2"), "score.pdf")},
			want:  true,
		},
		{
			name:  "legacy tree with an interrupted upload",
			files: []string{filepath.Join(encode("s1"), "score.pdf.part")},
			want:  true,
		},
		{
			name:  "migrated tree",
			files: []string{filepath.Join(encode("alice"), encode("s1"), "score.pdf")},
			want:  false,
		},
		{
			name:      "migrated tree with an empty user dir",
			emptyDirs: []string{encode("alice")},
			want:      false,
		},
		{
			name:      "migrated tree where only one user has files",
			files:     []string{filepath.Join(encode("bob"), encode("s1"), "score.pdf")},
			emptyDirs: []string{encode("alice")},
			want:      false,
		},
		{
			name: "empty scores dir",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scoresDir := filepath.Join(t.TempDir(), "scores")
			err := os.MkdirAll(scoresDir, 0o755)
			if err != nil {
				t.Fatalf("create scores dir: %s", err)
			}
			for _, dir := range tt.emptyDirs {
				err = os.MkdirAll(filepath.Join(scoresDir, dir), 0o755)
				if err != nil {
					t.Fatalf("create %s: %s", dir, err)
				}
			}
			for _, file := range tt.files {
				writeFile(t, filepath.Join(scoresDir, file))
			}

			got, err := isLegacyLayout(scoresDir)
			if err != nil {
				t.Fatalf("is legacy layout: %s", err)
			}
			if got != tt.want {
				t.Errorf("isLegacyLayout() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsLegacyLayoutOnMissingScoresDir(t *testing.T) {
	got, err := isLegacyLayout(filepath.Join(t.TempDir(), "scores"))
	if err != nil {
		t.Fatalf("is legacy layout: %s", err)
	}
	if got {
		t.Error("isLegacyLayout() = true for a missing scores dir")
	}
}

// setupMigration builds a data dir holding a database with the given scores and a legacy score file
// tree. The map keys are score ids, the values their owner.
func setupMigration(t *testing.T, scores map[string]string) (context.Context, *database.Queries) {
	t.Helper()
	ctx := context.Background()

	config.DataDir = t.TempDir()
	db, queries, err := database.Open(ctx, filepath.Join(config.DataDir, "database.sqlite"))
	if err != nil {
		t.Fatalf("open db: %s", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	users := make(map[string]struct{})
	for id, user := range scores {
		if _, ok := users[user]; !ok {
			err = queries.CreateUser(ctx, database.CreateUserParams{Name: user, PasswordHash: "hash"})
			if err != nil {
				t.Fatalf("create user %s: %s", user, err)
			}
			users[user] = struct{}{}
		}
		err = queries.UpsertScore(ctx, database.UpsertScoreParams{
			ID:                id,
			User:              user,
			MetadataUpdatedAt: time.Unix(1000, 0),
			Title:             id,
			MetadataJson:      []byte(`{}`),
		})
		if err != nil {
			t.Fatalf("upsert score %s: %s", id, err)
		}
		// legacy layout: scores/<b64 id>/score.pdf
		writeFile(t, filepath.Join(ScoresDir(), encode(id), "score.pdf"))
	}
	return ctx, queries
}

func assertMigrated(t *testing.T, scores map[string]string) {
	t.Helper()
	for id, user := range scores {
		path := filepath.Join(ScoreDir(user, id), "score.pdf")
		content, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("read migrated file of score %s: %s", id, err)
			continue
		}
		// the content is the legacy path it was written at, so this also proves nothing was mixed up
		want := filepath.Join(ScoresDir(), encode(id), "score.pdf")
		if string(content) != want {
			t.Errorf("file at %s holds %q, want %q", path, content, want)
		}
	}
	_, err := os.Stat(LegacyScoresDir())
	if err == nil {
		t.Errorf("%s still exists after a complete migration", LegacyScoresDir())
	}
}

func TestMigrateLegacyScoreFiles(t *testing.T) {
	scores := map[string]string{"s1": "alice", "s2": "alice", "s3": "bob"}
	ctx, queries := setupMigration(t, scores)

	err := MigrateLegacyScoreFiles(ctx, queries)
	if err != nil {
		t.Fatalf("migrate: %s", err)
	}
	assertMigrated(t, scores)
}

func TestMigrateLegacyScoreFilesIsIdempotent(t *testing.T) {
	scores := map[string]string{"s1": "alice", "s2": "bob"}
	ctx, queries := setupMigration(t, scores)

	for i := range 3 {
		err := MigrateLegacyScoreFiles(ctx, queries)
		if err != nil {
			t.Fatalf("migrate run %d: %s", i, err)
		}
		assertMigrated(t, scores)
	}
}

func TestMigrateLegacyScoreFilesResumes(t *testing.T) {
	scores := map[string]string{"s1": "alice", "s2": "alice", "s3": "bob", "s4": "bob"}
	ctx, queries := setupMigration(t, scores)

	// simulate a run that was interrupted after moving the first two score dirs
	err := os.Rename(ScoresDir(), LegacyScoresDir())
	if err != nil {
		t.Fatalf("move scores dir aside: %s", err)
	}
	err = os.MkdirAll(ScoresDir(), 0o755)
	if err != nil {
		t.Fatalf("recreate scores dir: %s", err)
	}
	for _, id := range []string{"s1", "s2"} {
		target := ScoreDir(scores[id], id)
		err = os.MkdirAll(filepath.Dir(target), 0o755)
		if err != nil {
			t.Fatalf("create user dir: %s", err)
		}
		err = os.Rename(filepath.Join(LegacyScoresDir(), encode(id)), target)
		if err != nil {
			t.Fatalf("move %s: %s", id, err)
		}
	}

	err = MigrateLegacyScoreFiles(ctx, queries)
	if err != nil {
		t.Fatalf("migrate: %s", err)
	}
	assertMigrated(t, scores)
}

func TestMigrateLegacyScoreFilesKeepsOrphans(t *testing.T) {
	scores := map[string]string{"s1": "alice"}
	ctx, queries := setupMigration(t, scores)
	// a score dir with no database row
	writeFile(t, filepath.Join(ScoresDir(), encode("orphan"), "score.pdf"))

	err := MigrateLegacyScoreFiles(ctx, queries)
	if err != nil {
		t.Fatalf("migrate: %s", err)
	}

	_, err = os.Stat(filepath.Join(ScoreDir("alice", "s1"), "score.pdf"))
	if err != nil {
		t.Errorf("score of alice was not migrated: %s", err)
	}
	leftovers, err := os.ReadDir(LegacyScoresDir())
	if err != nil {
		t.Fatalf("read legacy dir: %s", err)
	}
	names := make([]string, 0, len(leftovers))
	for _, entry := range leftovers {
		names = append(names, entry.Name())
	}
	if !slices.Equal(names, []string{encode("orphan")}) {
		t.Errorf("legacy dir holds %v, want only the orphan", names)
	}
}

// An orphan keeps the legacy dir alive across boots, so a later run sees a score whose id matches
// an orphan and whose file already sits in the new layout.
func TestMigrateLegacyScoreFilesKeepsFilesUploadedAfterAnOrphanRun(t *testing.T) {
	scores := map[string]string{"s1": "alice"}
	ctx, queries := setupMigration(t, scores)
	writeFile(t, filepath.Join(ScoresDir(), encode("orphan"), "score.pdf"))

	err := MigrateLegacyScoreFiles(ctx, queries)
	if err != nil {
		t.Fatalf("first migrate: %s", err)
	}

	// the orphan id is uploaded through the API afterwards, which writes into the new layout
	err = queries.UpsertScore(ctx, database.UpsertScoreParams{
		ID:                "orphan",
		User:              "alice",
		MetadataUpdatedAt: time.Unix(1000, 0),
		Title:             "orphan",
		MetadataJson:      []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("upsert score: %s", err)
	}
	fresh := filepath.Join(ScoreDir("alice", "orphan"), "score.pdf")
	writeFile(t, fresh)

	err = MigrateLegacyScoreFiles(ctx, queries)
	if err != nil {
		t.Fatalf("second migrate: %s", err)
	}
	content, err := os.ReadFile(fresh)
	if err != nil {
		t.Fatalf("read file uploaded after the orphan run: %s", err)
	}
	if string(content) != fresh {
		t.Errorf("the stale legacy file replaced the uploaded one at %s", fresh)
	}
}

func TestMigrateLegacyScoreFilesSkipsCurrentLayout(t *testing.T) {
	ctx, queries := setupMigration(t, map[string]string{"s1": "alice"})

	err := MigrateLegacyScoreFiles(ctx, queries)
	if err != nil {
		t.Fatalf("first migrate: %s", err)
	}

	// a second boot must leave the migrated tree alone, which is what a rule keyed on
	// "scores/ is non-empty" would get wrong
	before := filepath.Join(ScoreDir("alice", "s1"), "score.pdf")
	err = MigrateLegacyScoreFiles(ctx, queries)
	if err != nil {
		t.Fatalf("second migrate: %s", err)
	}
	_, err = os.Stat(before)
	if err != nil {
		t.Errorf("second migration moved the already migrated file: %s", err)
	}
	_, err = os.Stat(LegacyScoresDir())
	if err == nil {
		t.Error("second migration recreated the legacy dir")
	}
}

func TestMigrateLegacyScoreFilesOnFreshInstall(t *testing.T) {
	ctx, queries := setupMigration(t, nil)

	err := MigrateLegacyScoreFiles(ctx, queries)
	if err != nil {
		t.Fatalf("migrate: %s", err)
	}
	_, err = os.Stat(LegacyScoresDir())
	if err == nil {
		t.Error("migration created a legacy dir on a fresh install")
	}
}
