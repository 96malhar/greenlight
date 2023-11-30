package main

import (
	"math"
	"net/http"
	"testing"
)

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

type paginationMetadata struct {
	CurrentPage  int `json:"current_page"`
	PageSize     int `json:"page_size"`
	TotalRecords int `json:"total_records"`
	LastPage     int `json:"last_page"`
	FirstPage    int `json:"first_page"`
}

type listMovieResponse struct {
	Movies             []movie            `json:"movies"`
	PaginationMetadata paginationMetadata `json:"metadata"`
}

var notFoundResponse = map[string]string{
	"error": "the requested resource could not be found",
}

func newPaginationMetadata(currentPage, pageSize, totalRecords int) paginationMetadata {
	return paginationMetadata{
		CurrentPage:  currentPage,
		PageSize:     pageSize,
		TotalRecords: totalRecords,
		LastPage:     int(math.Ceil(float64(totalRecords) / float64(pageSize))),
		FirstPage:    1,
	}
}

func TestCreateMovieHandler(t *testing.T) {
	testcases := []handlerTestcase{
		{
			name:                   "Valid movie",
			requestUrlPath:         "/v1/movies",
			requestBody:            `{"title":"Die Hard","year":1988,"runtime":"207 mins","genres":["Action", "Thriller"]}`,
			wantResponseStatusCode: http.StatusCreated,
			wantResponseHeader: map[string]string{
				"Location": "/v1/movies/1",
			},
			wantResponse: movieResponse{
				Movie: movie{
					ID: 1, Title: "Die Hard", Year: 1988, Runtime: "207 mins",
					Genres: []string{"Action", "Thriller"}, Version: 1,
				},
			},
		},
		{
			name:                   "Empty year",
			requestUrlPath:         "/v1/movies",
			requestBody:            `{"title":"Die Hard","runtime":"207 mins","genres":["Action", "Thriller"]}`,
			wantResponseStatusCode: http.StatusUnprocessableEntity,
			wantResponse: validationErrorResponse{
				Error: map[string]string{
					"year": "must be provided",
				},
			},
		},
		{
			name:                   "Duplicate genre",
			requestUrlPath:         "/v1/movies",
			requestBody:            `{"title":"Die Hard","year":1988,"runtime":"207 mins","genres":["Action", "Action", "Thriller"]}`,
			wantResponseStatusCode: http.StatusUnprocessableEntity,
			wantResponse: validationErrorResponse{
				Error: map[string]string{
					"genres": "must not contain duplicate values",
				},
			},
		},
		{
			name:                   "Invalid runtime format",
			requestUrlPath:         "/v1/movies",
			requestBody:            `{"title":"Die Hard","year":1988,"runtime":207,"genres":["Action", "Thriller"]}`,
			wantResponseStatusCode: http.StatusBadRequest,
			wantResponse: map[string]string{
				"error": "invalid runtime format, example valid value 107 mins",
			},
		},
	}

	db := newTestDB(t)
	app := newTestApplication(db)
	authToken := generateAuthToken(t, app, db)

	for _, tc := range testcases {
		if tc.requestHeader == nil {
			tc.requestHeader = make(map[string]string)
		}
		tc.requestHeader["Authorization"] = "Bearer " + authToken
		tc.requestMethodType = http.MethodPost
		testHandler(t, app, tc)
	}
}

func TestShowMovieHandler(t *testing.T) {
	db := newTestDB(t)

	insertMovie(t, db, "Die Hard", "1988", 207, []string{"Action", "Thriller"})

	testcases := []handlerTestcase{
		{
			name:                   "Valid ID",
			requestUrlPath:         "/v1/movies/1",
			wantResponseStatusCode: http.StatusOK,
			wantResponse: movieResponse{
				Movie: movie{
					ID: 1, Title: "Die Hard", Year: 1988, Runtime: "207 mins",
					Genres: []string{"Action", "Thriller"}, Version: 1,
				},
			},
		},
		{
			name:                   "Resource not found",
			requestUrlPath:         "/v1/movies/2",
			wantResponseStatusCode: http.StatusNotFound,
			wantResponse:           notFoundResponse,
		},
		{
			name:                   "Invalid ID",
			requestUrlPath:         "/v1/movies/asad",
			wantResponseStatusCode: http.StatusNotFound,
			wantResponse:           notFoundResponse,
		},
	}

	app := newTestApplication(db)
	authToken := generateAuthToken(t, app, db)

	for _, tc := range testcases {
		if tc.requestHeader == nil {
			tc.requestHeader = make(map[string]string)
		}
		tc.requestHeader["Authorization"] = "Bearer " + authToken
		tc.requestMethodType = http.MethodGet
		testHandler(t, app, tc)
	}
}

