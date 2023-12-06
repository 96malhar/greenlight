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
	requestHeader          map[string]string
	wantResponseStatusCode int
	wantResponse           any
	wantResponseHeader     map[string]string
	additionalChecks       func(t *testing.T, res *http.Response)
}

func testHandler(t *testing.T, ts *testServer, testcases ...handlerTestcase) {
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.wantResponse != nil && reflect.TypeOf(tc.wantResponse).Kind() == reflect.Ptr {
				t.Fatal("wantResponseHeader should not be of pointer type")
			}

			res, err := ts.executeRequest(tc.requestMethodType, tc.requestUrlPath, tc.requestBody, tc.requestHeader)
			require.NoError(t, err)
			defer res.Body.Close()

			assert.Equal(t, tc.wantResponseStatusCode, res.StatusCode, "response status codes do not match")
			assert.Equal(t, res.Header.Get("Content-Type"), "application/json", "response content type is not application/json")

			if tc.wantResponse != nil {
				// reflect.New() returns a pointer to the type
				dst := reflect.New(reflect.TypeOf(tc.wantResponse)).Interface()
				readJsonResponse(t, res.Body, dst)

				// deference the pointer to get the value
				gotResponse := reflect.ValueOf(dst).Elem().Interface()
				assert.Equal(t, tc.wantResponse, gotResponse)
			}

			for key, wantVal := range tc.wantResponseHeader {
				gotVal := res.Header.Get(key)
				assert.Equalf(t, wantVal, gotVal, "Header values do not match for %s", key)
			}

			if tc.additionalChecks != nil {
				tc.additionalChecks(t, res)
			}
		})
	}
}
