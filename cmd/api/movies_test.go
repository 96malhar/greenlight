package main

import (
	"net/http"
	"testing"
)

func TestCreateMovieHandler(t *testing.T) {
	db, dbCleanup := newTestDB(t)
	t.Cleanup(func() {
		dbCleanup()
	})

	testcases := []handlerTestcase{
		{
			name:                   "Valid movie",
			requestUrlPath:         "/v1/movies",
			requestMethodType:      http.MethodPost,
			requestBody:            `{"title":"Die Hard","year":1988,"runtime":"207 mins","genres":["Action", "Thriller"]}`,
			responseDst:            &movieResponse{},
			wantResponseStatusCode: http.StatusCreated,
			wantResponseHeader: map[string]string{
				"Content-Type": "application/json",
				"Location":     "/v1/movies/1",
			},
			wantResponse: &movieResponse{
				Movie: movie{
					ID: 1, Title: "Die Hard", Year: 1988, Runtime: "207 mins",
					Genres: []string{"Action", "Thriller"}, Version: 1,
				},
			},
		},
		{
			name:                   "Empty year",
			requestUrlPath:         "/v1/movies",
			requestMethodType:      http.MethodPost,
			requestBody:            `{"title":"Die Hard","runtime":"207 mins","genres":["Action", "Thriller"]}`,
			responseDst:            &validationErrorResponse{},
			wantResponseStatusCode: http.StatusUnprocessableEntity,
			wantResponse: &validationErrorResponse{
				Error: map[string]string{
					"year": "must be provided",
				},
			},
		},
		{
			name:                   "Duplicate genre",
			requestUrlPath:         "/v1/movies",
			requestMethodType:      http.MethodPost,
			requestBody:            `{"title":"Die Hard","year":1988,"runtime":"207 mins","genres":["Action", "Action", "Thriller"]}`,
			responseDst:            &validationErrorResponse{},
			wantResponseStatusCode: http.StatusUnprocessableEntity,
			wantResponse: &validationErrorResponse{
				Error: map[string]string{
					"genres": "must not contain duplicate values",
				},
			},
		},
		{
			name:                   "Invalid runtime format",
			requestUrlPath:         "/v1/movies",
			requestMethodType:      http.MethodPost,
			requestBody:            `{"title":"Die Hard","year":1988,"runtime":207,"genres":["Action", "Thriller"]}`,
			responseDst:            &map[string]string{},
			wantResponseStatusCode: http.StatusBadRequest,
			wantResponse: &map[string]string{
				"error": "invalid runtime format, example valid value 107 mins",
			},
		},
	}

	testHandler(t, newTestApplication(db), testcases...)
}

func TestShowMovieHandler(t *testing.T) {
	db, dbCleanup := newTestDB(t)
	t.Cleanup(func() {
		dbCleanup()
	})

	insertMovie(t, db, "Die Hard", "1988", 207, []string{"Action", "Thriller"})

	testcases := []handlerTestcase{
		{
			name:                   "Valid ID",
			requestUrlPath:         "/v1/movies/1",
			requestMethodType:      http.MethodGet,
			responseDst:            &movieResponse{},
			wantResponseStatusCode: http.StatusOK,
			wantResponse: &movieResponse{
				Movie: movie{
					ID: 1, Title: "Die Hard", Year: 1988, Runtime: "207 mins",
					Genres: []string{"Action", "Thriller"}, Version: 1,
				},
			},
			wantResponseHeader: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			name:                   "Resource not found",
			requestUrlPath:         "/v1/movies/2",
			requestMethodType:      http.MethodGet,
			responseDst:            &map[string]string{},
			wantResponseStatusCode: http.StatusNotFound,
			wantResponse: &map[string]string{
				"error": "the requested resource could not be found",
			},
			wantResponseHeader: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			name:                   "Invalid ID",
			requestUrlPath:         "/v1/movies/asad",
			requestMethodType:      http.MethodGet,
			responseDst:            &map[string]string{},
			wantResponseStatusCode: http.StatusNotFound,
			wantResponse: &map[string]string{
				"error": "the requested resource could not be found",
			},
			wantResponseHeader: map[string]string{
				"Content-Type": "application/json",
			},
		},
	}

	testHandler(t, newTestApplication(db), testcases...)
}

