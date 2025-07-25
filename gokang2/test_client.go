package restclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// Test helper functions
func createTestServer(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()

	// GET /users/{id}
	mux.HandleFunc("/users/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			handleGetUser(w, r)
		case "PUT":
			handleUpdateUser(w, r)
		case "DELETE":
			handleDeleteUser(w, r)
		}
	})

	// POST /users
	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			handleListUsers(w, r)
		case "POST":
			handleCreateUser(w, r)
		}
	})

	// Authentication test endpoints
	mux.HandleFunc("/auth/basic", handleBasicAuth)
	mux.HandleFunc("/auth/bearer", handleBearerAuth)
	mux.HandleFunc("/auth/apikey", handleAPIKeyAuth)

	// Error simulation endpoints
	mux.HandleFunc("/error/400", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TestError{Error: "Bad Request", Code: 400})
	})

	mux.HandleFunc("/error/500", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(TestError{Error: "Internal Server Error", Code: 500})
	})

	mux.HandleFunc("/delay", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "delayed response"})
	})

	return httptest.NewServer(mux)
}

func handleGetUser(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/users/")
	if path == "1" {
		user := TestUser{ID: 1, Name: "John Doe", Email: "john@example.com"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	} else {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(TestError{Error: "User not found", Code: 404})
	}
}

func handleListUsers(w http.ResponseWriter, r *http.Request) {
	limit := r.URL.Query().Get("limit")
	if limit == "" {
		limit = "10"
	}

	users := []TestUser{
		{ID: 1, Name: "John Doe", Email: "john@example.com"},
		{ID: 2, Name: "Jane Smith", Email: "jane@example.com"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"users": users,
		"limit": limit,
		"total": len(users),
	})
}

func handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var user TestUser
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TestError{Error: "Invalid JSON", Code: 400})
		return
	}

	if user.Name == "" || user.Email == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TestError{Error: "Name and email are required", Code: 400})
		return
	}

	user.ID = 123
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	var user TestUser
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TestError{Error: "Invalid JSON", Code: 400})
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/users/")
	if path == "1" {
		user.ID = 1
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	} else {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(TestError{Error: "User not found", Code: 404})
	}
}

func handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/users/")
	if path == "1" {
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(TestError{Error: "User not found", Code: 404})
	}
}

func handleBasicAuth(w http.ResponseWriter, r *http.Request) {
	username, password, ok := r.BasicAuth()
	if !ok || username != "testuser" || password != "testpass" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(TestError{Error: "Unauthorized", Code: 401})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "authenticated"})
}

func handleBearerAuth(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	if auth != "Bearer test-token-123" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(TestError{Error: "Invalid token", Code: 401})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "authenticated"})
}

func handleAPIKeyAuth(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("X-API-Key")
	if apiKey != "test-api-key-456" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(TestError{Error: "Invalid API key", Code: 401})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "authenticated"})
}

// Test cases

func TestNewRESTClient(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		auth      AuthConfig
		wantError bool
	}{
		{
			name:    "No authentication",
			baseURL: "https://api.example.com",
			auth:    AuthConfig{Type: NoAuth},
		},
		{
			name:    "Basic authentication",
			baseURL: "https://api.example.com",
			auth: AuthConfig{
				Type:     BasicAuth,
				Username: "user",
				Password: "pass",
			},
		},
		{
			name:    "Bearer token authentication",
			baseURL: "https://api.example.com",
			auth: AuthConfig{
				Type:  BearerAuth,
				Token: "token123",
			},
		},
		{
			name:    "API Key authentication",
			baseURL: "https://api.example.com",
			auth: AuthConfig{
				Type:   APIKeyAuth,
				APIKey: "key123",
			},
		},
		{
			name:    "OAuth2 authentication - missing config",
			baseURL: "https://api.example.com",
			auth: AuthConfig{
				Type: OAuth2Auth,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewRESTClient(tt.baseURL, tt.auth)

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.Equal(t, tt.baseURL, client.baseURL)
			}
		})
	}
}

func TestRESTClient_GET(t *testing.T) {
	server := createTestServer(t)
	defer server.Close()

	client, err := NewRESTClient(server.URL, AuthConfig{Type: NoAuth})
	require.NoError(t, err)

	tests := []struct {
		name           string
		endpoint       string
		queryParams    map[string]string
		expectedStatus int
		checkBody      func(t *testing.T, body string)
	}{
		{
			name:           "Get existing user",
			endpoint:       "/users/1",
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				var user TestUser
				err := json.Unmarshal([]byte(body), &user)
				assert.NoError(t, err)
				assert.Equal(t, 1, user.ID)
				assert.Equal(t, "John Doe", user.Name)
			},
		},
		{
			name:           "Get non-existing user",
			endpoint:       "/users/999",
			expectedStatus: 404,
			checkBody: func(t *testing.T, body string) {
				var errorResp TestError
				err := json.Unmarshal([]byte(body), &errorResp)
				assert.NoError(t, err)
				assert.Equal(t, "User not found", errorResp.Error)
			},
		},
		{
			name:        "List users with query params",
			endpoint:    "/users",
			queryParams: map[string]string{"limit": "5"},
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				var response map[string]interface{}
				err := json.Unmarshal([]byte(body), &response)
				assert.NoError(t, err)
				assert.Equal(t, "5", response["limit"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			resp, err := client.GET(ctx, tt.endpoint, tt.queryParams)

			assert.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.checkBody != nil {
				tt.checkBody(t, string(resp.Body))
			}
		})
	}
}

