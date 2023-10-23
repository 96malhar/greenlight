package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/96malhar/greenlight/internal/data"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

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

func newTestDB(t *testing.T) *sql.DB {
	randomSuffix := strings.Split(uuid.New().String(), "-")[0]
	testDBName := fmt.Sprintf("greenlight_test_%s", randomSuffix)

	db := getRootDbConn(t)
	_, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s", testDBName))
	require.NoErrorf(t, err, "Failed to create database %s", testDBName)
	db.Close()

	db = getDbConn(t, testDBName)
	t.Logf("Connected to database %s", testDBName)
	migrator := runMigrations(t, db, testDBName)

	t.Cleanup(func() {
		db.Close()
		migrator.Close()
		dropDB(t, testDBName)
	})

	return db
}

func getDbConn(t *testing.T, dbname string) *sql.DB {
	dsn := fmt.Sprintf("host=localhost port=5432 user=postgres password=postgres sslmode=disable dbname=%s", dbname)
	db, _ := sql.Open("postgres", dsn)
	err := db.Ping()
	require.NoErrorf(t, err, "Failed to connect to postgres with DSN = %s", dsn)

	return db
}

func getRootDbConn(t *testing.T) *sql.DB {
	dsn := "host=localhost port=5432 user=postgres password=postgres sslmode=disable"
	db, _ := sql.Open("postgres", dsn)
	err := db.Ping()
	require.NoErrorf(t, err, "Failed to connect to postgres with DSN = %s", dsn)

	return db
}

func dropDB(t *testing.T, dbName string) {
	db := getRootDbConn(t)
	defer db.Close()
	_, err := db.Exec(fmt.Sprintf("DROP DATABASE %s", dbName))
	require.NoErrorf(t, err, "Failed to drop database %s", dbName)
	t.Logf("Dropped database %s", dbName)
}

func newTestApplication(db *sql.DB) *application {
	return &application{
		logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		config:     config{env: "development"},
		modelStore: data.NewModelStore(db),
	}
}

func readJsonResponse(t *testing.T, body io.Reader, dst any) {
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()
	err := dec.Decode(dst)
	require.NoError(t, err)
}

func runMigrations(t *testing.T, db *sql.DB, dbName string) *migrate.Migrate {
	migrationDriver, err := postgres.WithInstance(db, &postgres.Config{})
	require.NoError(t, err)

	migrator, err := migrate.NewWithDatabaseInstance("file://../../migrations", dbName, migrationDriver)
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

func insertUser(t *testing.T, db *sql.DB, name, email, plaintextPassword string, createdAt time.Time) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), 12)
	require.NoError(t, err)

	query := `
		INSERT INTO users (name, email, created_at, password_hash, activated)
		VALUES ($1, $2, $3, $4, $5)`

	_, err = db.Exec(query, name, email, createdAt, hash, false)
	require.NoError(t, err, "Failed to insert user in the database")
}

// Mocks the mailer interface
type mockMailer struct {
	SendInvoked    bool
	TokenPlainText string
}

func (m *mockMailer) Send(recipient, templateFile string, data any) error {
	m.SendInvoked = true
	d := data.(map[string]any)
	m.TokenPlainText = d["activationToken"].(string)
	return nil
}
