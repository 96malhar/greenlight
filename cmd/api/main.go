package main

import (
	"context"
	"database/sql"
	"flag"
	"github.com/96malhar/greenlight/internal/data"
	"github.com/96malhar/greenlight/internal/email"
	_ "github.com/lib/pq"
	"log/slog"
	"os"
	"sync"
	"time"
)

const version = "1.0.0"

type config struct {
	port int
	env  string
	db   struct {
		dsn string
	}
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
}

func (c config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("port", c.port),
		slog.String("env", c.env),
		slog.Float64("limiter-rps", c.limiter.rps),
		slog.Int("limiter-burst", c.limiter.burst),
		slog.Bool("limiter-enabled", c.limiter.enabled),
		slog.String("version", version),
	)
}

type application struct {
	config     config
	logger     *slog.Logger
	modelStore data.ModelStore
	mailer     email.MailerInterface
	utcNow     func() time.Time
	wg         sync.WaitGroup
}

type envelope map[string]any

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := parseConfig()

	db, err := openDB(cfg)
	if err != nil {
		logger.Error(err.Error())
		logger.Error("cannot connect to database", "dsn", cfg.db.dsn)
		os.Exit(1)
	}
	defer db.Close()

	app := &application{
		config: cfg,
		logger: logger,
		utcNow: func() time.Time {
			return time.Now().UTC()
		},
		mailer:     email.NewMailer(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
		modelStore: data.NewModelStore(db),
	}

	err = app.serve()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}

func parseConfig() config {
	var cfg config

	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("GREENLIGHT_DB_DSN"), "PostgreSQL DSN")

	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")

	flag.StringVar(&cfg.smtp.host, "smtp-host", "sandbox.smtp.mailtrap.io", "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 2525, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", os.Getenv("MAILTRAP_UNAME"), "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", os.Getenv("MAILTRAP_PASS"), "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "Greenlight <no-reply@greenlight.alexedwards.net>", "SMTP sender")

	flag.Parse()

	return cfg
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
