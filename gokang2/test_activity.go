package activities

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/testsuite"

	"your-module/restclient" // Replace with your actual module path
)

// Test data structures
type TestUser struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type TestError struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Mock logger for testing
type testLogger struct{}

func (l *testLogger) Debug(msg string, keyvals ...interface{}) {}
func (l *testLogger) Info(msg string, keyvals ...interface{})  {}
func (l *testLogger) Warn(msg string, keyvals ...interface{})  {}
func (l *testLogger) Error(msg string, keyvals ...interface{}) {}

// Test server setup
func createTestServer(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()

	// Success endpoints
	mux.HandleFunc("/users/1", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			user := TestUser{ID: 1, Name: "John Doe", Email: "john@example.com"}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(user)
		case "PUT":
			var user TestUser
			json.NewDecoder(r.Body).Decode(&user)
			user.ID = 1
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(user)
		case "DELETE":
			w.WriteHeader(http.StatusNoContent)
		}
	})

	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			var user TestUser
			json.NewDecoder(r.Body).Decode(&user)
			user.ID = 123
			w.WriteHeader(http.StatusCreated)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(user)
		}
	})

	// Error simulation endpoints
	mux.HandleFunc("/error/500", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(TestError{Error: "Internal Server Error", Code: 500})
	})

	mux.HandleFunc("/error/429", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(TestError{Error: "Too Many Requests", Code: 429})
	})

	mux.HandleFunc("/error/400", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TestError{Error: "Bad Request", Code: 400})
	})

	// Retry test endpoint (fails first 2 times, succeeds on 3rd)
	var attemptCount int
	mux.HandleFunc("/retry-test", func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(TestError{Error: "Temporary failure", Code: 500})
		} else {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"message": "success after retry",
				"attempt": attemptCount,
			})
		}
	})

	// Slow endpoint for timeout testing
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "slow response"})
	})

	// Authentication test
	mux.HandleFunc("/auth/bearer", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(TestError{Error: "Unauthorized", Code: 401})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "authenticated"})
	})

	return httptest.NewServer(mux)
}

