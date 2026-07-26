package storage

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/juho05/sheetopia-sync/database"
)

// MigrateLegacyScoreFiles moves score files from the old scores/<id> layout to the user scoped
// scores/<user>/<id> layout. It does nothing once the move has happened.
// TODO: remove at some point
func MigrateLegacyScoreFiles(ctx context.Context, queries *database.Queries) error {
	scoresDir := ScoresDir()
	legacyDir := LegacyScoresDir()

	_, err := os.Stat(legacyDir)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("stat legacy score dir: %w", err)
		}
		// no interrupted run to finish, so decide from the layout on disk
		legacy, err := isLegacyLayout(scoresDir)
		if err != nil {
			return fmt.Errorf("detect score file layout: %w", err)
		}
		if !legacy {
			return nil
		}
		err = os.Rename(scoresDir, legacyDir)
		if err != nil {
			return fmt.Errorf("move score dir aside: %w", err)
		}
		err = os.MkdirAll(scoresDir, 0o755)
		if err != nil {
			return fmt.Errorf("recreate score dir: %w", err)
		}
	}

	scores, err := queries.FindAllScoreIDs(ctx)
	if err != nil {
		return fmt.Errorf("find all score ids: %w", err)
	}

	// Moving each directory with os.Rename makes this loop idempotent: an entry an interrupted run
	// already moved has no source left and is skipped, so rerunning simply finishes the job.
	moved := 0
	for _, score := range scores {
		source := filepath.Join(legacyDir, encode(score.ID))
		_, err = os.Stat(source)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("stat legacy dir of score %s: %w", score.ID, err)
		}
		target := ScoreDir(score.User, score.ID)
		// don't overwrite existing files in migrated dir
		_, err = os.Stat(target)
		if err == nil {
			continue
		}
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("stat score dir of score %s: %w", score.ID, err)
		}
		err = os.MkdirAll(filepath.Dir(target), 0o755)
		if err != nil {
			return fmt.Errorf("create score dir of user %s: %w", score.User, err)
		}
		err = os.Rename(source, target)
		if err != nil {
			return fmt.Errorf("move score dir of score %s: %w", score.ID, err)
		}
		moved++
	}

	leftovers, err := os.ReadDir(legacyDir)
	if err != nil {
		return fmt.Errorf("read legacy score dir: %w", err)
	}
	if len(leftovers) > 0 {
		log.Printf("migrated %d score directories to the user scoped layout, %d entries in %s belong to no score and were left in place", moved, len(leftovers), legacyDir)
		return nil
	}

	err = os.Remove(legacyDir)
	if err != nil {
		return fmt.Errorf("remove legacy score dir: %w", err)
	}
	log.Printf("migrated %d score directories to the user scoped layout", moved)
	return nil
}

func isLegacyLayout(scoresDir string) (bool, error) {
	children, err := os.ReadDir(scoresDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("read score dir: %w", err)
	}

	sawFile := false
	for _, child := range children {
		if !child.IsDir() {
			continue
		}
		entries, err := os.ReadDir(filepath.Join(scoresDir, child.Name()))
		if err != nil {
			return false, fmt.Errorf("read score dir %s: %w", child.Name(), err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				return false, nil
			}
			if entry.Type().IsRegular() {
				sawFile = true
			}
		}
	}
	return sawFile, nil
}
