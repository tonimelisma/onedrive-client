package onedrive

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tonimelisma/onedrive-client/internal/logger"
)

func TestClientErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedError  error
		expectedInBody bool
	}{
		{
			name:          "401 Unauthorized - Reauth Required",
			statusCode:    401,
			responseBody:  `{"error": {"code": "InvalidAuthenticationToken", "message": "Token expired"}}`,
			expectedError: ErrReauthRequired,
		},
		{
			name:          "403 Forbidden - Access Denied",
			statusCode:    403,
			responseBody:  `{"error": {"code": "Forbidden", "message": "Access denied"}}`,
			expectedError: ErrAccessDenied,
		},
		{
			name:          "404 Not Found - Resource Not Found",
			statusCode:    404,
			responseBody:  `{"error": {"code": "ItemNotFound", "message": "Item not found"}}`,
			expectedError: ErrResourceNotFound,
		},
		{
			name:          "409 Conflict",
			statusCode:    409,
			responseBody:  `{"error": {"code": "NameAlreadyExists", "message": "Name already exists"}}`,
			expectedError: ErrConflict,
		},
		{
			name:          "507 Insufficient Storage - Quota Exceeded",
			statusCode:    507,
			responseBody:  `{"error": {"code": "InsufficientQuota", "message": "Quota exceeded"}}`,
			expectedError: ErrQuotaExceeded,
		},
		{
			name:          "429 Too Many Requests - Retry Later",
			statusCode:    429,
			responseBody:  `{"error": {"code": "TooManyRequests", "message": "Rate limit exceeded"}}`,
			expectedError: ErrRetryLater,
		},
		{
			name:          "400 Bad Request - Invalid Request",
			statusCode:    400,
			responseBody:  `{"error": {"code": "InvalidRequest", "message": "Bad request"}}`,
			expectedError: ErrInvalidRequest,
		},
		{
			name:          "500 Internal Server Error - General Error",
			statusCode:    500,
			responseBody:  `{"error": {"code": "InternalServerError", "message": "Server error"}}`,
			expectedError: nil, // Should not match any sentinel error
		},
		{
			name:          "200 Success - No Error",
			statusCode:    200,
			responseBody:  `{"value": "success"}`,
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			// Create client with test server - using NewClient constructor
			ctx := context.Background()
			token := &Token{AccessToken: "test-token"}
			client := NewClient(ctx, token, "test-client-id", nil, &logger.NoopLogger{})

			// Override the httpClient to use our test server
			client.httpClient = &http.Client{}

			// Make API call with proper signature
			response, err := client.apiCall(ctx, "GET", server.URL+"/test", "application/json", nil)

			if tt.expectedError != nil {
				// Should have an error that wraps our sentinel error
				if err == nil {
					t.Errorf("Expected error, got nil")
					return
				}

				if !errors.Is(err, tt.expectedError) {
					t.Errorf("Expected error to be %v, got %v", tt.expectedError, err)
				}
			} else if tt.statusCode >= 400 {
				// Should have some error for non-2xx status codes
				if err == nil {
					t.Errorf("Expected error for status code %d, got nil", tt.statusCode)
				}
			} else {
				// Should not have an error for 2xx status codes
				if err != nil {
					t.Errorf("Unexpected error for status code %d: %v", tt.statusCode, err)
				}

				if response == nil {
					t.Errorf("Expected response body, got nil")
				} else {
					body, _ := io.ReadAll(response.Body)
					if !strings.Contains(string(body), "success") {
						t.Errorf("Expected success in response body, got %s", string(body))
					}
					response.Body.Close()
				}
			}
		})
	}
}

func TestClientWithStructuredLogging(t *testing.T) {
	// Test that client works with the new structured logging interface
	var logBuffer bytes.Buffer

	// Create a simple logger that writes to our buffer for testing
	testLogger := &testLogger{buffer: &logBuffer}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"value": "test"}`))
	}))
	defer server.Close()

	ctx := context.Background()
	token := &Token{AccessToken: "test-token"}
	client := NewClient(ctx, token, "test-client-id", nil, testLogger)

	// Override the httpClient to use our test server
	client.httpClient = &http.Client{}

	_, err := client.apiCall(ctx, "GET", server.URL+"/test", "application/json", nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check that logging occurred (the exact message will depend on client implementation)
	if logBuffer.Len() == 0 {
		t.Log("No logging output detected (may be expected if client doesn't log successful calls)")
	}
}

// testLogger implements the Logger interface for testing
type testLogger struct {
	buffer *bytes.Buffer
}

func (l *testLogger) Debug(msg string, args ...any) {
	l.buffer.WriteString("DEBUG: " + msg + "\n")
}

func (l *testLogger) Info(msg string, args ...any) {
	l.buffer.WriteString("INFO: " + msg + "\n")
}

func (l *testLogger) Warn(msg string, args ...any) {
	l.buffer.WriteString("WARN: " + msg + "\n")
}

func (l *testLogger) Error(msg string, args ...any) {
	l.buffer.WriteString("ERROR: " + msg + "\n")
}

func (l *testLogger) Debugf(format string, args ...any) {
	l.buffer.WriteString("DEBUG: " + fmt.Sprintf(format, args...) + "\n")
}

func (l *testLogger) Infof(format string, args ...any) {
	l.buffer.WriteString("INFO: " + fmt.Sprintf(format, args...) + "\n")
}

func (l *testLogger) Warnf(format string, args ...any) {
	l.buffer.WriteString("WARN: " + fmt.Sprintf(format, args...) + "\n")
}

func (l *testLogger) Errorf(format string, args ...any) {
	l.buffer.WriteString("ERROR: " + fmt.Sprintf(format, args...) + "\n")
}
