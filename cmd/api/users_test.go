package main

import (
	"net/http"
	"testing"
	"time"
)

func TestRegisterUserHandler(t *testing.T) {
	db, dbCleanup := newTestDB(t)
	t.Cleanup(func() {
		dbCleanup()
	})

	currTime := time.Date(2021, time.January, 5, 15, 42, 58, 0, time.UTC)

	// seed users table with a user
	insertUser(t, db, "Alice", "alice@gmail.com", "pa55word1234", currTime)

	testcases := []handlerTestcase{
		{
			name:                   "Valid user",
			requestUrlPath:         "/v1/users",
			requestMethodType:      http.MethodPost,
			requestBody:            `{"name":"Bob", "email":"bob@gmail.com", "password":"5ecret1234"}`,
			wantResponseStatusCode: http.StatusCreated,
			wantResponse: userResponse{
				User: user{
					ID: 2, Name: "Bob",
					Email: "bob@gmail.com", Activated: false,
					CreatedAt: currTime,
				},
			},
		},
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

	app := newTestApplication(db)
	app.utcNow = func() time.Time {
		return currTime
	}
	testHandler(t, app, testcases...)
}
