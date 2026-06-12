// Command server starts the CourseHub HTTP service. main is the composition
// root: it constructs every concrete dependency (SQLite repositories, grading
// strategies via the factory, the observer publisher) and injects them into the
// services, then into the HTTP server. No other package wires dependencies, so
// the pattern implementations stay decoupled and swappable.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"coursehub/internal/httpapi"
	"coursehub/internal/progress"
	"coursehub/internal/service"
	"coursehub/internal/storage"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dsn := flag.String("db", "coursehub.db", "SQLite database file (use \":memory:\" for ephemeral)")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// Cancel the context on SIGINT/SIGTERM so run can shut down gracefully.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, *addr, *dsn, log); err != nil {
		log.Error("server exited with error", "err", err)
		os.Exit(1)
	}
}

// run wires every dependency and serves until ctx is cancelled or the listener
// fails. ctx is passed in (rather than created here) so tests can drive shutdown
// without sending OS signals.
func run(ctx context.Context, addr, dsn string, log *slog.Logger) error {
	db, err := storage.Open(dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	// Repositories (Repository pattern — concrete SQLite implementations).
	studentRepo := storage.NewStudentRepo(db)
	courseRepo := storage.NewCourseRepo(db)
	enrollmentRepo := storage.NewEnrollmentRepo(db)

	// Observer pattern: the publisher fans grading events out to a progress
	// tracker (persists progress) and a log notifier. enrollmentRepo doubles as
	// the progress.Store. Add observers here to extend behaviour.
	publisher := progress.NewPublisher(
		progress.NewProgressTracker(enrollmentRepo),
		progress.NewLogNotifier(log),
	)

	// Services (business logic).
	studentSvc := service.NewStudentService(studentRepo)
	courseSvc := service.NewCourseService(courseRepo)
	enrollmentSvc := service.NewEnrollmentService(enrollmentRepo, courseSvc, publisher)

	srv := httpapi.NewServer(studentSvc, courseSvc, enrollmentSvc, log)
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", addr, "db", dsn)
		errCh <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		log.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	}
}
