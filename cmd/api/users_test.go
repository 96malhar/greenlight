package main

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
	"time"
)

type user struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Activated bool      `json:"activated"`
}

type userResponse struct {
	User user `json:"user"`
}

func TestRegisterUserHandler_ValidRequest(t *testing.T) {
	db := newTestDB(t)
	mailer := &mockMailer{}
	app := newTestApplication(db)
	app.mailer = mailer
	ts := newTestServer(app.routes())

	res, err := ts.executeRequest(http.MethodPost, "/v1/users", `{"name":"Bob", "email":"bob@gmail.com", "password":"5ecret1234"}`, nil)
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, http.StatusAccepted, res.StatusCode)

	dst := &userResponse{}
	readJsonResponse(t, res.Body, dst)
	assert.Equal(t, int64(1), dst.User.ID)
	assert.Equal(t, "Bob", dst.User.Name)
	assert.Equal(t, "bob@gmail.com", dst.User.Email)
	assert.False(t, dst.User.Activated)
	assert.WithinDuration(t, time.Now().UTC(), dst.User.CreatedAt, 1*time.Second)

	// wait for the user to get the activation email
	time.Sleep(200 * time.Millisecond)
	assert.True(t, mailer.SendInvoked)
	assert.True(t, mailer.TokenPlainText != "")
}

func TestRegisterUserHandler_InvalidRequest(t *testing.T) {
	db := newTestDB(t)

	// seed users table with a user
	insertUser(t, db, "Alice", "alice@gmail.com", "pa55word1234", time.Now().UTC())

	app := newTestApplication(db)

	testcases := []handlerTestcase{
		{
			name:                   "Duplicate email",
			requestUrlPath:         "/v1/users",
			requestMethodType:      http.MethodPost,
			requestBody:            `{"name":"Bob", "email":"alice@gmail.com", "password":"5ecret1234"}`,
			wantResponseStatusCode: http.StatusUnprocessableEntity,
			wantResponse: validationErrorResponse{
				Error: map[string]string{
					"email": "a user with this email address already exists",
				},
			},
		},
		{
			name:                   "Invalid user",
			requestUrlPath:         "/v1/users",
			requestMethodType:      http.MethodPost,
			requestBody:            `{"email":"bob@.com", "password":"secret"}`,
			wantResponseStatusCode: http.StatusUnprocessableEntity,
			wantResponse: validationErrorResponse{
				Error: map[string]string{
					"email":    "must be a valid email address",
					"password": "must be at least 8 bytes long",
					"name":     "must be provided",
				},
			},
		},
		{
			name:                   "Bad request body",
			requestUrlPath:         "/v1/users",
			requestMethodType:      http.MethodPost,
			requestBody:            `{"names":"Bob", "email":"bob@.com", "password":"secret"}`,
			wantResponseStatusCode: http.StatusBadRequest,
			wantResponse: errorResponse{
				Error: "body contains unknown key \"names\"",
			},
		},
	}

	testHandler(t, app, testcases...)
}

func TestActivateUserHandler_ValidRequest(t *testing.T) {
	app := newTestApplication(newTestDB(t))
	mailer := &mockMailer{}
	app.mailer = mailer

	ts := newTestServer(app.routes())

	// Register a user
	_, err := ts.executeRequest(http.MethodPost, "/v1/users", `{"name":"Bob", "email":"bob@gmail.com", "password":"5ecret1234"}`, nil)

	require.NoError(t, err)

	// wait for the user to get the activation email
	time.Sleep(200 * time.Millisecond)

	// activate the above user
	res, err := ts.executeRequest(http.MethodPut, "/v1/users/activated", `{"token":"`+mailer.TokenPlainText+`"}`, nil)
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)

	dst := &userResponse{}
	readJsonResponse(t, res.Body, dst)
	assert.Equal(t, int64(1), dst.User.ID)
	assert.Equal(t, "Bob", dst.User.Name)
	assert.Equal(t, "bob@gmail.com", dst.User.Email)
	assert.True(t, dst.User.Activated)
	assert.WithinDuration(t, time.Now().UTC(), dst.User.CreatedAt, 1*time.Second)
}

func TestActivateUserHandler_InvalidRequest(t *testing.T) {
	testcases := []handlerTestcase{
		{
			name:                   "Token less that 26 bytes",
			requestUrlPath:         "/v1/users/activated",
			requestMethodType:      http.MethodPut,
			requestBody:            `{"token":"invalid-token"}`,
			wantResponseStatusCode: http.StatusUnprocessableEntity,
			wantResponse: validationErrorResponse{
				Error: map[string]string{
					"token": "must be 26 bytes long",
				},
			},
		},
		{
			name:                   "Token does not exist",
			requestUrlPath:         "/v1/users/activated",
			requestMethodType:      http.MethodPut,
			requestBody:            `{"token":"H2NMasdasdasnfjadhskjlfjhs"}`,
			wantResponseStatusCode: http.StatusUnprocessableEntity,
			wantResponse: validationErrorResponse{
				Error: map[string]string{
					"token": "invalid or expired activation token",
				},
			},
		},
	}

	app := newTestApplication(newTestDB(t))
	testHandler(t, app, testcases...)
}
