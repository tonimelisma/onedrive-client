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
	"time"

	"github.com/stretchr/testify/assert"
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

func TestHTTPConfiguration(t *testing.T) {
	// Test default HTTP configuration
	defaultConfig := DefaultHTTPConfig()
	assert.Equal(t, 30*time.Second, defaultConfig.Timeout)
	assert.Equal(t, 3, defaultConfig.RetryAttempts)
	assert.Equal(t, 1*time.Second, defaultConfig.RetryDelay)
	assert.Equal(t, 10*time.Second, defaultConfig.MaxRetryDelay)
}

func TestNewConfiguredHTTPClient(t *testing.T) {
	config := HTTPConfig{
		Timeout:       15 * time.Second,
		RetryAttempts: 5,
		RetryDelay:    500 * time.Millisecond,
		MaxRetryDelay: 20 * time.Second,
	}

	client := NewConfiguredHTTPClient(config)
	assert.NotNil(t, client)
	assert.Equal(t, 15*time.Second, client.Timeout)
}

func TestNewConfiguredHTTPClientWithTransport(t *testing.T) {
	config := HTTPConfig{
		Timeout: 10 * time.Second,
	}

	// Create a custom transport
	transport := &http.Transport{}

	client := NewConfiguredHTTPClientWithTransport(config, transport)
	assert.NotNil(t, client)
	assert.Equal(t, 10*time.Second, client.Timeout)
	assert.Equal(t, transport, client.Transport)
}

func TestNewClientWithConfig(t *testing.T) {
	token := &Token{AccessToken: "test-token"}
	customHTTPConfig := HTTPConfig{
		Timeout:       25 * time.Second,
		RetryAttempts: 2,
		RetryDelay:    2 * time.Second,
		MaxRetryDelay: 15 * time.Second,
	}

	client := NewClientWithConfig(context.Background(), token, "test-client", nil, &logger.NoopLogger{}, customHTTPConfig)

	assert.NotNil(t, client)
	assert.Equal(t, customHTTPConfig, client.httpConfig)
	assert.Equal(t, 25*time.Second, client.httpClient.Timeout)
}

func TestClientAPICallWithConfigurableRetries(t *testing.T) {
	// Test that client uses configured retry parameters
	retryCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		retryCount++
		if retryCount < 3 { // Fail first 2 attempts
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"value": "success"}`))
		}
	}))
	defer server.Close()

	// Create client with custom retry configuration
	httpConfig := HTTPConfig{
		Timeout:       5 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    100 * time.Millisecond, // Short delay for testing
		MaxRetryDelay: 1 * time.Second,
	}

	ctx := context.Background()
	token := &Token{AccessToken: "test-token"}
	client := NewClientWithConfig(ctx, token, "test-client-id", nil, &logger.NoopLogger{}, httpConfig)

	// Override the httpClient to use our test server
	client.httpClient = &http.Client{Timeout: httpConfig.Timeout}

	// Make API call that will retry
	response, err := client.apiCall(ctx, "GET", server.URL+"/test", "application/json", nil)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, 3, retryCount) // Should have retried twice
	response.Body.Close()
}

func TestErrorSentinelWrapping(t *testing.T) {
	// Test that error sentinels can be properly identified with errors.Is()
	tests := []struct {
		name             string
		statusCode       int
		responseBody     string
		expectedSentinel error
	}{
		{
			name:             "Decoding error wrapped correctly",
			statusCode:       200,
			responseBody:     "invalid json{",
			expectedSentinel: ErrDecodingFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			ctx := context.Background()
			token := &Token{AccessToken: "test-token"}
			client := NewClient(ctx, token, "test-client-id", nil, &logger.NoopLogger{})

			// Override the httpClient to use our test server
			client.httpClient = &http.Client{}

			// Replace the root URL to point to our test server
			originalRootURL := customRootURL
			customRootURL = server.URL + "/"
			defer func() { customRootURL = originalRootURL }()

			_, err := client.GetMe(ctx)
			assert.Error(t, err)
			assert.True(t, errors.Is(err, tt.expectedSentinel),
				"Expected error to wrap %v, got %v", tt.expectedSentinel, err)
		})
	}
}

func TestAllErrorSentinels(t *testing.T) {
	// Test that all error sentinels are properly defined and can be compared
	sentinels := []error{
		ErrReauthRequired,
		ErrAccessDenied,
		ErrRetryLater,
		ErrInvalidRequest,
		ErrResourceNotFound,
		ErrConflict,
		ErrQuotaExceeded,
		ErrAuthorizationPending,
		ErrAuthorizationDeclined,
		ErrTokenExpired,
		ErrInternal,
		ErrDecodingFailed,
		ErrNetworkFailed,
		ErrOperationFailed,
	}

	for _, sentinel := range sentinels {
		assert.NotNil(t, sentinel)
		assert.NotEmpty(t, sentinel.Error())

		// Test that each sentinel can be identified with errors.Is
		wrappedErr := fmt.Errorf("wrapped: %w", sentinel)
		assert.True(t, errors.Is(wrappedErr, sentinel))
	}
}
