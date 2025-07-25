package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// TestConfig tests configuration loading and validation
func TestConfig(t *testing.T) {
	t.Run("LoadConfigFromFile", func(t *testing.T) {
		// Create temporary config file
		config := Config{
			BaseURL:  "https://test.example.com",
			Timeout:  30,
			AuthType: "basic",
			BasicAuth: BasicAuthConfig{
				Username: "testuser",
				Password: "testpass",
			},
			DefaultHeaders: map[string]string{
				"User-Agent": "TestClient/1.0",
			},
		}

		configData, _ := json.Marshal(config)
		tmpFile := "test_config.json"
		err := os.WriteFile(tmpFile, configData, 0644)
		if err != nil {
			t.Fatalf("Failed to create test config file: %v", err)
		}
		defer os.Remove(tmpFile)

		// Load config
		loadedConfig, err := loadConfig(tmpFile)
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// Verify loaded config
		if loadedConfig.BaseURL != config.BaseURL {
			t.Errorf("Expected BaseURL %s, got %s", config.BaseURL, loadedConfig.BaseURL)
		}
		if loadedConfig.AuthType != config.AuthType {
			t.Errorf("Expected AuthType %s, got %s", config.AuthType, loadedConfig.AuthType)
		}
		if loadedConfig.BasicAuth.Username != config.BasicAuth.Username {
			t.Errorf("Expected Username %s, got %s", config.BasicAuth.Username, loadedConfig.BasicAuth.Username)
		}
	})

	t.Run("LoadConfigFromEnvironment", func(t *testing.T) {
		// Set environment variables
		os.Setenv("REST_BASE_URL", "https://env.example.com")
		os.Setenv("REST_TIMEOUT", "45")
		os.Setenv("REST_AUTH_TYPE", "bearer")
		os.Setenv("REST_BEARER_TOKEN", "env-token-123")

		defer func() {
			os.Unsetenv("REST_BASE_URL")
			os.Unsetenv("REST_TIMEOUT")
			os.Unsetenv("REST_AUTH_TYPE")
			os.Unsetenv("REST_BEARER_TOKEN")
		}()

		// Load config (no file)
		config, err := loadConfig("")
		if err != nil {
			t.Fatalf("Failed to load config from env: %v", err)
		}

		// Verify environment overrides
		if config.BaseURL != "https://env.example.com" {
			t.Errorf("Expected BaseURL from env, got %s", config.BaseURL)
		}
		if config.Timeout != 45 {
			t.Errorf("Expected Timeout 45, got %d", config.Timeout)
		}
		if config.AuthType != "bearer" {
			t.Errorf("Expected AuthType bearer, got %s", config.AuthType)
		}
		if config.BearerToken != "env-token-123" {
			t.Errorf("Expected BearerToken from env, got %s", config.BearerToken)
		}
	})

	t.Run("DefaultValues", func(t *testing.T) {
		config, err := loadConfig("nonexistent.json")
		if err != nil {
			t.Fatalf("Failed to load config with defaults: %v", err)
		}

		if config.Timeout != 30 {
			t.Errorf("Expected default timeout 30, got %d", config.Timeout)
		}
		if config.AuthType != "none" {
			t.Errorf("Expected default auth type 'none', got %s", config.AuthType)
		}
	})
}

