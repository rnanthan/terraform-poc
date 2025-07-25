package restclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// RESTMethod represents HTTP methods for REST operations
type RESTMethod string

const (
	GET    RESTMethod = "GET"
	POST   RESTMethod = "POST"
	PUT    RESTMethod = "PUT"
	DELETE RESTMethod = "DELETE"
	PATCH  RESTMethod = "PATCH"
	HEAD   RESTMethod = "HEAD"
)

// AuthType represents authentication methods
type AuthType string

const (
	NoAuth     AuthType = "none"
	BasicAuth  AuthType = "basic"
	BearerAuth AuthType = "bearer"
	OAuth2Auth AuthType = "oauth2"
	APIKeyAuth AuthType = "apikey"
)

// Authentication configuration
type AuthConfig struct {
	Type AuthType `json:"type"`

	// Basic Authentication
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`

	// Bearer Token
	Token string `json:"token,omitempty"`

	// OAuth2 Configuration
	ClientID     string   `json:"client_id,omitempty"`
	ClientSecret string   `json:"client_secret,omitempty"`
	TokenURL     string   `json:"token_url,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`

	// API Key Configuration
	APIKey    string `json:"api_key,omitempty"`
	KeyHeader string `json:"key_header,omitempty"` // Default: "X-API-Key"
	KeyQuery  string `json:"key_query,omitempty"`  // Alternative: send as query param
}

// REST request configuration
type RESTRequest struct {
	BaseURL     string            `json:"base_url"`
	Endpoint    string            `json:"endpoint"`
	Method      RESTMethod        `json:"method"`
	Headers     map[string]string `json:"headers,omitempty"`
	QueryParams map[string]string `json:"query_params,omitempty"`
	Body        interface{}       `json:"body,omitempty"`
	Timeout     time.Duration     `json:"timeout,omitempty"`
}

// REST response
type RESTResponse struct {
	StatusCode    int                 `json:"status_code"`
	Status        string              `json:"status"`
	Headers       map[string][]string `json:"headers"`
	Body          []byte              `json:"body"`
	ContentType   string              `json:"content_type"`
	ContentLength int64               `json:"content_length"`
	Duration      time.Duration       `json:"duration"`
	URL           string              `json:"url"`
}

// REST client with authentication support
type RESTClient struct {
	httpClient   *http.Client
	auth         AuthConfig
	oauth2Client *http.Client
	baseURL      string
	defaultHeaders map[string]string
}

// NewRESTClient creates a new REST client
func NewRESTClient(baseURL string, auth AuthConfig) (*RESTClient, error) {
	client := &RESTClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		auth:    auth,
		baseURL: strings.TrimSuffix(baseURL, "/"),
		defaultHeaders: map[string]string{
			"Content-Type": "application/json",
			"Accept":       "application/json",
			"User-Agent":   "RESTClient/1.0",
		},
	}

	// Setup OAuth2 if configured
	if auth.Type == OAuth2Auth {
		if err := client.setupOAuth2(); err != nil {
			return nil, fmt.Errorf("failed to setup OAuth2: %w", err)
		}
	}

	return client, nil
}

// setupOAuth2 configures OAuth2 client credentials flow
func (c *RESTClient) setupOAuth2() error {
	if c.auth.ClientID == "" || c.auth.ClientSecret == "" || c.auth.TokenURL == "" {
		return fmt.Errorf("OAuth2 requires client_id, client_secret, and token_url")
	}

	config := &clientcredentials.Config{
		ClientID:     c.auth.ClientID,
		ClientSecret: c.auth.ClientSecret,
		TokenURL:     c.auth.TokenURL,
		Scopes:       c.auth.Scopes,
	}

	c.oauth2Client = config.Client(context.Background())
	return nil
}

