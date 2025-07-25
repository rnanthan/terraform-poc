package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// Config holds all configuration for the REST client
type Config struct {
	BaseURL     string `json:"base_url"`
	Timeout     int    `json:"timeout_seconds"`
	AuthType    string `json:"auth_type"` // "basic", "oauth2", "bearer", "none"

	// Basic Auth
	BasicAuth BasicAuthConfig `json:"basic_auth"`

	// OAuth2
	OAuth2 OAuth2Config `json:"oauth2"`

	// Bearer Token
	BearerToken string `json:"bearer_token"`

	// Default Headers
	DefaultHeaders map[string]string `json:"default_headers"`
}

type BasicAuthConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type OAuth2Config struct {
	ClientID     string            `json:"client_id"`
	ClientSecret string            `json:"client_secret"`
	TokenURL     string            `json:"token_url"`
	Scopes       []string          `json:"scopes"`
	ExtraParams  map[string]string `json:"extra_params"`
}

// RestClient represents the REST client
type RestClient struct {
	config     Config
	httpClient *http.Client
}

// NewRestClient creates a new REST client from config
func NewRestClient(configPath string) (*RestClient, error) {
	config, err := loadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	client := &RestClient{
		config: config,
	}

	// Setup HTTP client based on auth type
	switch strings.ToLower(config.AuthType) {
	case "oauth2":
		client.httpClient, err = client.setupOAuth2Client()
		if err != nil {
			return nil, fmt.Errorf("failed to setup OAuth2 client: %w", err)
		}
	default:
		client.httpClient = &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		}
	}

	return client, nil
}

// loadConfig loads configuration from JSON file or environment variables
func loadConfig(configPath string) (Config, error) {
	var config Config

	// Try to load from file first
	if configPath != "" {
		file, err := os.Open(configPath)
		if err == nil {
			defer file.Close()
			decoder := json.NewDecoder(file)
			if err := decoder.Decode(&config); err != nil {
				return config, fmt.Errorf("failed to decode config file: %w", err)
			}
		}
	}

	// Override with environment variables if present
	if val := os.Getenv("REST_BASE_URL"); val != "" {
		config.BaseURL = val
	}
	if val := os.Getenv("REST_TIMEOUT"); val != "" {
		if timeout, err := strconv.Atoi(val); err == nil {
			config.Timeout = timeout
		}
	}
	if val := os.Getenv("REST_AUTH_TYPE"); val != "" {
		config.AuthType = val
	}
	if val := os.Getenv("REST_BASIC_USERNAME"); val != "" {
		config.BasicAuth.Username = val
	}
	if val := os.Getenv("REST_BASIC_PASSWORD"); val != "" {
		config.BasicAuth.Password = val
	}
	if val := os.Getenv("REST_OAUTH2_CLIENT_ID"); val != "" {
		config.OAuth2.ClientID = val
	}
	if val := os.Getenv("REST_OAUTH2_CLIENT_SECRET"); val != "" {
		config.OAuth2.ClientSecret = val
	}
	if val := os.Getenv("REST_OAUTH2_TOKEN_URL"); val != "" {
		config.OAuth2.TokenURL = val
	}
	if val := os.Getenv("REST_BEARER_TOKEN"); val != "" {
		config.BearerToken = val
	}

	// Set defaults
	if config.Timeout == 0 {
		config.Timeout = 30
	}
	if config.AuthType == "" {
		config.AuthType = "none"
	}

	return config, nil
}

// setupOAuth2Client creates an HTTP client with OAuth2 authentication
func (c *RestClient) setupOAuth2Client() (*http.Client, error) {
	oauthConfig := &clientcredentials.Config{
		ClientID:     c.config.OAuth2.ClientID,
		ClientSecret: c.config.OAuth2.ClientSecret,
		TokenURL:     c.config.OAuth2.TokenURL,
		Scopes:       c.config.OAuth2.Scopes,
	}

	// Add extra parameters if provided
	if len(c.config.OAuth2.ExtraParams) > 0 {
		params := url.Values{}
		for k, v := range c.config.OAuth2.ExtraParams {
			params.Set(k, v)
		}
		oauthConfig.EndpointParams = params
	}

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, &http.Client{
		Timeout: time.Duration(c.config.Timeout) * time.Second,
	})

	return oauthConfig.Client(ctx), nil
}

