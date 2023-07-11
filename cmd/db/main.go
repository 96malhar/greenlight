package main

import (
	"database/sql"
	"fmt"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
	"log"
	"os"
	"runtime/debug"
)

var dbUser, dbName, dbPassword string
var errorLog = log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)
var infoLog = log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
var superUserDSN = "postgresql://postgres:postgres@localhost:5432/?sslmode=disable"

func init() {
	must(godotenv.Load())
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
			infoLog.Printf("Created database %s", dbName)

			Exec(db, fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s'", dbUser, dbPassword))
			infoLog.Printf("Created user %s", dbUser)

			Exec(db, "CREATE EXTENSION IF NOT EXISTS citext")
			infoLog.Printf("Created extension citext")
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

			Exec(db, fmt.Sprintf("DROP USER IF EXISTS %s", dbUser))
			infoLog.Printf("Dropped user %s", dbUser)

			Exec(db, fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
			infoLog.Printf("Dropped database %s", dbName)
		},
	}
	return cmd
}

func must(err error) {
	if err != nil {
		trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
		errorLog.Output(2, trace)
		os.Exit(1)
	}
}

func Exec(db *sql.DB, query string) {
	_, err := db.Exec(query)
	must(err)
}