// Execute performs REST API call
func (c *RESTClient) Execute(ctx context.Context, req RESTRequest) (*RESTResponse, error) {
	start := time.Now()

	// Build full URL
	fullURL := c.buildURL(req.BaseURL, req.Endpoint, req.QueryParams)

	// Prepare request body
	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := c.marshalRequestBody(req.Body, req.Headers)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, string(req.Method), fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	c.setRequestHeaders(httpReq, req.Headers)

	// Apply authentication
	if err := c.applyAuthentication(httpReq, req.QueryParams); err != nil {
		return nil, fmt.Errorf("failed to apply authentication: %w", err)
	}

	// Select HTTP client
	client := c.selectHTTPClient(req.Timeout)

	// Execute request
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Build response
	response := &RESTResponse{
		StatusCode:    httpResp.StatusCode,
		Status:        httpResp.Status,
		Headers:       httpResp.Header,
		Body:          body,
		ContentType:   httpResp.Header.Get("Content-Type"),
		ContentLength: httpResp.ContentLength,
		Duration:      time.Since(start),
		URL:           fullURL,
	}

	return response, nil
}

// GET performs HTTP GET request
func (c *RESTClient) GET(ctx context.Context, endpoint string, queryParams map[string]string) (*RESTResponse, error) {
	return c.Execute(ctx, RESTRequest{
		Method:      GET,
		Endpoint:    endpoint,
		QueryParams: queryParams,
	})
}

// POST performs HTTP POST request
func (c *RESTClient) POST(ctx context.Context, endpoint string, body interface{}) (*RESTResponse, error) {
	return c.Execute(ctx, RESTRequest{
		Method:   POST,
		Endpoint: endpoint,
		Body:     body,
	})
}

// PUT performs HTTP PUT request
func (c *RESTClient) PUT(ctx context.Context, endpoint string, body interface{}) (*RESTResponse, error) {
	return c.Execute(ctx, RESTRequest{
		Method:   PUT,
		Endpoint: endpoint,
		Body:     body,
	})
}

// PATCH performs HTTP PATCH request
func (c *RESTClient) PATCH(ctx context.Context, endpoint string, body interface{}) (*RESTResponse, error) {
	return c.Execute(ctx, RESTRequest{
		Method:   PATCH,
		Endpoint: endpoint,
		Body:     body,
	})
}

// DELETE performs HTTP DELETE request
func (c *RESTClient) DELETE(ctx context.Context, endpoint string) (*RESTResponse, error) {
	return c.Execute(ctx, RESTRequest{
		Method:   DELETE,
		Endpoint: endpoint,
	})
}

// buildURL constructs the full URL
func (c *RESTClient) buildURL(baseURL, endpoint string, queryParams map[string]string) string {
	// Use provided baseURL or fallback to client's baseURL
	if baseURL == "" {
		baseURL = c.baseURL
	}

	// Build full URL
	fullURL := baseURL
	if endpoint != "" {
		endpoint = strings.TrimPrefix(endpoint, "/")
		fullURL = fmt.Sprintf("%s/%s", baseURL, endpoint)
	}

	// Add query parameters
	if len(queryParams) > 0 {
		u, err := url.Parse(fullURL)
		if err == nil {
			q := u.Query()
			for key, value := range queryParams {
				q.Set(key, value)
			}
			u.RawQuery = q.Encode()
			fullURL = u.String()
		}
	}

	return fullURL
}

// marshalRequestBody converts request body to bytes based on content type
func (c *RESTClient) marshalRequestBody(body interface{}, headers map[string]string) ([]byte, error) {
	if body == nil {
		return nil, nil
	}

	// Check content type
	contentType := headers["Content-Type"]
	if contentType == "" {
		contentType = c.defaultHeaders["Content-Type"]
	}

	switch {
	case strings.Contains(contentType, "application/json"):
		return json.Marshal(body)
	case strings.Contains(contentType, "application/x-www-form-urlencoded"):
		return c.marshalFormData(body)
	case strings.Contains(contentType, "text/plain"):
		if str, ok := body.(string); ok {
			return []byte(str), nil
		}
		return json.Marshal(body)
	default:
		// Default to JSON
		return json.Marshal(body)
	}
}