func TestDeleteMovieHandler(t *testing.T) {
	db := newTestDB(t)

	insertMovie(t, db, "Die Hard", "1988", 207, []string{"Action", "Thriller"})
	insertMovie(t, db, "Titanic", "1997", 196, []string{"Romance"})

	testcases := []handlerTestcase{
		{
			name:                   "Valid ID",
			requestUrlPath:         "/v1/movies/1",
			wantResponseStatusCode: http.StatusOK,
			wantResponse: map[string]string{
				"message": "movie successfully deleted",
			},
		},
		{
			name:                   "Resource not found",
			requestUrlPath:         "/v1/movies/6",
			wantResponseStatusCode: http.StatusNotFound,
			wantResponse:           notFoundResponse,
		},
		{
			name:                   "Invalid ID",
			requestUrlPath:         "/v1/movies/1ds",
			wantResponseStatusCode: http.StatusNotFound,
			wantResponse:           notFoundResponse,
		},
	}

	app := newTestApplication(db)
	authToken := generateAuthToken(t, app, db)

	for _, tc := range testcases {
		if tc.requestHeader == nil {
			tc.requestHeader = make(map[string]string)
		}
		tc.requestHeader["Authorization"] = "Bearer " + authToken
		tc.requestMethodType = http.MethodDelete
		testHandler(t, app, tc)
	}
}

func TestUpdateMovieHandler(t *testing.T) {
	db := newTestDB(t)

	insertMovie(t, db, "Die Hard", "1988", 207, []string{"Action", "Thriller"})

	testcases := []handlerTestcase{
		{
			name:                   "Valid update",
			requestUrlPath:         "/v1/movies/1",
			requestBody:            `{"genres": ["Romance"], "year": 1997}`,
			wantResponseStatusCode: http.StatusOK,
			wantResponse: movieResponse{
				Movie: movie{
					ID: 1, Title: "Die Hard", Year: 1997, Runtime: "207 mins",
					Genres: []string{"Romance"}, Version: 2,
				},
			},
		},
		{
			name:                   "ID does not exist",
			requestUrlPath:         "/v1/movies/5",
			requestBody:            `{"genres": ["Romance"], "year": 1997}`,
			wantResponseStatusCode: http.StatusNotFound,
			wantResponse:           notFoundResponse,
		},
		{
			name:                   "Badly formed JSON request",
			requestUrlPath:         "/v1/movies/1",
			requestBody:            `{"genres": ["Romance"], "year"- 1997}`,
			wantResponseStatusCode: http.StatusBadRequest,
			wantResponse: map[string]string{
				"error": "body contains badly-formed JSON (at character 31)",
			},
		},
		{
			name:                   "Validation error",
			requestUrlPath:         "/v1/movies/1",
			requestBody:            `{"genres":["Romance"], "year":1997, "runtime":"207 mins", "title":""}`,
			wantResponseStatusCode: http.StatusUnprocessableEntity,
			wantResponse: validationErrorResponse{
				Error: map[string]string{
					"title": "must be provided",
				},
			},
		},
	}

	app := newTestApplication(db)
	authToken := generateAuthToken(t, app, db)

	for _, tc := range testcases {
		if tc.requestHeader == nil {
			tc.requestHeader = make(map[string]string)
		}
		tc.requestHeader["Authorization"] = "Bearer " + authToken
		tc.requestMethodType = http.MethodPatch
		testHandler(t, app, tc)
	}
}