// Request represents an HTTP request
type Request struct {
	Method  string
	Path    string
	Headers map[string]string
	Body    interface{}
}

// Response represents an HTTP response
type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

// Execute performs the HTTP request
func (c *RestClient) Execute(req Request) (*Response, error) {
	// Build full URL
	fullURL := strings.TrimRight(c.config.BaseURL, "/") + "/" + strings.TrimLeft(req.Path, "/")

	// Prepare request body
	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest(req.Method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set default headers
	for k, v := range c.config.DefaultHeaders {
		httpReq.Header.Set(k, v)
	}

	// Set request-specific headers
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// Set Content-Type for JSON if body is present
	if req.Body != nil && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Apply authentication
	if err := c.applyAuth(httpReq); err != nil {
		return nil, fmt.Errorf("failed to apply authentication: %w", err)
	}

	// Execute request
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &Response{
		StatusCode: httpResp.StatusCode,
		Headers:    httpResp.Header,
		Body:       respBody,
	}, nil
}

// applyAuth applies the configured authentication to the request
func (c *RestClient) applyAuth(req *http.Request) error {
	switch strings.ToLower(c.config.AuthType) {
	case "basic":
		if c.config.BasicAuth.Username == "" || c.config.BasicAuth.Password == "" {
			return fmt.Errorf("basic auth credentials not configured")
		}
		req.SetBasicAuth(c.config.BasicAuth.Username, c.config.BasicAuth.Password)

	case "bearer":
		if c.config.BearerToken == "" {
			return fmt.Errorf("bearer token not configured")
		}
		req.Header.Set("Authorization", "Bearer "+c.config.BearerToken)

	case "oauth2":
		// OAuth2 is handled by the HTTP client itself

	case "none":
		// No authentication

	default:
		return fmt.Errorf("unsupported auth type: %s", c.config.AuthType)
	}

	return nil
}

// Convenience methods for common HTTP methods
func (c *RestClient) Get(path string, headers map[string]string) (*Response, error) {
	return c.Execute(Request{
		Method:  "GET",
		Path:    path,
		Headers: headers,
	})
}

func (c *RestClient) Post(path string, body interface{}, headers map[string]string) (*Response, error) {
	return c.Execute(Request{
		Method:  "POST",
		Path:    path,
		Headers: headers,
		Body:    body,
	})
}

func (c *RestClient) Put(path string, body interface{}, headers map[string]string) (*Response, error) {
	return c.Execute(Request{
		Method:  "PUT",
		Path:    path,
		Headers: headers,
		Body:    body,
	})
}

func (c *RestClient) Delete(path string, headers map[string]string) (*Response, error) {
	return c.Execute(Request{
		Method:  "DELETE",
		Path:    path,
		Headers: headers,
	})
}

// Example usage
func main() {
	// Create client from config file
	client, err := NewRestClient("config.json")
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		return
	}

	// Example GET request
	resp, err := client.Get("/api/users", map[string]string{
		"Accept": "application/json",
	})
	if err != nil {
		fmt.Printf("Error making GET request: %v\n", err)
		return
	}

	fmt.Printf("GET Response Status: %d\n", resp.StatusCode)
	fmt.Printf("GET Response Body: %s\n", string(resp.Body))

	// Example POST request
	postData := map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
	}

	resp, err = client.Post("/api/users", postData, map[string]string{
		"Accept": "application/json",
	})
	if err != nil {
		fmt.Printf("Error making POST request: %v\n", err)
		return
	}

	fmt.Printf("POST Response Status: %d\n", resp.StatusCode)
	fmt.Printf("POST Response Body: %s\n", string(resp.Body))
}