// marshalFormData converts body to form-encoded data
func (c *RESTClient) marshalFormData(body interface{}) ([]byte, error) {
	values := url.Values{}

	// Convert to map first
	var data map[string]interface{}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(bodyBytes, &data); err != nil {
		return nil, err
	}

	// Add to form values
	for key, value := range data {
		values.Set(key, fmt.Sprintf("%v", value))
	}

	return []byte(values.Encode()), nil
}

// setRequestHeaders sets HTTP headers
func (c *RESTClient) setRequestHeaders(req *http.Request, headers map[string]string) {
	// Set default headers first
	for key, value := range c.defaultHeaders {
		req.Header.Set(key, value)
	}

	// Override with request-specific headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}
}

// applyAuthentication applies the configured authentication
func (c *RESTClient) applyAuthentication(req *http.Request, queryParams map[string]string) error {
	switch c.auth.Type {
	case NoAuth:
		return nil

	case BasicAuth:
		if c.auth.Username == "" {
			return fmt.Errorf("basic auth requires username")
		}
		req.SetBasicAuth(c.auth.Username, c.auth.Password)

	case BearerAuth:
		if c.auth.Token == "" {
			return fmt.Errorf("bearer auth requires token")
		}
		req.Header.Set("Authorization", "Bearer "+c.auth.Token)

	case APIKeyAuth:
		if c.auth.APIKey == "" {
			return fmt.Errorf("API key auth requires api_key")
		}

		// Add as header (default)
		if c.auth.KeyHeader != "" {
			req.Header.Set(c.auth.KeyHeader, c.auth.APIKey)
		} else if c.auth.KeyQuery == "" {
			req.Header.Set("X-API-Key", c.auth.APIKey)
		}

		// Add as query parameter (alternative)
		if c.auth.KeyQuery != "" {
			q := req.URL.Query()
			q.Set(c.auth.KeyQuery, c.auth.APIKey)
			req.URL.RawQuery = q.Encode()
		}

	case OAuth2Auth:
		// OAuth2 is handled by the oauth2Client
		return nil

	default:
		return fmt.Errorf("unsupported authentication type: %s", c.auth.Type)
	}

	return nil
}

// selectHTTPClient returns appropriate HTTP client
func (c *RESTClient) selectHTTPClient(timeout time.Duration) *http.Client {
	if c.oauth2Client != nil {
		client := c.oauth2Client
		if timeout > 0 {
			// Create copy with custom timeout
			return &http.Client{
				Timeout:   timeout,
				Transport: client.Transport,
			}
		}
		return client
	}

	if timeout > 0 {
		return &http.Client{
			Timeout:   timeout,
			Transport: c.httpClient.Transport,
		}
	}

	return c.httpClient
}

// Helper methods for RESTResponse

// IsSuccess checks if the response indicates success (2xx status codes)
func (r *RESTResponse) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// IsClientError checks if the response indicates client error (4xx status codes)
func (r *RESTResponse) IsClientError() bool {
	return r.StatusCode >= 400 && r.StatusCode < 500
}

// IsServerError checks if the response indicates server error (5xx status codes)
func (r *RESTResponse) IsServerError() bool {
	return r.StatusCode >= 500
}

// UnmarshalJSON unmarshals response body into provided interface
func (r *RESTResponse) UnmarshalJSON(v interface{}) error {
	if !strings.Contains(r.ContentType, "application/json") {
		return fmt.Errorf("response content type is not JSON: %s", r.ContentType)
	}
	return json.Unmarshal(r.Body, v)
}

// String returns response body as string
func (r *RESTResponse) String() string {
	return string(r.Body)
}

// JSON returns response body as JSON string (formatted)
func (r *RESTResponse) JSON() (string, error) {
	var jsonData interface{}
	if err := json.Unmarshal(r.Body, &jsonData); err != nil {
		return "", err
	}

	formatted, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		return "", err
	}

	return string(formatted), nil
}