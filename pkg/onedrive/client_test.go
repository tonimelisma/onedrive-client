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

// Tests for the new helper functions from apiCall refactoring
func TestIsSuccessStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		expected bool
	}{
		{"200 OK", 200, true},
		{"201 Created", 201, true},
		{"202 Accepted", 202, true},
		{"204 No Content", 204, true},
		{"400 Bad Request", 400, false},
		{"401 Unauthorized", 401, false},
		{"404 Not Found", 404, false},
		{"500 Internal Server Error", 500, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSuccessStatus(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsRetryableStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		expected bool
	}{
		{"429 Too Many Requests", 429, true},
		{"401 Unauthorized", 401, true},
		{"503 Service Unavailable", 503, true},
		{"500 Internal Server Error", 500, false},
		{"502 Bad Gateway", 502, false},
		{"504 Gateway Timeout", 504, false},
		{"400 Bad Request", 400, false},
		{"403 Forbidden", 403, false},
		{"404 Not Found", 404, false},
		{"409 Conflict", 409, false},
		{"200 OK", 200, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableStatus(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetStatusDescription(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		expected string
	}{
		{"200 OK", 200, "unexpected status"},
		{"201 Created", 201, "unexpected status"},
		{"400 Bad Request", 400, "bad request"},
		{"401 Unauthorized", 401, "unauthorized"},
		{"403 Forbidden", 403, "forbidden"},
		{"404 Not Found", 404, "not found"},
		{"409 Conflict", 409, "conflict"},
		{"413 Payload Too Large", 413, "payload too large"},
		{"429 Too Many Requests", 429, "rate limited"},
		{"500 Internal Server Error", 500, "unexpected status"},
		{"502 Bad Gateway", 502, "unexpected status"},
		{"503 Service Unavailable", 503, "service unavailable"},
		{"504 Gateway Timeout", 504, "unexpected status"},
		{"507 Insufficient Storage", 507, "insufficient storage"},
		{"999 Unknown Status", 999, "unexpected status"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStatusDescription(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHandleRetryableStatus(t *testing.T) {
	ctx := context.Background()
	token := &Token{AccessToken: "test-token"}
	client := NewClient(ctx, token, "test-client-id", nil, &logger.NoopLogger{})

	// Create mock response for testing
	mockResponse := &http.Response{
		StatusCode: 429,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(`{"error": {"code": "TooManyRequests"}}`)),
	}

	// Test with 429 Too Many Requests - should return true (retry)
	result := client.handleRetryableStatus(mockResponse, 1, 3, "https://test.com", nil, time.Microsecond, time.Millisecond)
	assert.True(t, result)

	// Test with 401 Unauthorized - should return true (retry)
	mockResponse.StatusCode = 401
	mockResponse.Body = io.NopCloser(strings.NewReader(`{"error": {"code": "Unauthorized"}}`))
	result = client.handleRetryableStatus(mockResponse, 1, 3, "https://test.com", nil, time.Microsecond, time.Millisecond)
	assert.True(t, result)

	// Test with 503 Service Unavailable - should return true (retry)
	mockResponse.StatusCode = 503
	mockResponse.Body = io.NopCloser(strings.NewReader(`{"error": {"code": "ServiceUnavailable"}}`))
	result = client.handleRetryableStatus(mockResponse, 1, 3, "https://test.com", nil, time.Microsecond, time.Millisecond)
	assert.True(t, result)

	// Test at max attempts - should return false (no more retries)
	mockResponse.StatusCode = 429
	mockResponse.Body = io.NopCloser(strings.NewReader(`{"error": {"code": "TooManyRequests"}}`))
	result = client.handleRetryableStatus(mockResponse, 3, 3, "https://test.com", nil, time.Microsecond, time.Millisecond)
	assert.False(t, result)
}

func TestCreateRetryableError(t *testing.T) {
	ctx := context.Background()
	token := &Token{AccessToken: "test-token"}
	client := NewClient(ctx, token, "test-client-id", nil, &logger.NoopLogger{})

	// Test with 429 Too Many Requests
	err := client.createRetryableError(429, "https://test.com", 3)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rate limited")
	assert.Contains(t, err.Error(), "https://test.com")
	assert.ErrorIs(t, err, ErrRetryLater)

	// Test with 401 Unauthorized
	err = client.createRetryableError(401, "https://test.com", 3)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "401")
	assert.Contains(t, err.Error(), "https://test.com")
	assert.ErrorIs(t, err, ErrReauthRequired)

	// Test with 503 Service Unavailable
	err = client.createRetryableError(503, "https://test.com", 3)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service unavailable")
	assert.Contains(t, err.Error(), "https://test.com")
	assert.ErrorIs(t, err, ErrRetryLater)
}

func TestHandleNonRetryableStatus(t *testing.T) {
	ctx := context.Background()
	token := &Token{AccessToken: "test-token"}
	client := NewClient(ctx, token, "test-client-id", nil, &logger.NoopLogger{})

	tests := []struct {
		name          string
		status        int
		expectedError error
		shouldContain string
	}{
		{
			name:          "403 Forbidden",
			status:        403,
			expectedError: ErrAccessDenied,
			shouldContain: "403",
		},
		{
			name:          "404 Not Found",
			status:        404,
			expectedError: ErrResourceNotFound,
			shouldContain: "404",
		},
		{
			name:          "409 Conflict",
			status:        409,
			expectedError: ErrConflict,
			shouldContain: "409",
		},
		{
			name:          "507 Insufficient Storage",
			status:        507,
			expectedError: ErrQuotaExceeded,
			shouldContain: "507",
		},
		{
			name:          "400 Bad Request",
			status:        400,
			expectedError: ErrInvalidRequest,
			shouldContain: "400",
		},
		{
			name:          "413 Payload Too Large",
			status:        413,
			expectedError: ErrQuotaExceeded,
			shouldContain: "413",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock response for testing
			mockResponse := &http.Response{
				StatusCode: tt.status,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"error": {"code": "TestError", "message": "test message"}}`)),
			}

			err := client.handleNonRetryableStatus(mockResponse, "https://test.com")
			assert.Error(t, err)
			assert.ErrorIs(t, err, tt.expectedError)
			assert.Contains(t, err.Error(), tt.shouldContain)
		})
	}
}

func TestAPICallRefactoredFunctions(t *testing.T) {
	// Test that the refactored apiCall function still works correctly
	// by testing the integration of all helper functions

	// Test successful status
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"value": "success"}`))
	}))
	defer server.Close()

	ctx := context.Background()
	token := &Token{AccessToken: "test-token"}
	client := NewClient(ctx, token, "test-client-id", nil, &logger.NoopLogger{})
	client.httpClient = &http.Client{}

	response, err := client.apiCall(ctx, "GET", server.URL+"/test", "application/json", nil)
	assert.NoError(t, err)
	assert.NotNil(t, response)

	// Test retryable status
	retryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(429)
		w.Write([]byte(`{"error": {"code": "TooManyRequests", "message": "Rate limit exceeded"}}`))
	}))
	defer retryServer.Close()

	_, err = client.apiCall(ctx, "GET", retryServer.URL+"/test", "application/json", nil)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrRetryLater)

	// Test non-retryable status
	errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(404)
		w.Write([]byte(`{"error": {"code": "ItemNotFound", "message": "Item not found"}}`))
	}))
	defer errorServer.Close()

	_, err = client.apiCall(ctx, "GET", errorServer.URL+"/test", "application/json", nil)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrResourceNotFound)
}

func TestErrorHandlingRefactoring(t *testing.T) {
	// Test the error handling improvements by creating scenarios
	// that test different error conditions

	tests := []struct {
		name          string
		statusCode    int
		expectedError error
	}{
		{"Unauthorized", 401, ErrReauthRequired},
		{"Forbidden", 403, ErrAccessDenied},
		{"Not Found", 404, ErrResourceNotFound},
		{"Conflict", 409, ErrConflict},
		{"Too Many Requests", 429, ErrRetryLater},
		{"Quota Exceeded", 507, ErrQuotaExceeded},
		{"Bad Request", 400, ErrInvalidRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(`{"error": {"code": "TestError", "message": "Test error message"}}`))
			}))
			defer server.Close()

			ctx := context.Background()
			token := &Token{AccessToken: "test-token"}
			client := NewClient(ctx, token, "test-client-id", nil, &logger.NoopLogger{})
			client.httpClient = &http.Client{}

			_, err := client.apiCall(ctx, "GET", server.URL+"/test", "application/json", nil)
			assert.Error(t, err)
			assert.ErrorIs(t, err, tt.expectedError)
		})
	}
}
