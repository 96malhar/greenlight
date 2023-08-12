package main

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"reflect"
	"testing"
)

type handlerTestcase struct {
	name                   string
	requestMethodType      string
	requestUrlPath         string
	requestBody            string
	requestHeader          http.Header
	wantResponseStatusCode int
	wantResponse           any
	wantResponseHeader     map[string]string
}

func testHandler(t *testing.T, app *application, testcases ...handlerTestcase) {
	ts := newTestServer(app.routes())

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.wantResponse != nil && reflect.TypeOf(tc.wantResponse).Kind() == reflect.Ptr {
				t.Fatal("wantResponseHeader should not be of pointer type")
			}

			res, err := ts.executeRequest(tc.requestMethodType, tc.requestUrlPath, tc.requestBody, tc.requestHeader)
			require.NoError(t, err)

			assert.Equal(t, tc.wantResponseStatusCode, res.StatusCode, "response status codes do not match")

			if tc.wantResponse != nil {
				// reflect.New() returns a pointer to the type
				gotResponse := reflect.New(reflect.TypeOf(tc.wantResponse)).Interface()
				readJsonResponse(t, res.Body, gotResponse)

				// deference the pointer to get the value
				gotResponse = reflect.ValueOf(gotResponse).Elem().Interface()
				assert.Equal(t, tc.wantResponse, gotResponse)
			}

			for key, wantVal := range tc.wantResponseHeader {
				gotVal := res.Header.Get(key)
				assert.Equalf(t, wantVal, gotVal, "Header values do not match for %s", key)
			}
		})
	}
}
