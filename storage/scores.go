// Package storage owns the layout of the score files inside the data directory.
package storage

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/juho05/sheetopia-sync/config"
	"github.com/juho05/sheetopia-sync/database"
)

func encode(s string) string {
	return base64.URLEncoding.EncodeToString([]byte(s))
}

func ScoresDir() string {
	return filepath.Join(config.DataDir, "scores")
}

// LegacyScoresDir is where the pre user scoped layout is parked while it is being migrated.
func LegacyScoresDir() string {
	return filepath.Join(config.DataDir, "scores.legacy")
}

func UserScoresDir(user string) string {
	return filepath.Join(ScoresDir(), encode(user))
}

func ScoreDir(user, scoreID string) string {
	return filepath.Join(UserScoresDir(user), encode(scoreID))
}

func ScoreFile(user, scoreID string, fileType database.FileType) string {
	var extension string
	switch fileType {
	case database.FileTypePDF:
		extension = ".pdf"
	}

	return filepath.Join(ScoreDir(user, scoreID), "score"+extension)
}

// DeleteUserScores removes the score files of a user.
func DeleteUserScores(user string) error {
	err := os.RemoveAll(UserScoresDir(user))
	if err != nil {
		return fmt.Errorf("remove score dir of user %s: %w", user, err)
	}
	return nil
}

// RenameUserScores moves the score files of a user to a new name.
func RenameUserScores(oldName, newName string) error {
	source := UserScoresDir(oldName)
	_, err := os.Stat(source)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat score dir of user %s: %w", oldName, err)
	}

	target := UserScoresDir(newName)
	_, err = os.Stat(target)
	if err == nil {
		return fmt.Errorf("score dir of user %s already exists at %s", newName, target)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat score dir of user %s: %w", newName, err)
	}

	err = os.Rename(source, target)
	if err != nil {
		return fmt.Errorf("move score dir of user %s to %s: %w", oldName, newName, err)
	}
	return nil
}
