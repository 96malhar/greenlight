package main

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServerErrorResponse(t *testing.T) {
	app, rr, req := arrangeErrorTest()

	app.serverErrorResponse(rr, req, errors.New("unexpected database error"))

	result := rr.Result()
	var gotResponse errorResponse
	readJsonResponse(t, result.Body, &gotResponse)
	wantResponse := errorResponse{
		Error: "the server encountered a problem and could not process your request",
	}

	assert.Equal(t, http.StatusInternalServerError, result.StatusCode)
	assert.Equal(t, wantResponse, gotResponse)
}

func TestNotFoundResponse(t *testing.T) {
	app, rr, req := arrangeErrorTest()

	app.notFoundResponse(rr, req)

	result := rr.Result()
	var gotResponse errorResponse
	readJsonResponse(t, result.Body, &gotResponse)
	wantResponse := errorResponse{
		Error: "the requested resource could not be found",
	}

	assert.Equal(t, http.StatusNotFound, result.StatusCode)
	assert.Equal(t, wantResponse, gotResponse)
}

func TestBadRequestResponse(t *testing.T) {
	app, rr, req := arrangeErrorTest()

	app.badRequestResponse(rr, req, errors.New("this is an unauthorized request"))

	result := rr.Result()
	var gotResponse errorResponse
	readJsonResponse(t, result.Body, &gotResponse)
	wantResponse := errorResponse{
		Error: "this is an unauthorized request",
	}

	assert.Equal(t, http.StatusBadRequest, result.StatusCode)
	assert.Equal(t, wantResponse, gotResponse)
}

func TestMethodNotAllowedResponse(t *testing.T) {
	app, rr, req := arrangeErrorTest()

	app.methodNotAllowedResponse(rr, req)

	result := rr.Result()
	var gotResponse errorResponse
	readJsonResponse(t, result.Body, &gotResponse)
	wantResponse := errorResponse{
		Error: fmt.Sprintf("the %s method is not supported for this resource", req.Method),
	}

	assert.Equal(t, http.StatusMethodNotAllowed, result.StatusCode)
	assert.Equal(t, wantResponse, gotResponse)
}

func TestFailedValidationResponse(t *testing.T) {
	app, rr, req := arrangeErrorTest()
	validationErrors := map[string]string{
		"genres": "must be unique",
		"title":  "cannot be empty",
	}
	app.failedValidationResponse(rr, req, validationErrors)

	result := rr.Result()
	gotResponse := validationErrorResponse{}
	readJsonResponse(t, result.Body, &gotResponse)
	wantResponse := validationErrorResponse{
		Error: validationErrors,
	}

	assert.Equal(t, http.StatusUnprocessableEntity, result.StatusCode)
	assert.Equal(t, wantResponse, gotResponse)
}

func TestEditConflictResponse(t *testing.T) {
	app, rr, req := arrangeErrorTest()

	app.editConflictResponse(rr, req)

	result := rr.Result()
	var gotResponse errorResponse
	readJsonResponse(t, result.Body, &gotResponse)
	wantResponse := errorResponse{
		Error: "unable to update the record due to an edit conflict, please try again",
	}

	assert.Equal(t, http.StatusConflict, result.StatusCode)
	assert.Equal(t, wantResponse, gotResponse)
}

func TestRateLimitExceededResponse(t *testing.T) {
	app, rr, req := arrangeErrorTest()

	app.rateLimitExceededResponse(rr, req)

	result := rr.Result()
	var gotResponse errorResponse
	readJsonResponse(t, result.Body, &gotResponse)
	wantResponse := errorResponse{
		Error: "rate limit exceeded",
	}

	assert.Equal(t, http.StatusTooManyRequests, result.StatusCode)
	assert.Equal(t, wantResponse, gotResponse)
}

func TestInvalidCredentialsResponse(t *testing.T) {
	app, rr, req := arrangeErrorTest()

	app.invalidCredentialsResponse(rr, req)

	result := rr.Result()
	var gotResponse errorResponse
	readJsonResponse(t, result.Body, &gotResponse)
	wantResponse := errorResponse{
		Error: "invalid authentication credentials",
	}

	assert.Equal(t, http.StatusUnauthorized, result.StatusCode)
	assert.Equal(t, wantResponse, gotResponse)
}

func arrangeErrorTest() (*application, *httptest.ResponseRecorder, *http.Request) {
	app := newTestApplication(nil)
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)

	return app, rr, req
}