func TestRESTClient_POST(t *testing.T) {
	server := createTestServer(t)
	defer server.Close()

	client, err := NewRESTClient(server.URL, AuthConfig{Type: NoAuth})
	require.NoError(t, err)

	tests := []struct {
		name           string
		endpoint       string
		body           interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, resp *RESTResponse)
	}{
		{
			name:     "Create user - success",
			endpoint: "/users",
			body: TestUser{
				Name:  "Alice Johnson",
				Email: "alice@example.com",
			},
			expectedStatus: 201,
			checkResponse: func(t *testing.T, resp *RESTResponse) {
				var user TestUser
				err := json.Unmarshal(resp.Body, &user)
				assert.NoError(t, err)
				assert.Equal(t, 123, user.ID)
				assert.Equal(t, "Alice Johnson", user.Name)
				assert.True(t, resp.IsSuccess())
			},
		},
		{
			name:     "Create user - validation error",
			endpoint: "/users",
			body: TestUser{
				Name: "Incomplete User",
				// Missing email
			},
			expectedStatus: 400,
			checkResponse: func(t *testing.T, resp *RESTResponse) {
				var errorResp TestError
				err := json.Unmarshal(resp.Body, &errorResp)
				assert.NoError(t, err)
				assert.Contains(t, errorResp.Error, "required")
				assert.True(t, resp.IsClientError())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			resp, err := client.POST(ctx, tt.endpoint, tt.body)

			assert.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestRESTClient_PUT(t *testing.T) {
	server := createTestServer(t)
	defer server.Close()

	client, err := NewRESTClient(server.URL, AuthConfig{Type: NoAuth})
	require.NoError(t, err)

	ctx := context.Background()
	updateData := TestUser{
		Name:  "John Updated",
		Email: "john.updated@example.com",
	}

	resp, err := client.PUT(ctx, "/users/1", updateData)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
	assert.True(t, resp.IsSuccess())

	var user TestUser
	err = json.Unmarshal(resp.Body, &user)
	assert.NoError(t, err)
	assert.Equal(t, 1, user.ID)
	assert.Equal(t, "John Updated", user.Name)
}

func TestRESTClient_DELETE(t *testing.T) {
	server := createTestServer(t)
	defer server.Close()

	client, err := NewRESTClient(server.URL, AuthConfig{Type: NoAuth})
	require.NoError(t, err)

	ctx := context.Background()
	resp, err := client.DELETE(ctx, "/users/1")

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 204, resp.StatusCode)
	assert.True(t, resp.IsSuccess())
}

func TestRESTClient_Authentication(t *testing.T) {
	server := createTestServer(t)
	defer server.Close()

	tests := []struct {
		name           string
		auth           AuthConfig
		endpoint       string
		expectedStatus int
	}{
		{
			name: "Basic auth - success",
			auth: AuthConfig{
				Type:     BasicAuth,
				Username: "testuser",
				Password: "testpass",
			},
			endpoint:       "/auth/basic",
			expectedStatus: 200,
		},
		{
			name: "Basic auth - failure",
			auth: AuthConfig{
				Type:     BasicAuth,
				Username: "wronguser",
				Password: "wrongpass",
			},
			endpoint:       "/auth/basic",
			expectedStatus: 401,
		},
		{
			name: "Bearer auth - success",
			auth: AuthConfig{
				Type:  BearerAuth,
				Token: "test-token-123",
			},
			endpoint:       "/auth/bearer",
			expectedStatus: 200,
		},
		{
			name: "Bearer auth - failure",
			auth: AuthConfig{
				Type:  BearerAuth,
				Token: "wrong-token",
			},
			endpoint:       "/auth/bearer",
			expectedStatus: 401,
		},
		{
			name: "API Key auth - success",
			auth: AuthConfig{
				Type:   APIKeyAuth,
				APIKey: "test-api-key-456",
			},
			endpoint:       "/auth/apikey",
			expectedStatus: 200,
		},
		{
			name: "API Key auth - failure",
			auth: AuthConfig{
				Type:   APIKeyAuth,
				APIKey: "wrong-api-key",
			},
			endpoint:       "/auth/apikey",
			expectedStatus: 401,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewRESTClient(server.URL, tt.auth)
			require.NoError(t, err)

			ctx := context.Background()
			resp, err := client.GET(ctx, tt.endpoint, nil)

			assert.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestRESTClient_Timeout(t *testing.T) {
	server := createTestServer(t)
	defer server.Close()

	client, err := NewRESTClient(server.URL, AuthConfig{Type: NoAuth})
	require.NoError(t, err)

	ctx := context.Background()
	req := RESTRequest{
		Method:   GET,
		Endpoint: "/delay",
		Timeout:  1 * time.Second, // Shorter than server delay
	}

	start := time.Now()
	resp, err := client.Execute(ctx, req)
	duration := time.Since(start)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, duration < 2*time.Second, "Request should have timed out before 2 seconds")
}

func TestRESTClient_ErrorStatusCodes(t *testing.T) {
	server := createTestServer(t)
	defer server.Close()

	client, err := NewRESTClient(server.URL, AuthConfig{Type: NoAuth})
	require.NoError(t, err)

	tests := []struct {
		name           string
		endpoint       string
		expectedStatus int
		checkMethod    func(resp *RESTResponse) bool
	}{
		{
			name:           "Client error (400)",
			endpoint:       "/error/400",
			expectedStatus: 400,
			checkMethod:    (*RESTResponse).IsClientError,
		},
		{
			name:           "Server error (500)",
			endpoint:       "/error/500",
			expectedStatus: 500,
			checkMethod:    (*RESTResponse).IsServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			resp, err := client.GET(ctx, tt.endpoint, nil)

			assert.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			assert.False(t, resp.IsSuccess())
			assert.True(t, tt.checkMethod(resp))
		})
	}
}

func TestRESTResponse_UnmarshalJSON(t *testing.T) {
	server := createTestServer(t)
	defer server.Close()

	client, err := NewRESTClient(server.URL, AuthConfig{Type: NoAuth})
	require.NoError(t, err)

	ctx := context.Background()
	resp, err := client.GET(ctx, "/users/1", nil)

	require.NoError(t, err)
	require.NotNil(t, resp)

	var user TestUser
	err = resp.UnmarshalJSON(&user)

	assert.NoError(t, err)
	assert.Equal(t, 1, user.ID)
	assert.Equal(t, "John Doe", user.Name)
	assert.Equal(t, "john@example.com", user.Email)
}

func TestRESTResponse_JSON(t *testing.T) {
	server := createTestServer(t)
	defer server.Close()

	client, err := NewRESTClient(server.URL, AuthConfig{Type: NoAuth})
	require.NoError(t, err)

	ctx := context.Background()
	resp, err := client.GET(ctx, "/users/1", nil)

	require.NoError(t, err)
	require.NotNil(t, resp)

	jsonStr, err := resp.JSON()

	assert.NoError(t, err)
	assert.Contains(t, jsonStr, "John Doe")
	assert.Contains(t, jsonStr, "john@example.com")
}

func TestRESTClient_CustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check custom headers
		assert.Equal(t, "application/xml", r.Header.Get("Content-Type"))
		assert.Equal(t, "custom-value", r.Header.Get("X-Custom-Header"))
		assert.Equal(t, "MyApp/1.0", r.Header.Get("User-Agent"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"ok"}`))
	}))
	defer server.Close()

	client, err := NewRESTClient(server.URL, AuthConfig{Type: NoAuth})
	require.NoError(t, err)

	ctx := context.Background()
	req := RESTRequest{
		Method:   POST,
		Endpoint: "/test",
		Headers: map[string]string{
			"Content-Type":     "application/xml",
			"X-Custom-Header":  "custom-value",
			"User-Agent":       "MyApp/1.0",
		},
		Body: `<test>data</test>`,
	}

	resp, err := client.Execute(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestRESTClient_FormData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		body, _ := io.ReadAll(r.Body)
		bodyStr := string(body)
		assert.Contains(t, bodyStr, "name=John")
		assert.Contains(t, bodyStr, "email=john%40example.com")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"form received"}`))
	}))
	defer server.Close()

	client, err := NewRESTClient(server.URL, AuthConfig{Type: NoAuth})
	require.NoError(t, err)

	ctx := context.Background()
	req := RESTRequest{
		Method:   POST,
		Endpoint: "/form",
		Headers: map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		},
		Body: map[string]interface{}{
			"name":  "John",
			"email": "john@example.com",
		},
	}

	resp, err := client.Execute(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
}

// Benchmark tests
func BenchmarkRESTClient_GET(b *testing.B) {
	server := createTestServer(b)
	defer server.Close()

	client, err := NewRESTClient(server.URL, AuthConfig{Type: NoAuth})
	require.NoError(b, err)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.GET(ctx, "/users/1", nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRESTClient_POST(b *testing.B) {
	server := createTestServer(b)
	defer server.Close()

	client, err := NewRESTClient(server.URL, AuthConfig{Type: NoAuth})
	require.NoError(b, err)

	ctx := context.Background()
	userData := TestUser{
		Name:  "Benchmark User",
		Email: "benchmark@example.com",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.POST(ctx, "/users", userData)
		if err != nil {
			b.Fatal(err)
		}
	}
}