func TestRESTServiceActivities_InvokeRESTService(t *testing.T) {
	server := createTestServer(t)
	defer server.Close()

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	activities := NewRESTServiceActivities(&testLogger{})
	env.RegisterActivity(activities.InvokeRESTService)

	tests := []struct {
		name           string
		request        RESTServiceRequest
		expectedStatus int
		expectSuccess  bool
		checkResponse  func(t *testing.T, resp *RESTServiceResponse)
	}{
		{
			name: "Successful GET request",
			request: RESTServiceRequest{
				ServiceName: "UserService",
				BaseURL:     server.URL,
				Auth:        restclient.AuthConfig{Type: restclient.NoAuth},
				Request: restclient.RESTRequest{
					Method:   restclient.GET,
					Endpoint: "/users/1",
				},
			},
			expectedStatus: 200,
			expectSuccess:  true,
			checkResponse: func(t *testing.T, resp *RESTServiceResponse) {
				var user TestUser
				err := json.Unmarshal([]byte(resp.Body), &user)
				assert.NoError(t, err)
				assert.Equal(t, 1, user.ID)
				assert.Equal(t, "John Doe", user.Name)
			},
		},
		{
			name: "Successful POST request",
			request: RESTServiceRequest{
				ServiceName: "UserService",
				BaseURL:     server.URL,
				Auth:        restclient.AuthConfig{Type: restclient.NoAuth},
				Request: restclient.RESTRequest{
					Method:   restclient.POST,
					Endpoint: "/users",
					Body: TestUser{
						Name:  "Alice Johnson",
						Email: "alice@example.com",
					},
				},
			},
			expectedStatus: 201,
			expectSuccess:  true,
			checkResponse: func(t *testing.T, resp *RESTServiceResponse) {
				var user TestUser
				err := json.Unmarshal([]byte(resp.Body), &user)
				assert.NoError(t, err)
				assert.Equal(t, 123, user.ID)
				assert.Equal(t, "Alice Johnson", user.Name)
			},
		},
		{
			name: "Server error response",
			request: RESTServiceRequest{
				ServiceName: "UserService",
				BaseURL:     server.URL,
				Auth:        restclient.AuthConfig{Type: restclient.NoAuth},
				Request: restclient.RESTRequest{
					Method:   restclient.GET,
					Endpoint: "/error/500",
				},
			},
			expectedStatus: 500,
			expectSuccess:  false,
			checkResponse: func(t *testing.T, resp *RESTServiceResponse) {
				assert.Contains(t, resp.ErrorMessage, "HTTP 500")
				var errorResp TestError
				err := json.Unmarshal([]byte(resp.Body), &errorResp)
				assert.NoError(t, err)
				assert.Equal(t, "Internal Server Error", errorResp.Error)
			},
		},
		{
			name: "Authentication required",
			request: RESTServiceRequest{
				ServiceName: "AuthService",
				BaseURL:     server.URL,
				Auth: restclient.AuthConfig{
					Type:  restclient.BearerAuth,
					Token: "test-token",
				},
				Request: restclient.RESTRequest{
					Method:   restclient.GET,
					Endpoint: "/auth/bearer",
				},
			},
			expectedStatus: 200,
			expectSuccess:  true,
			checkResponse: func(t *testing.T, resp *RESTServiceResponse) {
				var response map[string]string
				err := json.Unmarshal([]byte(resp.Body), &response)
				assert.NoError(t, err)
				assert.Equal(t, "authenticated", response["message"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := env.ExecuteActivity(activities.InvokeRESTService, tt.request)

			assert.NoError(t, err)

			var response RESTServiceResponse
			err = val.Get(&response)
			assert.NoError(t, err)

			assert.Equal(t, tt.request.ServiceName, response.ServiceName)
			assert.Equal(t, tt.expectedStatus, response.StatusCode)
			assert.Equal(t, tt.expectSuccess, response.Success)

			if tt.checkResponse != nil {
				tt.checkResponse(t, &response)
			}
		})
	}
}

func TestRESTServiceActivities_InvokeRESTServiceWithRetry(t *testing.T) {
	server := createTestServer(t)
	defer server.Close()

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	activities := NewRESTServiceActivities(&testLogger{})
	env.RegisterActivity(activities.InvokeRESTService)
	env.RegisterActivity(activities.InvokeRESTServiceWithRetry)

	tests := []struct {
		name          string
		request       RESTServiceRequest
		expectSuccess bool
		expectedRetries int
		checkResponse func(t *testing.T, resp *RESTServiceResponse)
	}{
		{
			name: "Success after retries",
			request: RESTServiceRequest{
				ServiceName: "RetryService",
				BaseURL:     server.URL,
				Auth:        restclient.AuthConfig{Type: restclient.NoAuth},
				Request: restclient.RESTRequest{
					Method:   restclient.GET,
					Endpoint: "/retry-test",
				},
				Retry: &RetryConfig{
					MaxAttempts:          3,
					InitialBackoff:       100 * time.Millisecond,
					BackoffMultiplier:    2.0,
					RetryableStatusCodes: []int{500},
				},
			},
			expectSuccess: true,
			expectedRetries: 2, // Failed 2 times, succeeded on 3rd
			checkResponse: func(t *testing.T, resp *RESTServiceResponse) {
				var response map[string]interface{}
				err := json.Unmarshal([]byte(resp.Body), &response)
				assert.NoError(t, err)
				assert.Equal(t, "success after retry", response["message"])
			},
		},
		{
			name: "Non-retryable error",
			request: RESTServiceRequest{
				ServiceName: "BadRequestService",
				BaseURL:     server.URL,
				Auth:        restclient.AuthConfig{Type: restclient.NoAuth},
				Request: restclient.RESTRequest{
					Method:   restclient.GET,
					Endpoint: "/error/400",
				},
				Retry: &RetryConfig{
					MaxAttempts:          3,
					InitialBackoff:       100 * time.Millisecond,
					RetryableStatusCodes: []int{500, 429}, // 400 not in list
				},
			},
			expectSuccess: false,
			expectedRetries: 0, // Should not retry 400 errors
			checkResponse: func(t *testing.T, resp *RESTServiceResponse) {
				assert.Equal(t, 400, resp.StatusCode)
				assert.Contains(t, resp.Body, "Bad Request")
			},
		},
		{
			name: "Max retries exceeded",
			request: RESTServiceRequest{
				ServiceName: "FailingService",
				BaseURL:     server.URL,
				Auth:        restclient.AuthConfig{Type: restclient.NoAuth},
				Request: restclient.RESTRequest{
					Method:   restclient.GET,
					Endpoint: "/error/500",
				},
				Retry: &RetryConfig{
					MaxAttempts:          2,
					InitialBackoff:       50 * time.Millisecond,
					RetryableStatusCodes: []int{500},
				},
			},
			expectSuccess: false,
			expectedRetries: 1, // 1 retry (2 total attempts)
			checkResponse: func(t *testing.T, resp *RESTServiceResponse) {
				assert.Equal(t, 500, resp.StatusCode)
				assert.Contains(t, resp.ErrorMessage, "All 2 attempts failed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := env.ExecuteActivity(activities.InvokeRESTServiceWithRetry, tt.request)

			assert.NoError(t, err)

			var response RESTServiceResponse
			err = val.Get(&response)
			assert.NoError(t, err)

			assert.Equal(t, tt.expectSuccess, response.Success)
			assert.Equal(t, tt.expectedRetries, response.Retries)

			if tt.checkResponse != nil {
				tt.checkResponse(t, &response)
			}
		})
	}
}

func TestRESTServiceActivities_CRUDOperations(t *testing.T) {
	server := createTestServer(t)
	defer server.Close()

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	activities := NewRESTServiceActivities(&testLogger{})
	env.RegisterActivity(activities.GetResource)
	env.RegisterActivity(activities.CreateResource)
	env.RegisterActivity(activities.UpdateResource)
	env.RegisterActivity(activities.DeleteResource)

	auth := restclient.AuthConfig{Type: restclient.NoAuth}

	t.Run("GetResource", func(t *testing.T) {
		val, err := env.ExecuteActivity(
			activities.GetResource,
			"UserService",
			server.URL,
			"/users/1",
			auth,
			nil,
		)

		assert.NoError(t, err)

		var response RESTServiceResponse
		err = val.Get(&response)
		assert.NoError(t, err)

		assert.True(t, response.Success)
		assert.Equal(t, 200, response.StatusCode)

		var user TestUser
		err = json.Unmarshal([]byte(response.Body), &user)
		assert.NoError(t, err)
		assert.Equal(t, 1, user.ID)
	})

	t.Run("CreateResource", func(t *testing.T) {
		newUser := TestUser{
			Name:  "Bob Wilson",
			Email: "bob@example.com",
		}

		val, err := env.ExecuteActivity(
			activities.CreateResource,
			"UserService",
			server.URL,
			"/users",
			auth,
			newUser,
		)

		assert.NoError(t, err)

		var response RESTServiceResponse
		err = val.Get(&response)
		assert.NoError(t, err)

		assert.True(t, response.Success)
		assert.Equal(t, 201, response.StatusCode)

		var createdUser TestUser
		err = json.Unmarshal([]byte(response.Body), &createdUser)
		assert.NoError(t, err)
		assert.Equal(t, 123, createdUser.ID)
		assert.Equal(t, "Bob Wilson", createdUser.Name)
	})

	t.Run("UpdateResource", func(t *testing.T) {
		updateUser := TestUser{
			Name:  "John Updated",
			Email: "john.updated@example.com",
		}

		val, err := env.ExecuteActivity(
			activities.UpdateResource,
			"UserService",
			server.URL,
			"/users/1",
			auth,
			updateUser,
		)

		assert.NoError(t, err)

		var response RESTServiceResponse
		err = val.Get(&response)
		assert.NoError(t, err)

		assert.True(t, response.Success)
		assert.Equal(t, 200, response.StatusCode)

		var updatedUser TestUser
		err = json.Unmarshal([]byte(response.Body), &updatedUser)
		assert.NoError(t, err)
		assert.Equal(t, 1, updatedUser.ID)
		assert.Equal(t, "John Updated", updatedUser.Name)
	})

	t.Run("DeleteResource", func(t *testing.T) {
		val, err := env.ExecuteActivity(
			activities.DeleteResource,
			"UserService",
			server.URL,
			"/users/1",
			auth,
		)

		assert.NoError(t, err)

		var response RESTServiceResponse
		err = val.Get(&response)
		assert.NoError(t, err)

		assert.True(t, response.Success)
		assert.Equal(t, 204, response.StatusCode)
	})
}

func TestRESTServiceActivities_BatchRESTCalls(t *testing.T) {
	server := createTestServer(t)
	defer server.Close()

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	activities := NewRESTServiceActivities(&testLogger{})
	env.RegisterActivity(activities.InvokeRESTService)
	env.RegisterActivity(activities.BatchRESTCalls)

	requests := []RESTServiceRequest{
		{
			ServiceName: "UserService",
			BaseURL:     server.URL,
			Auth:        restclient.AuthConfig{Type: restclient.NoAuth},
			Request: restclient.RESTRequest{
				Method:   restclient.GET,
				Endpoint: "/users/1",
			},
		},
		{
			ServiceName: "UserService",
			BaseURL:     server.URL,
			Auth:        restclient.AuthConfig{Type: restclient.NoAuth},
			Request: restclient.RESTRequest{
				Method:   restclient.POST,
				Endpoint: "/users",
				Body: TestUser{
					Name:  "Batch User",
					Email: "batch@example.com",
				},
			},
		},
		{
			ServiceName: "ErrorService",
			BaseURL:     server.URL,
			Auth:        restclient.AuthConfig{Type: restclient.NoAuth},
			Request: restclient.RESTRequest{
				Method:   restclient.GET,
				Endpoint: "/error/500",
			},
		},
	}

	val, err := env.ExecuteActivity(activities.BatchRESTCalls, requests)
	assert.NoError(t, err)

	var responses []*RESTServiceResponse
	err = val.Get(&responses)
	assert.NoError(t, err)

	assert.Len(t, responses, 3)

	// First request should succeed
	assert.True(t, responses[0].Success)
	assert.Equal(t, 200, responses[0].StatusCode)

	// Second request should succeed
	assert.True(t, responses[1].Success)
	assert.Equal(t, 201, responses[1].StatusCode)

	// Third request should fail
	assert.False(t, responses[2].Success)
	assert.Equal(t, 500, responses[2].StatusCode)
}

func TestRESTServiceActivities_ValidateRESTResponse(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	activities := NewRESTServiceActivities(&testLogger{})
	env.RegisterActivity(activities.ValidateRESTResponse)

	tests := []struct {
		name            string
		response        *RESTServiceResponse
		expectedStatus  int
		requiredFields  []string
		expectError     bool
		expectedError   string
	}{
		{
			name: "Valid response with correct status",
			response: &RESTServiceResponse{
				StatusCode: 200,
				Body:       `{"id": 1, "name": "John", "email": "john@example.com"}`,
				Success:    true,
			},
			expectedStatus: 200,
			requiredFields: []string{"id", "name", "email"},
			expectError:    false,
		},
		{
			name: "Invalid status code",
			response: &RESTServiceResponse{
				StatusCode: 404,
				Body:       `{"error": "Not found"}`,
				Success:    false,
			},
			expectedStatus: 200,
			expectError:    true,
			expectedError:  "expected status code 200, got 404",
		},
		{
			name: "Missing required field",
			response: &RESTServiceResponse{
				StatusCode: 200,
				Body:       `{"id": 1, "name": "John"}`,
				Success:    true,
			},
			expectedStatus: 200,
			requiredFields: []string{"id", "name", "email"},
			expectError:    true,
			expectedError:  "required field 'email' not found",
		},
		{
			name: "Invalid JSON response",
			response: &RESTServiceResponse{
				StatusCode: 200,
				Body:       `invalid json`,
				Success:    true,
			},
			expectedStatus: 200,
			requiredFields: []string{"id"},
			expectError:    true,
			expectedError:  "failed to parse JSON response",
		},
		{
			name: "Failed response",
			response: &RESTServiceResponse{
				StatusCode:   500,
				Body:         `{"error": "Internal server error"}`,
				Success:      false,
				ErrorMessage: "HTTP 500: Internal Server Error",
			},
			expectError:   true,
			expectedError: "API call failed with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := env.ExecuteActivity(
				activities.ValidateRESTResponse,
				tt.response,
				tt.expectedStatus,
				tt.requiredFields,
			)

			assert.NoError(t, err)

			var result interface{}
			err = val.Get(&result)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRESTServiceActivities_Timeout(t *testing.T) {
	server := createTestServer(t)
	defer server.Close()

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	activities := NewRESTServiceActivities(&testLogger{})
	env.RegisterActivity(activities.InvokeRESTService)

	request := RESTServiceRequest{
		ServiceName: "SlowService",
		BaseURL:     server.URL,
		Auth:        restclient.AuthConfig{Type: restclient.NoAuth},
		Request: restclient.RESTRequest{
			Method:   restclient.GET,
			Endpoint: "/slow",
		},
		Timeout: 1 * time.Second, // Shorter than server delay (2 seconds)
	}

	val, err := env.ExecuteActivity(activities.InvokeRESTService, request)
	assert.NoError(t, err)

	var response RESTServiceResponse
	err = val.Get(&response)
	assert.Error(t, err) // Should timeout
}

// Integration test with mock workflow
func TestRESTServiceActivities_WorkflowIntegration(t *testing.T) {
	server := createTestServer(t)
	defer server.Close()

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	activities := NewRESTServiceActivities(&testLogger{})
	env.RegisterActivity(activities.InvokeRESTService)
	env.RegisterActivity(activities.CreateResource)
	env.RegisterActivity(activities.GetResource)

	// Mock workflow that creates a user and then retrieves it
	userWorkflow := func(ctx context.Context, userName, userEmail string) (*TestUser, error) {
		var createdUser TestUser

		// Create user
		createReq := RESTServiceRequest{
			ServiceName: "UserService",
			BaseURL:     server.URL,
			Auth:        restclient.AuthConfig{Type: restclient.NoAuth},
			Request: restclient.RESTRequest{
				Method:   restclient.POST,
				Endpoint: "/users",
				Body: TestUser{
					Name:  userName,
					Email: userEmail,
				},
			},
		}

		var createResp RESTServiceResponse
		err := env.ExecuteActivity(activities.InvokeRESTService, createReq).Get(ctx, &createResp)
		if err != nil {
			return nil, err
		}

		if !createResp.Success {
			return nil, fmt.Errorf("failed to create user: %s", createResp.ErrorMessage)
		}

		err = json.Unmarshal([]byte(createResp.Body), &createdUser)
		if err != nil {
			return nil, err
		}

		return &createdUser, nil
	}

	env.ExecuteWorkflow(userWorkflow, "Test User", "test@example.com")

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result TestUser
	err := env.GetWorkflowResult(&result)
	assert.NoError(t, err)
	assert.Equal(t, 123, result.ID)
	assert.Equal(t, "Test User", result.Name)
	assert.Equal(t, "test@example.com", result.Email)
}

// Benchmark tests
func BenchmarkRESTServiceActivities_InvokeRESTService(b *testing.B) {
	server := createTestServer(nil)
	defer server.Close()

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	activities := NewRESTServiceActivities(&testLogger{})
	env.RegisterActivity(activities.InvokeRESTService)

	request := RESTServiceRequest{
		ServiceName: "BenchmarkService",
		BaseURL:     server.URL,
		Auth:        restclient.AuthConfig{Type: restclient.NoAuth},
		Request: restclient.RESTRequest{
			Method:   restclient.GET,
			Endpoint: "/users/1",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		val, err := env.ExecuteActivity(activities.InvokeRESTService, request)
		if err != nil {
			b.Fatal(err)
		}

		var response RESTServiceResponse
		err = val.Get(&response)
		if err != nil {
			b.Fatal(err)
		}

		if !response.Success {
			b.Fatal("Request failed")
		}
	}
}

func TestRESTActivityOptions(t *testing.T) {
	options := GetRESTActivityOptions()

	assert.NotNil(t, options)
	assert.Equal(t, 60*time.Second, options.StartToCloseTimeout)
	assert.Equal(t, 10*time.Second, options.HeartbeatTimeout)
	assert.NotNil(t, options.RetryPolicy)
	assert.Equal(t, 2*time.Second, options.RetryPolicy.InitialInterval)
	assert.Equal(t, 2.0, options.RetryPolicy.BackoffCoefficient)
}