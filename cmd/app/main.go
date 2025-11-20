package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"avito/internal/service"
	"avito/internal/storage/pgx"
	transport "avito/internal/transport/http"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("env DATABASE_URL is empty")
	}

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	st, err := pgx.NewPgxStorage(ctx, dsn)
	if err != nil {
		log.Fatalf("failed to init storage: %v", err)
	}
	defer st.Close()

	if err := st.Ping(ctx); err != nil {
		log.Fatalf("db ping failed: %v", err)
	}

	svc := service.NewService(
		st, // TeamStorage
		st, // UserStorage
		st, // PullRequestStorage
		st, // txManager
	)

	router := transport.NewHandler(
		svc, // TeamsService
		svc, // UsersService
		svc, // PullRequestsService
	)

	srv := &http.Server{
		Addr:         addr,
		Handler:      router.Routes(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("HTTP server listening on %s\n", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("ListenAndServe failed: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("signal received, shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	} else {
		log.Println("HTTP server gracefully stopped")
	}
}