// TestRestClientCreation tests REST client creation with different configs
func TestRestClientCreation(t *testing.T) {
	t.Run("CreateWithBasicAuth", func(t *testing.T) {
		config := Config{
			BaseURL:  "https://test.example.com",
			Timeout:  30,
			AuthType: "basic",
			BasicAuth: BasicAuthConfig{
				Username: "testuser",
				Password: "testpass",
			},
		}

		configData, _ := json.Marshal(config)
		tmpFile := "test_basic_config.json"
		os.WriteFile(tmpFile, configData, 0644)
		defer os.Remove(tmpFile)

		client, err := NewRestClient(tmpFile)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		if client.config.AuthType != "basic" {
			t.Errorf("Expected basic auth, got %s", client.config.AuthType)
		}
		if client.httpClient == nil {
			t.Error("HTTP client should not be nil")
		}
	})

	t.Run("CreateWithBearerAuth", func(t *testing.T) {
		config := Config{
			BaseURL:     "https://test.example.com",
			Timeout:     30,
			AuthType:    "bearer",
			BearerToken: "test-token-123",
		}

		configData, _ := json.Marshal(config)
		tmpFile := "test_bearer_config.json"
		os.WriteFile(tmpFile, configData, 0644)
		defer os.Remove(tmpFile)

		client, err := NewRestClient(tmpFile)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		if client.config.AuthType != "bearer" {
			t.Errorf("Expected bearer auth, got %s", client.config.AuthType)
		}
		if client.config.BearerToken != "test-token-123" {
			t.Errorf("Expected bearer token, got %s", client.config.BearerToken)
		}
	})

	t.Run("CreateWithNoAuth", func(t *testing.T) {
		config := Config{
			BaseURL:  "https://test.example.com",
			Timeout:  30,
			AuthType: "none",
		}

		configData, _ := json.Marshal(config)
		tmpFile := "test_none_config.json"
		os.WriteFile(tmpFile, configData, 0644)
		defer os.Remove(tmpFile)

		client, err := NewRestClient(tmpFile)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		if client.config.AuthType != "none" {
			t.Errorf("Expected no auth, got %s", client.config.AuthType)
		}
	})
}

