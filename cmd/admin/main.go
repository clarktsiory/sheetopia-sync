package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/juho05/sheetopia-sync/database"
)

var ErrUsage = errors.New("usage")

func run() error {
	ctx := context.Background()
	db, queries, err := database.Open(ctx, "database.sqlite")
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	if len(os.Args) < 2 {
		fmt.Println("USAGE:", os.Args[0], "<command>\n\nCOMMANDS:\n  users")
		return ErrUsage
	}

	switch os.Args[1] {
	case "users":
		err = users(os.Args, queries)
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
