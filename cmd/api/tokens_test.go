package main

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
	"time"
)

type authenticationTokenResponse struct {
	AuthenticationToken struct {
		Token  string `json:"token"`
		Expiry string `json:"expiry"`
	} `json:"authentication_token"`
}

func TestCreateAuthenticationTokenHandler_ValidCredentials(t *testing.T) {
	ts := newTestServer(t)

	// Create a new user
	_, err := ts.executeRequest(http.MethodPost, "/v1/users", `{"name":"Bob", "email":"bob@gmail.com", "password":"5ecret1234"}`, nil)
	require.NoError(t, err)

	// Get authentication token
	res, err := ts.executeRequest(http.MethodPost, "/v1/tokens/authentication", `{"email":"bob@gmail.com", "password":"5ecret1234"}`, nil)

	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, res.StatusCode)
	defer res.Body.Close()

	dst := &authenticationTokenResponse{}
	readJsonResponse(t, res.Body, dst)
	assert.True(t, dst.AuthenticationToken.Token != "")

	expiry, err := time.Parse(time.RFC3339, dst.AuthenticationToken.Expiry)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().UTC().Add(24*time.Hour), expiry, 1*time.Second)
}

func TestCreateAuthenticationTokenHandler_InvalidCredentials(t *testing.T) {
	ts := newTestServer(t)

	// Create a new user
	_, err := ts.executeRequest(http.MethodPost, "/v1/users", `{"name":"Bob", "email":"bob@gmail.com", "password":"5ecret1234"}`, nil)
	require.NoError(t, err)

	testcases := []handlerTestcase{
		{
			name:                   "Bad request body",
			requestUrlPath:         "/v1/tokens/authentication",
			requestMethodType:      http.MethodPost,
			requestBody:            `{"emailAddress":"abc123", "password":"5ecret1234"}`,
			wantResponseStatusCode: http.StatusBadRequest,
			wantResponse: errorResponse{
				Error: "body contains unknown key \"emailAddress\"",
			},
		},
		{
			name:                   "Invalid email",
			requestUrlPath:         "/v1/tokens/authentication",
			requestMethodType:      http.MethodPost,
			requestBody:            `{"email":"abc123", "password":"5ecret1234"}`,
			wantResponseStatusCode: http.StatusUnprocessableEntity,
			wantResponse: validationErrorResponse{
				Error: map[string]string{
					"email": "must be a valid email address",
				},
			},
		},
		{
			name:                   "Invalid password",
			requestUrlPath:         "/v1/tokens/authentication",
			requestMethodType:      http.MethodPost,
			requestBody:            `{"email":"abc@gmail.com", "password":"pass"}`,
			wantResponseStatusCode: http.StatusUnprocessableEntity,
			wantResponse: validationErrorResponse{
				Error: map[string]string{
					"password": "must be at least 8 bytes long",
				},
			},
		},
		{
			name:                   "Wrong password",
			requestUrlPath:         "/v1/tokens/authentication",
			requestMethodType:      http.MethodPost,
			requestBody:            `{"email":"bob@gmail.com", "password":"pass12345"}`,
			wantResponseStatusCode: http.StatusUnauthorized,
			wantResponse: errorResponse{
				Error: "invalid authentication credentials",
			},
		},
		{
			name:                   "User does not exist",
			requestUrlPath:         "/v1/tokens/authentication",
			requestMethodType:      http.MethodPost,
			requestBody:            `{"email":"abc@gmail.com", "password":"pass12345"}`,
			wantResponseStatusCode: http.StatusUnauthorized,
			wantResponse: errorResponse{
				Error: "invalid authentication credentials",
			},
		},
	}

	testHandler(t, ts, testcases...)
}
