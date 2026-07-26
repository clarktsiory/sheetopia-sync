package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/juho05/sheetopia-sync/config"
)

// seedScoreFiles writes one score file per user into the current layout.
func seedScoreFiles(t *testing.T, users ...string) {
	t.Helper()
	config.DataDir = t.TempDir()
	for _, user := range users {
		writeFile(t, filepath.Join(ScoreDir(user, "s1"), "score.pdf"))
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestDeleteUserScores(t *testing.T) {
	seedScoreFiles(t, "alice", "bob")

	err := DeleteUserScores("alice")
	if err != nil {
		t.Fatalf("delete user scores: %s", err)
	}

	if exists(UserScoresDir("alice")) {
		t.Error("score dir of alice still exists")
	}
	if !exists(filepath.Join(ScoreDir("bob", "s1"), "score.pdf")) {
		t.Error("deleting alice removed the score files of bob")
	}
}

func TestDeleteUserScoresWithoutFiles(t *testing.T) {
	seedScoreFiles(t, "alice")

	// a user that never uploaded a score has no directory at all
	err := DeleteUserScores("bob")
	if err != nil {
		t.Errorf("delete user scores of a user without files: %s", err)
	}
	if !exists(filepath.Join(ScoreDir("alice", "s1"), "score.pdf")) {
		t.Error("score files of alice disappeared")
	}
}

func TestRenameUserScores(t *testing.T) {
	seedScoreFiles(t, "alice", "bob")
	before := filepath.Join(ScoreDir("alice", "s1"), "score.pdf")

	err := RenameUserScores("alice", "carol")
	if err != nil {
		t.Fatalf("rename user scores: %s", err)
	}

	if exists(UserScoresDir("alice")) {
		t.Error("score dir of the old name still exists")
	}
	content, err := os.ReadFile(filepath.Join(ScoreDir("carol", "s1"), "score.pdf"))
	if err != nil {
		t.Fatalf("read score file of carol: %s", err)
	}
	if string(content) != before {
		t.Errorf("score file of carol holds %q, want %q", content, before)
	}
	if !exists(filepath.Join(ScoreDir("bob", "s1"), "score.pdf")) {
		t.Error("renaming alice touched the score files of bob")
	}
}

func TestRenameUserScoresWithoutFiles(t *testing.T) {
	seedScoreFiles(t, "alice")

	err := RenameUserScores("bob", "carol")
	if err != nil {
		t.Errorf("rename user scores of a user without files: %s", err)
	}
	if exists(UserScoresDir("carol")) {
		t.Error("rename created a score dir for a user without files")
	}
}

func TestRenameUserScoresRefusesToOverwrite(t *testing.T) {
	seedScoreFiles(t, "alice", "bob")

	err := RenameUserScores("alice", "bob")
	if err == nil {
		t.Fatal("rename onto an existing score dir did not fail")
	}
	if !exists(filepath.Join(ScoreDir("alice", "s1"), "score.pdf")) {
		t.Error("failed rename lost the score files of alice")
	}
	if !exists(filepath.Join(ScoreDir("bob", "s1"), "score.pdf")) {
		t.Error("failed rename lost the score files of bob")
	}
}
