package main

import (
	"net/http"
	"testing"
)

func TestHealthcheckHandler(t *testing.T) {
	want := healthCheckResponse{
		Status: "available",
		SystemInfo: map[string]string{
			"environment": "development",
			"version":     version,
		},
	}

	requestUrlPath := "/v1/healthcheck"

	validResponseTC := handlerTestcase{
		name:                   "Valid response",
		requestMethodType:      http.MethodGet,
		requestUrlPath:         requestUrlPath,
		wantResponseStatusCode: http.StatusOK,
		wantResponse:           want,
	}

	methodNotAllowedTC := handlerTestcase{
		name:                   "Method not allowed",
		requestMethodType:      http.MethodPost,
		requestUrlPath:         requestUrlPath,
		wantResponseStatusCode: http.StatusMethodNotAllowed,
	}

	testHandler(t, newTestApplication(nil), validResponseTC)
	testHandler(t, newTestApplication(nil), methodNotAllowedTC)
}
