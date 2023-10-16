package main

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
	"time"
)

func TestRegisterUserHandler_ValidRequest(t *testing.T) {
	db := newTestDB(t)

	currTime := time.Date(2021, time.January, 5, 15, 42, 58, 0, time.UTC)

	// seed users table with a user
	insertUser(t, db, "Alice", "alice@gmail.com", "pa55word1234", currTime)

	app := newTestApplication(db)
	mailer := &mockMailer{}
	app.mailer = mailer
	app.utcNow = func() time.Time {
		return currTime
	}

	checkEmailSent := func(t *testing.T) {
		// wait for the user to get the activation email
		time.Sleep(200 * time.Millisecond)
		assert.True(t, mailer.SendInvoked)
		assert.True(t, mailer.TokenPlainText != "")
	}

	testcase := handlerTestcase{
		name:                   "Valid user",
		requestUrlPath:         "/v1/users",
		requestMethodType:      http.MethodPost,
		requestBody:            `{"name":"Bob", "email":"bob@gmail.com", "password":"5ecret1234"}`,
		wantResponseStatusCode: http.StatusAccepted,
		wantResponse: userResponse{
			User: user{
				ID: 2, Name: "Bob",
				Email: "bob@gmail.com", Activated: false,
				CreatedAt: currTime,
			},
		},
		additionalChecks: []func(t *testing.T){
			checkEmailSent,
		},
	}

	testHandler(t, app, testcase)
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
	db := newTestDB(t)
	currTime := time.Date(2021, time.January, 5, 15, 42, 58, 0, time.UTC)
	app := newTestApplication(db)
	app.utcNow = func() time.Time {
		return currTime
	}
	mailer := &mockMailer{}
	app.mailer = mailer

	ts := newTestServer(app.routes())

	// Register a user
	res, err := ts.executeRequest(http.MethodPost, "/v1/users", `{"name":"Bob", "email":"bob@gmail.com", "password":"5ecret1234"}`, nil)

	require.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, res.StatusCode)

	// wait for the user to get the activation email
	time.Sleep(200 * time.Millisecond)

	// activate the above user
	activationToken := mailer.TokenPlainText
	testcase := handlerTestcase{
		name:                   "Valid token",
		requestUrlPath:         "/v1/users/activated",
		requestMethodType:      http.MethodPut,
		requestBody:            `{"token":"` + activationToken + `"}`,
		wantResponseStatusCode: http.StatusOK,
		wantResponse: userResponse{
			User: user{
				ID: 1, Name: "Bob",
				Email: "bob@gmail.com", Activated: true,
				CreatedAt: currTime,
			},
		},
	}

	testHandler(t, app, testcase)
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
