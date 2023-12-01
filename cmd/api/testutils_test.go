package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
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
	app    *application
	db     *sql.DB
}

type dummyUser struct {
	name          string
	email         string
	password      string
	activated     bool
	authenticated bool
	authTTL       time.Duration
	permCodes     []string
}

func newTestServer(t *testing.T) *testServer {
	testDb := newTestDB(t)
	app := &application{
		logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		config:     config{env: "development"},
		modelStore: data.NewModelStore(testDb),
	}

	return &testServer{
		router: app.routes(),
		app:    app,
		db:     testDb,
	}
}

func (ts *testServer) executeRequest(method, urlPath, body string, requestHeader map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, urlPath, strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	// convert requestHeader map to http.Header
	header := http.Header{}
	for key, val := range requestHeader {
		header.Add(key, val)
	}
	req.Header = header

	rr := httptest.NewRecorder()
	ts.router.ServeHTTP(rr, req)
	return rr.Result(), nil
}

func (ts *testServer) insertMovie(t *testing.T, title string, year int, runtime data.Runtime, genres []string) {
	m := &data.Movie{
		Title:   title,
		Year:    int32(year),
		Runtime: runtime,
		Genres:  genres,
		Version: 1,
	}
	err := ts.app.modelStore.Movies.Insert(m)
	require.NoError(t, err, "Failed to insert movie in the database")
}

func (ts *testServer) insertUser(t *testing.T, usr dummyUser) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(usr.password), 12)
	require.NoError(t, err, "Failed to generate password hash")

	query := `
		INSERT INTO users (name, email, created_at, password_hash, activated)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`

	var id int64
	err = ts.db.QueryRow(query, usr.name, usr.email, time.Now().UTC(), hash, usr.activated).Scan(&id)
	require.NoError(t, err, "Failed to insert user in the database")

	if !usr.authenticated {
		return ""
	}

	if usr.authTTL == 0 {
		usr.authTTL = 24 * time.Hour
	}
	u, err := ts.app.modelStore.Users.GetByEmail(usr.email)
	require.NoError(t, err, "Failed to fetch user from the database")
	authToken, err := ts.app.modelStore.Tokens.New(u.ID, usr.authTTL, data.ScopeAuthentication)
	require.NoError(t, err, "Failed to create auth token")

	err = ts.app.modelStore.Permissions.AddForUser(id, usr.permCodes...)
	require.NoError(t, err, "Failed to add permissions for user")

	return authToken.Plaintext
}

func newTestDB(t *testing.T) *sql.DB {
	randomSuffix := strings.Split(uuid.New().String(), "-")[0]
	testDbName := fmt.Sprintf("greenlight_test_%s", randomSuffix)
	rootDsn := "host=localhost port=5432 user=postgres password=postgres sslmode=disable"
	testDbDsn := fmt.Sprintf("%s dbname=%s", rootDsn, testDbName)

	rootDb := getDbConn(t, rootDsn)
	_, err := rootDb.Exec(fmt.Sprintf("CREATE DATABASE %s", testDbName))
	require.NoErrorf(t, err, "Failed to create database %s", testDbName)

	testDb := getDbConn(t, testDbDsn)
	t.Logf("Connected to database %s", testDbName)
	migrator := runMigrations(t, testDb, testDbName)

	t.Cleanup(func() {
		testDb.Close()
		migrator.Close()

		_, err := rootDb.Exec(fmt.Sprintf("DROP DATABASE %s", testDbName))
		require.NoErrorf(t, err, "Failed to drop database %s", testDbName)
		t.Logf("Dropped database %s", testDbName)
		rootDb.Close()
	})

	return testDb
}

func getDbConn(t *testing.T, dsn string) *sql.DB {
	db, _ := sql.Open("postgres", dsn)
	err := db.Ping()
	require.NoErrorf(t, err, "Failed to connect to postgres with DSN = %s", dsn)

	return db
}

func newTestApplication(db *sql.DB) *application {
	return &application{
		logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		config:     config{env: "development"},
		modelStore: data.NewModelStore(db),
	}
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

func readJsonResponse(t *testing.T, body io.Reader, dst any) {
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()
	err := dec.Decode(dst)
	require.NoError(t, err)
}
