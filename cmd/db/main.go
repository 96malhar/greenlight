package main

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
	"log/slog"
	"os"
	"runtime/debug"
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
			db, err := sql.Open("postgres", superUserDSN)
			must(err)
			Exec(db, fmt.Sprintf("CREATE DATABASE %s", dbName))
			logger.Info("Created database", "name", dbName)

			Exec(db, fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s'", dbUser, dbPassword))
			logger.Info("Created database user", "user", dbUser)

			Exec(db, fmt.Sprintf("ALTER DATABASE %s OWNER TO %s", dbName, dbUser))
			logger.Info("Gave ownership of database", "user", dbUser)

			Exec(db, "CREATE EXTENSION IF NOT EXISTS citext")
			logger.Info("Created extension citext")
		},
	}
	return cmd
}

func teardown() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "teardown",
		Short: "Tears down the database and the database user used by the Greenlight app",
		Run: func(cmd *cobra.Command, args []string) {
			db, err := sql.Open("postgres", superUserDSN)
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

func Exec(db *sql.DB, query string) {
	_, err := db.Exec(query)
	must(err)
}
