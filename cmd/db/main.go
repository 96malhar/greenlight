package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/spf13/cobra"
	"log/slog"
	"os"
	"runtime/debug"
	"time"
)

var dbUser, dbName, dbPassword string
var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
var superUserDSN = "postgresql://postgres:postgres@localhost:5432/?sslmode=disable"

func init() {
	dbName = os.Getenv("DB_NAME")
	dbUser = os.Getenv("DB_USER")
	dbPassword = os.Getenv("DB_PASSWORD")
}

func main() {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Control the database lifecycle for the Greenlight app",
	}
	cmd.AddCommand(setupDB(), teardown())
	must(cmd.Execute())
}

func setupDB() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Initializes the database and the database user used by the Greenlight app",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			db, err := pgx.Connect(ctx, superUserDSN)
			must(err)
			Exec(db, fmt.Sprintf("CREATE DATABASE %s", dbName))
			logger.Info("Created database", "name", dbName)

			Exec(db, fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s'", dbUser, dbPassword))
			logger.Info("Created database user", "user", dbUser)

			Exec(db, fmt.Sprintf("ALTER DATABASE %s OWNER TO %s", dbName, dbUser))
			logger.Info("Gave ownership of database", "user", dbUser)
		},
	}
	return cmd
}

func teardown() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "teardown",
		Short: "Tears down the database and the database user used by the Greenlight app",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			db, err := pgx.Connect(ctx, superUserDSN)
			must(err)

			Exec(db, fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
			logger.Info("Dropped database", "name", dbName)

			Exec(db, fmt.Sprintf("DROP USER IF EXISTS %s", dbUser))
			logger.Info("Dropped user", "name", dbUser)
		},
	}
	return cmd
}

func must(err error) {
	if err != nil {
		logger.Error("An error occurred", "error", err.Error(), "trace", debug.Stack())
		os.Exit(1)
	}
}

func Exec(db *pgx.Conn, query string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := db.Exec(ctx, query)
	must(err)
}