// TestHTTPMethods tests all HTTP methods with mock server
func TestHTTPMethods(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Echo back request details
		response := map[string]interface{}{
			"method":  r.Method,
			"path":    r.URL.Path,
			"headers": r.Header,
		}

		// Read body for POST/PUT requests
		if r.Method == "POST" || r.Method == "PUT" {
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			response["body"] = body
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	config := Config{
		BaseURL:  server.URL,
		Timeout:  30,
		AuthType: "none",
		DefaultHeaders: map[string]string{
			"User-Agent": "TestClient/1.0",
		},
	}

	configData, _ := json.Marshal(config)
	tmpFile := "test_http_config.json"
	os.WriteFile(tmpFile, configData, 0644)
	defer os.Remove(tmpFile)

	client, err := NewRestClient(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Run("GET", func(t *testing.T) {
		resp, err := client.Get("/test", map[string]string{
			"X-Test-Header": "test-value",
		})
		if err != nil {
			t.Fatalf("GET request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var responseData map[string]interface{}
		json.Unmarshal(resp.Body, &responseData)

		if responseData["method"] != "GET" {
			t.Errorf("Expected method GET, got %v", responseData["method"])
		}
		if responseData["path"] != "/test" {
			t.Errorf("Expected path /test, got %v", responseData["path"])
		}
	})

	t.Run("POST", func(t *testing.T) {
		postData := map[string]interface{}{
			"name":  "John Doe",
			"email": "john@example.com",
		}

		resp, err := client.Post("/users", postData, nil)
		if err != nil {
			t.Fatalf("POST request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var responseData map[string]interface{}
		json.Unmarshal(resp.Body, &responseData)

		if responseData["method"] != "POST" {
			t.Errorf("Expected method POST, got %v", responseData["method"])
		}

		body, ok := responseData["body"].(map[string]interface{})
		if !ok {
			t.Error("Expected body in response")
		} else {
			if body["name"] != "John Doe" {
				t.Errorf("Expected name 'John Doe', got %v", body["name"])
			}
		}
	})

	t.Run("PUT", func(t *testing.T) {
		putData := map[string]interface{}{
			"id":   123,
			"name": "Updated Name",
		}

		resp, err := client.Put("/users/123", putData, nil)
		if err != nil {
			t.Fatalf("PUT request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var responseData map[string]interface{}
		json.Unmarshal(resp.Body, &responseData)

		if responseData["method"] != "PUT" {
			t.Errorf("Expected method PUT, got %v", responseData["method"])
		}
	})

	t.Run("DELETE", func(t *testing.T) {
		resp, err := client.Delete("/users/123", nil)
		if err != nil {
			t.Fatalf("DELETE request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var responseData map[string]interface{}
		json.Unmarshal(resp.Body, &responseData)

		if responseData["method"] != "DELETE" {
			t.Errorf("Expected method DELETE, got %v", responseData["method"])
		}
	})
}

// TestAuthentication tests different authentication methods
func TestAuthentication(t *testing.T) {
	t.Run("BasicAuth", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()
			if !ok {
				w.WriteHeader(401)
				return
			}

			response := map[string]string{
				"username": username,
				"password": password,
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		config := Config{
			BaseURL:  server.URL,
			Timeout:  30,
			AuthType: "basic",
			BasicAuth: BasicAuthConfig{
				Username: "testuser",
				Password: "testpass",
			},
		}

		configData, _ := json.Marshal(config)
		tmpFile := "test_basic_auth_config.json"
		os.WriteFile(tmpFile, configData, 0644)
		defer os.Remove(tmpFile)

		client, err := NewRestClient(tmpFile)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		resp, err := client.Get("/protected", nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var responseData map[string]string
		json.Unmarshal(resp.Body, &responseData)

		if responseData["username"] != "testuser" {
			t.Errorf("Expected username 'testuser', got '%s'", responseData["username"])
		}
		if responseData["password"] != "testpass" {
			t.Errorf("Expected password 'testpass', got '%s'", responseData["password"])
		}
	})

	t.Run("BearerAuth", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				w.WriteHeader(401)
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			response := map[string]string{
				"token": token,
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		config := Config{
			BaseURL:     server.URL,
			Timeout:     30,
			AuthType:    "bearer",
			BearerToken: "test-token-123",
		}

		configData, _ := json.Marshal(config)
		tmpFile := "test_bearer_auth_config.json"
		os.WriteFile(tmpFile, configData, 0644)
		defer os.Remove(tmpFile)

		client, err := NewRestClient(tmpFile)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		resp, err := client.Get("/protected", nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var responseData map[string]string
		json.Unmarshal(resp.Body, &responseData)

		if responseData["token"] != "test-token-123" {
			t.Errorf("Expected token 'test-token-123', got '%s'", responseData["token"])
		}
	})

	t.Run("NoAuth", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			response := map[string]string{
				"auth_header": authHeader,
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		config := Config{
			BaseURL:  server.URL,
			Timeout:  30,
			AuthType: "none",
		}

		configData, _ := json.Marshal(config)
		tmpFile := "test_no_auth_config.json"
		os.WriteFile(tmpFile, configData, 0644)
		defer os.Remove(tmpFile)

		client, err := NewRestClient(tmpFile)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		resp, err := client.Get("/public", nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		var responseData map[string]string
		json.Unmarshal(resp.Body, &responseData)

		if responseData["auth_header"] != "" {
			t.Errorf("Expected no auth header, got '%s'", responseData["auth_header"])
		}
	})
}

// TestHeaders tests default and custom headers
func TestHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers := make(map[string]string)
		for name, values := range r.Header {
			if len(values) > 0 {
				headers[name] = values[0]
			}
		}
		json.NewEncoder(w).Encode(headers)
	}))
	defer server.Close()

	config := Config{
		BaseURL:  server.URL,
		Timeout:  30,
		AuthType: "none",
		DefaultHeaders: map[string]string{
			"User-Agent":     "TestClient/1.0",
			"X-API-Version":  "v1",
			"Content-Type":   "application/json",
		},
	}

	configData, _ := json.Marshal(config)
	tmpFile := "test_headers_config.json"
	os.WriteFile(tmpFile, configData, 0644)
	defer os.Remove(tmpFile)

	client, err := NewRestClient(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Run("DefaultHeaders", func(t *testing.T) {
		resp, err := client.Get("/test", nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		var headers map[string]string
		json.Unmarshal(resp.Body, &headers)

		if headers["User-Agent"] != "TestClient/1.0" {
			t.Errorf("Expected User-Agent 'TestClient/1.0', got '%s'", headers["User-Agent"])
		}
		if headers["X-Api-Version"] != "v1" {
			t.Errorf("Expected X-API-Version 'v1', got '%s'", headers["X-Api-Version"])
		}
	})

	t.Run("CustomHeaders", func(t *testing.T) {
		customHeaders := map[string]string{
			"X-Custom-Header": "custom-value",
			"X-Request-ID":    "req-123",
		}

		resp, err := client.Get("/test", customHeaders)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		var headers map[string]string
		json.Unmarshal(resp.Body, &headers)

		if headers["X-Custom-Header"] != "custom-value" {
			t.Errorf("Expected X-Custom-Header 'custom-value', got '%s'", headers["X-Custom-Header"])
		}
		if headers["X-Request-Id"] != "req-123" {
			t.Errorf("Expected X-Request-ID 'req-123', got '%s'", headers["X-Request-Id"])
		}

		// Default headers should still be present
		if headers["User-Agent"] != "TestClient/1.0" {
			t.Errorf("Expected User-Agent 'TestClient/1.0', got '%s'", headers["User-Agent"])
		}
	})

	t.Run("HeaderOverride", func(t *testing.T) {
		overrideHeaders := map[string]string{
			"User-Agent": "OverrideClient/2.0",
		}

		resp, err := client.Get("/test", overrideHeaders)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		var headers map[string]string
		json.Unmarshal(resp.Body, &headers)

		// Request-specific header should override default
		if headers["User-Agent"] != "OverrideClient/2.0" {
			t.Errorf("Expected overridden User-Agent 'OverrideClient/2.0', got '%s'", headers["User-Agent"])
		}
	})
}

// TestErrorHandling tests error scenarios
func TestErrorHandling(t *testing.T) {
	t.Run("InvalidConfig", func(t *testing.T) {
		// Write invalid JSON to config file
		tmpFile := "invalid_config.json"
		os.WriteFile(tmpFile, []byte(`{invalid json`), 0644)
		defer os.Remove(tmpFile)

		_, err := NewRestClient(tmpFile)
		if err == nil {
			t.Error("Expected error for invalid config, got nil")
		}
	})

	t.Run("ServerError", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
			w.Write([]byte("Internal Server Error"))
		}))
		defer server.Close()

		config := Config{
			BaseURL:  server.URL,
			Timeout:  30,
			AuthType: "none",
		}

		configData, _ := json.Marshal(config)
		tmpFile := "test_error_config.json"
		os.WriteFile(tmpFile, configData, 0644)
		defer os.Remove(tmpFile)

		client, err := NewRestClient(tmpFile)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		resp, err := client.Get("/error", nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if resp.StatusCode != 500 {
			t.Errorf("Expected status 500, got %d", resp.StatusCode)
		}

		if string(resp.Body) != "Internal Server Error" {
			t.Errorf("Expected error message, got '%s'", string(resp.Body))
		}
	})

	t.Run("Timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(2 * time.Second) // Longer than client timeout
			w.WriteHeader(200)
		}))
		defer server.Close()

		config := Config{
			BaseURL:  server.URL,
			Timeout:  1, // 1 second timeout
			AuthType: "none",
		}

		configData, _ := json.Marshal(config)
		tmpFile := "test_timeout_config.json"
		os.WriteFile(tmpFile, configData, 0644)
		defer os.Remove(tmpFile)

		client, err := NewRestClient(tmpFile)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		_, err = client.Get("/slow", nil)
		if err == nil {
			t.Error("Expected timeout error, got nil")
		}

		if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "context deadline exceeded") {
			t.Errorf("Expected timeout error, got: %v", err)
		}
	})

	t.Run("MissingBasicAuthCredentials", func(t *testing.T) {
		config := Config{
			BaseURL:  "https://test.example.com",
			Timeout:  30,
			AuthType: "basic",
			// Missing BasicAuth config
		}

		configData, _ := json.Marshal(config)
		tmpFile := "test_missing_auth_config.json"
		os.WriteFile(tmpFile, configData, 0644)
		defer os.Remove(tmpFile)

		client, err := NewRestClient(tmpFile)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		// Try to make a request (should fail during auth application)
		req := Request{
			Method: "GET",
			Path:   "/test",
		}

		_, err = client.Execute(req)
		if err == nil {
			t.Error("Expected error for missing basic auth credentials")
		}

		if !strings.Contains(err.Error(), "basic auth credentials not configured") {
			t.Errorf("Expected basic auth error, got: %v", err)
		}
	})

	t.Run("MissingBearerToken", func(t *testing.T) {
		config := Config{
			BaseURL:  "https://test.example.com",
			Timeout:  30,
			AuthType: "bearer",
			// Missing BearerToken
		}

		configData, _ := json.Marshal(config)
		tmpFile := "test_missing_bearer_config.json"
		os.WriteFile(tmpFile, configData, 0644)
		defer os.Remove(tmpFile)

		client, err := NewRestClient(tmpFile)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		req := Request{
			Method: "GET",
			Path:   "/test",
		}

		_, err = client.Execute(req)
		if err == nil {
			t.Error("Expected error for missing bearer token")
		}

		if !strings.Contains(err.Error(), "bearer token not configured") {
			t.Errorf("Expected bearer token error, got: %v", err)
		}
	})

	t.Run("UnsupportedAuthType", func(t *testing.T) {
		config := Config{
			BaseURL:  "https://test.example.com",
			Timeout:  30,
			AuthType: "unsupported",
		}

		configData, _ := json.Marshal(config)
		tmpFile := "test_unsupported_auth_config.json"
		os.WriteFile(tmpFile, configData, 0644)
		defer os.Remove(tmpFile)

		client, err := NewRestClient(tmpFile)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		req := Request{
			Method: "GET",
			Path:   "/test",
		}

		_, err = client.Execute(req)
		if err == nil {
			t.Error("Expected error for unsupported auth type")
		}

		if !strings.Contains(err.Error(), "unsupported auth type") {
			t.Errorf("Expected unsupported auth type error, got: %v", err)
		}
	})
}

// TestJSONHandling tests JSON marshaling and unmarshaling
func TestJSONHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requestBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&requestBody)

		response := map[string]interface{}{
			"received": requestBody,
			"status":   "success",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := Config{
		BaseURL:  server.URL,
		Timeout:  30,
		AuthType: "none",
	}

	configData, _ := json.Marshal(config)
	tmpFile := "test_json_config.json"
	os.WriteFile(tmpFile, configData, 0644)
	defer os.Remove(tmpFile)

	client, err := NewRestClient(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Run("ValidJSON", func(t *testing.T) {
		testData := map[string]interface{}{
			"name":    "John Doe",
			"age":     30,
			"active":  true,
			"scores":  []int{85, 92, 78},
			"address": map[string]string{
				"street": "123 Main St",
				"city":   "Springfield",
			},
		}

		resp, err := client.Post("/data", testData, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var responseData map[string]interface{}
		json.Unmarshal(resp.Body, &responseData)

		received, ok := responseData["received"].(map[string]interface{})
		if !ok {
			t.Error("Expected received data in response")
		} else {
			if received["name"] != "John Doe" {
				t.Errorf("Expected name 'John Doe', got %v", received["name"])
			}
			if received["age"].(float64) != 30 {
				t.Errorf("Expected age 30, got %v", received["age"])
			}
		}
	})

	t.Run("InvalidJSONInput", func(t *testing.T) {
		// Create a struct that can't be marshaled to JSON
		type invalidStruct struct {
			Chan chan int `json:"chan"`
		}

		invalidData := invalidStruct{
			Chan: make(chan int),
		}

		req := Request{
			Method: "POST",
			Path:   "/data",
			Body:   invalidData,
		}

		_, err := client.Execute(req)
		if err == nil {
			t.Error("Expected error for invalid JSON input")
		}

		if !strings.Contains(err.Error(), "failed to marshal request body") {
			t.Errorf("Expected JSON marshal error, got: %v", err)
		}
	})
}

// BenchmarkRestClient benchmarks the REST client performance
func BenchmarkRestClient(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]string{
			"status": "ok",
			"method": r.Method,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := Config{
		BaseURL:  server.URL,
		Timeout:  30,
		AuthType: "none",
	}

	configData, _ := json.Marshal(config)
	tmpFile := "bench_config.json"
	os.WriteFile(tmpFile, configData, 0644)
	defer os.Remove(tmpFile)

	client, err := NewRestClient(