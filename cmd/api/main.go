package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"github.com/96malhar/greenlight/internal/data"
	_ "github.com/lib/pq"
	"log/slog"
	"net/http"
	"os"
	"time"
)

const version = "1.0.0"

type config struct {
	port int
	env  string
	db   struct {
		dsn string
	}
}

type application struct {
	config     config
	logger     *slog.Logger
	modelStore data.ModelStore
}

type envelope map[string]any

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	var cfg config

	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("GREENLIGHT_DB_DSN"), "PostgreSQL DSN")
	flag.Parse()

	db, err := openDB(cfg)
	if err != nil {
		logger.Error(err.Error())
		logger.Error("cannot connect to database", "dsn", cfg.db.dsn)
		os.Exit(1)
	}
	defer db.Close()

	app := &application{
		config:     cfg,
		logger:     logger,
		modelStore: data.NewModelStore(db),
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		ErrorLog:     slog.NewLogLogger(app.logger.Handler(), slog.LevelError),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	logger.Info("starting server", "port", srv.Addr, "env", cfg.env)
	err = srv.ListenAndServe()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}

func openDB(cfg config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	// Create a context with a 5-second timeout deadline.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	// Return the sql.DB connection pool.
	return db, nil
}