func TestDeleteMovieHandler(t *testing.T) {
	db, dbCleanup := newTestDB(t)
	t.Cleanup(func() {
		dbCleanup()
	})

	insertMovie(t, db, "Die Hard", "1988", 207, []string{"Action", "Thriller"})
	insertMovie(t, db, "Titanic", "1997", 196, []string{"Romance"})

	testcases := []handlerTestcase{
		{
			name:                   "Valid ID",
			requestUrlPath:         "/v1/movies/1",
			requestMethodType:      http.MethodDelete,
			responseDst:            &map[string]string{},
			wantResponseStatusCode: http.StatusOK,
			wantResponse: &map[string]string{
				"message": "movie successfully deleted",
			},
		},
		{
			name:                   "Resource not found",
			requestUrlPath:         "/v1/movies/6",
			requestMethodType:      http.MethodDelete,
			responseDst:            &map[string]string{},
			wantResponseStatusCode: http.StatusNotFound,
			wantResponse: &map[string]string{
				"error": "the requested resource could not be found",
			},
		},
		{
			name:                   "Invalid ID",
			requestUrlPath:         "/v1/movies/1ds",
			requestMethodType:      http.MethodDelete,
			responseDst:            &map[string]string{},
			wantResponseStatusCode: http.StatusNotFound,
			wantResponse: &map[string]string{
				"error": "the requested resource could not be found",
			},
		},
	}

	testHandler(t, newTestApplication(db), testcases...)
}

func TestUpdateMovieHandler(t *testing.T) {
	db, dbCleanup := newTestDB(t)
	t.Cleanup(func() {
		dbCleanup()
	})

	insertMovie(t, db, "Die Hard", "1988", 207, []string{"Action", "Thriller"})

	testcases := []handlerTestcase{
		{
			name:                   "Valid update",
			requestUrlPath:         "/v1/movies/1",
			requestMethodType:      http.MethodPatch,
			requestBody:            `{"genres": ["Romance"], "year": 1997}`,
			responseDst:            &movieResponse{},
			wantResponseStatusCode: http.StatusOK,
			wantResponse: &movieResponse{
				Movie: movie{
					ID: 1, Title: "Die Hard", Year: 1997, Runtime: "207 mins",
					Genres: []string{"Romance"}, Version: 2,
				},
			},
			wantResponseHeader: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			name:                   "ID does not exist",
			requestUrlPath:         "/v1/movies/5",
			requestMethodType:      http.MethodPatch,
			requestBody:            `{"genres": ["Romance"], "year": 1997}`,
			responseDst:            &map[string]string{},
			wantResponseStatusCode: http.StatusNotFound,
			wantResponse: &map[string]string{
				"error": "the requested resource could not be found",
			},
		},
		{
			name:                   "Badly formed JSON request",
			requestUrlPath:         "/v1/movies/1",
			requestMethodType:      http.MethodPatch,
			requestBody:            `{"genres": ["Romance"], "year"- 1997}`,
			responseDst:            &map[string]string{},
			wantResponseStatusCode: http.StatusBadRequest,
			wantResponse: &map[string]string{
				"error": "body contains badly-formed JSON (at character 31)",
			},
		},
		{
			name:                   "Validation error",
			requestUrlPath:         "/v1/movies/1",
			requestMethodType:      http.MethodPatch,
			requestBody:            `{"genres":["Romance"], "year":1997, "runtime":"207 mins", "title":""}`,
			responseDst:            &validationErrorResponse{},
			wantResponseStatusCode: http.StatusUnprocessableEntity,
			wantResponse: &validationErrorResponse{
				Error: map[string]string{
					"title": "must be provided",
				},
			},
		},
	}

	testHandler(t, newTestApplication(db), testcases...)
}