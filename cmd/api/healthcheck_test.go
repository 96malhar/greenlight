package main

import (
	"net/http"
	"testing"
)

type healthCheckResponse struct {
	Status     string            `json:"status"`
	SystemInfo map[string]string `json:"system_info"`
}

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

	ts := newTestServer(t)
	testHandler(t, ts, validResponseTC)
	testHandler(t, ts, methodNotAllowedTC)
}
