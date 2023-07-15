package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/lib/pq"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/96malhar/greenlight/internal/data"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type testServer struct {
	*httptest.Server
}

func newTestServer(h http.Handler) *testServer {
	ts := httptest.NewServer(h)
	return &testServer{ts}
}

func (ts *testServer) do(method, urlPath, body string, header http.Header) (*http.Response, error) {
	req, err := http.NewRequest(method, ts.URL+urlPath, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	if header != nil {
		req.Header = header
	}
	return ts.Client().Do(req)
}

func newTestDB(t *testing.T) (*sql.DB, func()) {
	randomSuffix := strings.Split(uuid.New().String(), "-")[0]
	testDBName := fmt.Sprintf("greenlight_test_%s", randomSuffix)

	db := getDBConn(t, "postgres", "postgres", "postgres")
	_, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s", testDBName))
	require.NoErrorf(t, err, "Failed to create database %s", testDBName)
	db.Close()

	db = getDBConn(t, "postgres", "postgres", testDBName)
	t.Logf("Connected to database %s", testDBName)
	migrator := runMigrations(t, db)

	cleanup := func() {
		db.Close()
		migrator.Close()
		dropDB(t, testDBName)
	}

	return db, cleanup
}

func getDBConn(t *testing.T, user, password, dbname string) *sql.DB {
	dsn := fmt.Sprintf("host=localhost port=5432 user=%s password=%s sslmode=disable dbname=%s", user, password, dbname)
	db, _ := sql.Open("postgres", dsn)
	err := db.Ping()
	require.NoErrorf(t, err, "Failed to connect to postgres with DSN = %s", dsn)

	return db
}

func dropDB(t *testing.T, dbName string) {
	db := getDBConn(t, "postgres", "postgres", "postgres")
	defer db.Close()
	_, err := db.Exec(fmt.Sprintf("DROP DATABASE %s", dbName))
	require.NoErrorf(t, err, "Failed to drop database %s", dbName)
	t.Logf("Dropped database %s", dbName)
}

func newTestApplication(t *testing.T, db *sql.DB) *application {
	return &application{
		logger:     log.New(os.Stdout, "", log.Ldate|log.Ltime),
		config:     config{env: "development"},
		modelStore: data.NewModelStore(db),
	}
}

func readJsonResponse(t *testing.T, res *http.Response, dst any) {
	dec := json.NewDecoder(res.Body)
	err := dec.Decode(dst)
	require.NoError(t, err)
}

func runMigrations(t *testing.T, db *sql.DB) *migrate.Migrate {
	migrationDriver, err := postgres.WithInstance(db, &postgres.Config{})
	require.NoError(t, err)

	migrator, err := migrate.NewWithDatabaseInstance("file://../../migrations", "postgres", migrationDriver)
	require.NoError(t, err)

	err = migrator.Up()
	require.NoError(t, err)

	t.Log("database migrations applied")
	return migrator
}

func insertMovie(t *testing.T, db *sql.DB, title string, year string, runtime int, genres []string) {
	query := `
        INSERT INTO movies (title, year, runtime, genres) 
        VALUES ($1, $2, $3, $4)`

	_, err := db.Exec(query, title, year, runtime, pq.Array(genres))
	require.NoError(t, err, "Failed to insert movie in the database")
}
