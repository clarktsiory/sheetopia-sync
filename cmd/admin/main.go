package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/juho05/sheetopia-sync/config"
	"github.com/juho05/sheetopia-sync/database"
	"github.com/juho05/sheetopia-sync/storage"
)

var ErrUsage = errors.New("usage")

func run() error {
	ctx := context.Background()

	err := config.Load()
	if err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	db, queries, err := database.Open(ctx, filepath.Join(config.DataDir, "database.sqlite"))
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	err = storage.MigrateLegacyScoreFiles(ctx, queries)
	if err != nil {
		return fmt.Errorf("migrate legacy score files: %w", err)
	}

	if len(os.Args) < 2 {
		fmt.Println("USAGE:", os.Args[0], "<command>\n\nCOMMANDS:\n  users")
		return ErrUsage
	}

	switch os.Args[1] {
	case "users":
		err = users(os.Args, db, queries)
	}
	return err
}

func main() {
	err := run()
	if err != nil {
		if errors.Is(err, ErrUsage) {
			os.Exit(1)
		}
		log.Fatal(err)
	}
}
