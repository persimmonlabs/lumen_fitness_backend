package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// NewTestRequest creates a new test HTTP request with JSON body
func NewTestRequest(t *testing.T, method, path string, body interface{}) *http.Request {
	t.Helper()

	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("Failed to encode request body: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	return req
}

// NewResponseRecorder creates a new response recorder
func NewResponseRecorder() *httptest.ResponseRecorder {
	return httptest.NewRecorder()
}

// ParseJSONResponse parses JSON response body into the given type
func ParseJSONResponse[T any](t *testing.T, w *httptest.ResponseRecorder) *T {
	t.Helper()

	var result T
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}
	return &result
}

// AssertStatusCode checks if the response has the expected status code
func AssertStatusCode(t *testing.T, w *httptest.ResponseRecorder, expected int) {
	t.Helper()

	if w.Code != expected {
		body, _ := io.ReadAll(w.Body)
		t.Errorf("Expected status code %d, got %d. Body: %s", expected, w.Code, string(body))
	}
}

// AssertJSONResponse checks status code and parses JSON response
func AssertJSONResponse[T any](t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) *T {
	t.Helper()

	AssertStatusCode(t, w, expectedStatus)
	return ParseJSONResponse[T](t, w)
}
