package main

import (
	"context"
	"expvar"
	"flag"
	"github.com/96malhar/greenlight/internal/data"
	"github.com/96malhar/greenlight/internal/email"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

const version = "1.0.0"

type config struct {
	port int
	env  string
	db   struct {
		dsn          string
		maxIdleTime  time.Duration
		maxOpenConns int
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
	cors struct {
		trustedOrigins []string
	}
}

func (c config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("port", c.port),
		slog.String("env", c.env),

		slog.Int("db-max-open-conns", c.db.maxOpenConns),
		slog.Duration("db-max-idle-time", c.db.maxIdleTime),

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
		config:     cfg,
		logger:     logger,
		mailer:     email.NewMailer(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
		modelStore: data.NewModelStore(db),
	}

	monitorMetrics(db)

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
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.DurationVar(&cfg.db.maxIdleTime, "db-max-idle-time", 15*time.Minute, "PostgreSQL max connection idle time")

	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")

	flag.StringVar(&cfg.smtp.host, "smtp-host", "sandbox.smtp.mailtrap.io", "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 2525, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", os.Getenv("MAILTRAP_UNAME"), "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", os.Getenv("MAILTRAP_PASS"), "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "Greenlight <no-reply@greenlight.alexedwards.net>", "SMTP sender")

	flag.Func("cors-trusted-origins", "Trusted CORS origins (space separated)", func(val string) error {
		cfg.cors.trustedOrigins = strings.Fields(val)
		return nil
	})

	flag.Parse()

	return cfg
}

func openDB(cfg config) (*pgxpool.Pool, error) {
	pgxConf, err := pgxpool.ParseConfig(cfg.db.dsn)
	if err != nil {
		return nil, err
	}
	pgxConf.MaxConnIdleTime = cfg.db.maxIdleTime
	pgxConf.MaxConns = int32(cfg.db.maxOpenConns)

	db, err := pgxpool.NewWithConfig(context.Background(), pgxConf)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.Ping(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func monitorMetrics(pool *pgxpool.Pool) {
	expvar.NewString("version").Set(version)

	// Publish the number of active goroutines.
	expvar.Publish("goroutines", expvar.Func(func() any {
		return runtime.NumGoroutine()
	}))

	//Publish the database connection pool statistics.
	expvar.Publish("database", expvar.Func(func() any {
		st := pool.Stat()
		return map[string]any{
			"maximum_pool_size":            st.MaxConns(),
			"current_pool_size":            st.TotalConns(),
			"current_idle_conns":           st.IdleConns(),
			"current_constructing_conns":   st.ConstructingConns(),
			"current_acquired_conns":       st.AcquiredConns(),
			"total_acquired_conns":         st.AcquireCount(),
			"total_starved_acquired_conns": st.EmptyAcquireCount(),
			"total_acquire_cancelled":      st.CanceledAcquireCount(),
			"total_acquired_duration":      st.AcquireDuration().Seconds(),
			"total_idle_closed":            st.MaxIdleDestroyCount(),
			"total_lifetime_closed":        st.MaxLifetimeDestroyCount(),
		}
	}))

	// Publish the current Unix timestamp.
	expvar.Publish("timestamp", expvar.Func(func() any {
		return time.Now().Unix()
	}))
}
