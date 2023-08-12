package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/lib/pq"
	"io"
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

type healthCheckResponse struct {
	Status     string            `json:"status"`
	SystemInfo map[string]string `json:"system_info"`
}

type movie struct {
	ID      int      `json:"id"`
	Title   string   `json:"title"`
	Year    int      `json:"year"`
	Runtime string   `json:"runtime"`
	Genres  []string `json:"genres"`
	Version int      `json:"version"`
}

type movieResponse struct {
	Movie movie `json:"movie"`
}

type listMovieResponse struct {
	Movies []movie `json:"movies"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type validationErrorResponse struct {
	Error map[string]string `json:"error"`
}

type testServer struct {
	router http.Handler
}

func newTestServer(router http.Handler) *testServer {
	return &testServer{router}
}

func (ts *testServer) executeRequest(method, urlPath, body string, header http.Header) (*http.Response, error) {
	req, err := http.NewRequest(method, urlPath, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	if header != nil {
		req.Header = header
	}

	rr := httptest.NewRecorder()
	ts.router.ServeHTTP(rr, req)
	return rr.Result(), nil
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

func newTestApplication(db *sql.DB) *application {
	return &application{
		logger:     log.New(os.Stdout, "", log.Ldate|log.Ltime),
		config:     config{env: "development"},
		modelStore: data.NewModelStore(db),
	}
}

func readJsonResponse(t *testing.T, body io.Reader, dst any) {
	dec := json.NewDecoder(body)
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
