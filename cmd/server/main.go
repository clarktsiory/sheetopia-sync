package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/juho05/sheetopia-sync/config"
	"github.com/juho05/sheetopia-sync/database"
	"github.com/juho05/sheetopia-sync/handlers"
)

func main() {
	ctx := context.Background()

	err := config.Load()
	if err != nil {
		log.Fatalf("invalid config: %s", err)
	}

	db, queries, err := database.Open(ctx, filepath.Join(config.DataDir, "database.sqlite"))
	if err != nil {
		log.Fatalf("Failed to open database: %s", err)
	}
	defer db.Close()

	handler := handlers.NewHandler(db, queries)

	server := http.Server{
		Addr:    fmt.Sprintf(":%d", config.Port),
		Handler: handler,
	}

	closed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM)
		<-sigint
		timeout, cancelTimeout := context.WithTimeout(context.Background(), 5*time.Second)
		err = server.Shutdown(timeout)
		if err != nil {
			log.Printf("shutdown: %s", err)
		}
		cancelTimeout()
		close(closed)
	}()

	log.Printf("listening on %s...", server.Addr)
	err = server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}
	if err == nil {
		<-closed
	}
}