func TestListMoviesHandler(t *testing.T) {
	db := newTestDB(t)

	// seed movies table with movies
	insertMovie(t, db, "Die Hard", "1988", 207, []string{"Action", "Thriller"})
	insertMovie(t, db, "Titanic", "1997", 167, []string{"Romance"})
	insertMovie(t, db, "Batman", "1989", 126, []string{"Action"})

	dieHard := movie{
		ID: 1, Title: "Die Hard", Year: 1988, Runtime: "207 mins",
		Genres: []string{"Action", "Thriller"}, Version: 1,
	}
	titanic := movie{
		ID: 2, Title: "Titanic", Year: 1997, Runtime: "167 mins",
		Genres: []string{"Romance"}, Version: 1,
	}
	batman := movie{
		ID: 3, Title: "Batman", Year: 1989, Runtime: "126 mins",
		Genres: []string{"Action"}, Version: 1,
	}

	testcases := []handlerTestcase{
		{
			name:                   "No query parameters",
			requestUrlPath:         "/v1/movies",
			wantResponseStatusCode: http.StatusOK,
			wantResponse: listMovieResponse{
				Movies:             []movie{dieHard, titanic, batman},
				PaginationMetadata: newPaginationMetadata(1, 20, 3),
			},
		},
		{
			name:                   "sort=-year",
			requestUrlPath:         "/v1/movies?sort=-year",
			wantResponseStatusCode: http.StatusOK,
			wantResponse: listMovieResponse{
				Movies:             []movie{titanic, batman, dieHard},
				PaginationMetadata: newPaginationMetadata(1, 20, 3),
			},
		},
		{
			name:                   "Invalid sort key",
			requestUrlPath:         "/v1/movies?sort=xyz",
			wantResponseStatusCode: http.StatusUnprocessableEntity,
			wantResponse: map[string]map[string]string{
				"error": {"sort": "invalid sort value"},
			},
		},
		{
			name:                   "genres=Action",
			requestUrlPath:         "/v1/movies?genres=Action",
			wantResponseStatusCode: http.StatusOK,
			wantResponse: listMovieResponse{
				Movies:             []movie{dieHard, batman},
				PaginationMetadata: newPaginationMetadata(1, 20, 2),
			},
		},
		{
			name:                   "genres=Action,Thriller",
			requestUrlPath:         "/v1/movies?genres=Action,Thriller",
			wantResponseStatusCode: http.StatusOK,
			wantResponse: listMovieResponse{
				Movies:             []movie{dieHard},
				PaginationMetadata: newPaginationMetadata(1, 20, 1),
			},
		},
		{
			name:                   "title=Titanic genres=Romance",
			requestUrlPath:         "/v1/movies?title=Titanic&genres=Romance",
			wantResponseStatusCode: http.StatusOK,
			wantResponse: listMovieResponse{
				Movies:             []movie{titanic},
				PaginationMetadata: newPaginationMetadata(1, 20, 1),
			},
		},
		{
			name:                   "title=die",
			requestUrlPath:         "/v1/movies?title=die",
			wantResponseStatusCode: http.StatusOK,
			wantResponse: listMovieResponse{
				Movies:             []movie{dieHard},
				PaginationMetadata: newPaginationMetadata(1, 20, 1),
			},
		},
		{
			name:                   "page=1&page_size=2",
			requestUrlPath:         "/v1/movies?page=1&page_size=2",
			wantResponseStatusCode: http.StatusOK,
			wantResponse: listMovieResponse{
				Movies:             []movie{dieHard, titanic},
				PaginationMetadata: newPaginationMetadata(1, 2, 3),
			},
		},
		{
			name:                   "page=2&page_size=2",
			requestUrlPath:         "/v1/movies?page=2&page_size=2",
			wantResponseStatusCode: http.StatusOK,
			wantResponse: listMovieResponse{
				Movies:             []movie{batman},
				PaginationMetadata: newPaginationMetadata(2, 2, 3),
			},
		},
		{
			name:                   "page=3&page_size=2",
			requestUrlPath:         "/v1/movies?page=3&page_size=2",
			wantResponseStatusCode: http.StatusOK,
			wantResponse: listMovieResponse{
				Movies:             []movie{},
				PaginationMetadata: paginationMetadata{},
			},
		},
		{
			name:                   "Empty movie list",
			requestUrlPath:         "/v1/movies?title=Spiderman",
			wantResponseStatusCode: http.StatusOK,
			wantResponse: listMovieResponse{
				Movies:             []movie{},
				PaginationMetadata: paginationMetadata{},
			},
		},
		{
			name:                   "Non-numeric page",
			requestUrlPath:         "/v1/movies?page=abc",
			wantResponseStatusCode: http.StatusUnprocessableEntity,
			wantResponse: map[string]map[string]string{
				"error": {"page": "must be an integer value"},
			},
		},
		{
			name:                   "Out-of-bounds page_size",
			requestUrlPath:         "/v1/movies?page_size=1000",
			wantResponseStatusCode: http.StatusUnprocessableEntity,
			wantResponse: map[string]map[string]string{
				"error": {"page_size": "must be a maximum of 100"},
			},
		},
	}

	app := newTestApplication(db)
	authToken := generateAuthToken(t, app, db)

	for _, tc := range testcases {
		if tc.requestHeader == nil {
			tc.requestHeader = make(map[string]string)
		}
		tc.requestHeader["Authorization"] = "Bearer " + authToken
		tc.requestMethodType = http.MethodGet
		testHandler(t, app, tc)
	}
}

func TestUnauthenticatedRequests(t *testing.T) {
	testcases := []handlerTestcase{
		{
			name:                   "Create movie",
			requestUrlPath:         "/v1/movies",
			requestMethodType:      http.MethodPost,
			wantResponseStatusCode: http.StatusUnauthorized,
			wantResponse: map[string]string{
				"error": "authentication required",
			},
		},
		{
			name:                   "Show movie",
			requestUrlPath:         "/v1/movies/1",
			requestMethodType:      http.MethodGet,
			wantResponseStatusCode: http.StatusUnauthorized,
			wantResponse: map[string]string{
				"error": "authentication required",
			},
		},
		{
			name:                   "Update movie",
			requestUrlPath:         "/v1/movies/1",
			requestMethodType:      http.MethodPatch,
			wantResponseStatusCode: http.StatusUnauthorized,
			wantResponse: map[string]string{
				"error": "authentication required",
			},
		},
		{
			name:                   "Delete movie",
			requestUrlPath:         "/v1/movies/1",
			requestMethodType:      http.MethodDelete,
			wantResponseStatusCode: http.StatusUnauthorized,
			wantResponse: map[string]string{
				"error": "authentication required",
			},
		},
		{
			name:                   "List movies",
			requestUrlPath:         "/v1/movies",
			requestMethodType:      http.MethodGet,
			wantResponseStatusCode: http.StatusUnauthorized,
			wantResponse: map[string]string{
				"error": "authentication required",
			},
		},
	}

	app := newTestApplication(nil)
	for _, tc := range testcases {
		tc.wantResponse = errorResponse{
			Error: "you must be authenticated to access this resource",
		}
		testHandler(t, app, tc)
	}
}
