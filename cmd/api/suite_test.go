package main

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

type handlerTestcase struct {
	name              string
	requestMethodType string
	requestUrlPath    string
	requestBody       string
	requestHeader     http.Header
	responseDst       any

	wantResponseStatusCode int
	wantResponse           any
	wantResponseHeader     map[string]string
}

func testHandler(t *testing.T, app *application, testcases ...handlerTestcase) {
	ts := newTestServer(app.routes())

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := ts.do(tc.requestMethodType, tc.requestUrlPath, tc.requestBody, tc.requestHeader)
			require.NoError(t, err)

			assert.Equal(t, tc.wantResponseStatusCode, res.StatusCode, "response status codes do not match")

			if tc.wantResponse != nil {
				readJsonResponse(t, res, tc.responseDst)
				assert.Equal(t, tc.wantResponse, tc.responseDst)
			}

			for key, wantVal := range tc.wantResponseHeader {
				gotVal := res.Header.Get(key)
				assert.Equalf(t, wantVal, gotVal, "Header values do not match for %s", key)
			}
		})
	}
}
