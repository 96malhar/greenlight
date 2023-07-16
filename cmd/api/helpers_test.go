package main

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReadJSON_Successful(t *testing.T) {
	jsonBody := `{
		   "id":1,
		   "title":"Die Hard",
		   "year":1988,
		   "runtime":"207 mins",
		   "genres":[
			  "Action",
			  "Thriller"
		   ],
		   "version":1
		}`

	req, _ := http.NewRequest(http.MethodGet, "/", strings.NewReader(jsonBody))
	rr := httptest.NewRecorder()

	var dst movie
	want := movie{
		ID: 1, Title: "Die Hard", Year: 1988, Runtime: "207 mins",
		Genres: []string{"Action", "Thriller"}, Version: 1,
	}
	app := newTestApplication(nil)
	app.readJSON(rr, req, &dst)

	assert.Equal(t, want, dst)
}

func TestReadJSON_Unsuccessful(t *testing.T) {
	testcases := []struct {
		name     string
		jsonBody string
		dst      any
		errMsg   string
	}{
		{
			name:     "Badly formed JSON",
			jsonBody: `{"genres": ["Romance"], "year"- 1997}`,
			dst:      &movie{},
			errMsg:   "body contains badly-formed JSON (at character 31)",
		},
		{
			name:     "Unknown key",
			jsonBody: `{"genres": ["Romance"], "releaseYear": 1997}`,
			dst:      &movie{},
			errMsg:   "body contains unknown key \"releaseYear\"",
		},
		{
			name:     "Incorrect JSON type for field",
			jsonBody: `{"genres": ["Romance"], "year": "abcded"}`,
			dst:      &movie{},
			errMsg:   "body contains incorrect JSON type for field \"year\"",
		},
		{
			name:     "Empty JSON body",
			jsonBody: "",
			dst:      &movie{},
			errMsg:   "body must not be empty",
		},
		{
			name:     "Multiple JSON",
			jsonBody: `{"genres": ["Romance"]}{"year": 1997}`,
			dst:      &movie{},
			errMsg:   "body must only contain a single JSON value",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			app := newTestApplication(nil)
			rr := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/", strings.NewReader(tc.jsonBody))
			err := app.readJSON(rr, req, tc.dst)

			assert.EqualError(t, err, tc.errMsg)
		})
	}
}

func TestWriteJSON(t *testing.T) {
	m := movie{
		ID: 1, Title: "Die Hard", Year: 1988, Runtime: "207 mins",
		Genres: []string{"Action", "Thriller"}, Version: 1,
	}

	app := newTestApplication(nil)
	headers := make(http.Header)
	headers.Set("Cache-Control", "No-Cache")
	rr := httptest.NewRecorder()

	err := app.writeJSON(rr, http.StatusCreated, envelope{"movie": m}, headers)
	assert.NoError(t, err)

	res := rr.Result()
	defer res.Body.Close()
	resBody, err := io.ReadAll(res.Body)
	resBody = bytes.TrimSpace(resBody)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, res.StatusCode)

	wantBody := `{"movie":{"id":1,"title":"Die Hard","year":1988,"runtime":"207 mins","genres":["Action","Thriller"],"version":1}}`
	assert.Equal(t, wantBody, string(resBody))
	assert.Equal(t, "No-Cache", res.Header.Get("Cache-Control"))
}
