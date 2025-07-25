// production_examples.go - Real-world production examples

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// Example 1: Slack API Integration with OAuth2
func exampleSlackIntegration() {
	fmt.Println("=== Slack API Integration Example ===")

	// Slack configuration
	config := Config{
		BaseURL:  "https://slack.com/api",
		Timeout:  30,
		AuthType: "bearer",
		BearerToken: "xoxb-your-bot-token-here", // Replace with actual token
		DefaultHeaders: map[string]string{
			"Content-Type": "application/json",
			"User-Agent":   "MyApp/1.0",
		},
	}

	// Save and create client
	configData, _ := json.MarshalIndent(config, "", "  ")
	fmt.Printf("Slack Config:\n%s\n", string(configData))
	fmt.Println("(Replace bearer_token with actual Slack bot token)")
}

// Example 2: GitHub API with Personal Access Token
func exampleGitHubIntegration() {
	fmt.Println("\n=== GitHub API Integration Example ===")

	config := Config{
		BaseURL:  "https://api.github.com",
		Timeout:  30,
		AuthType: "bearer",
		BearerToken: "ghp_your_personal_access_token", // Replace with actual token
		DefaultHeaders: map[string]string{
			"Accept":       "application/vnd.github.v3+json",
			"User-Agent":   "MyApp-GitHub-Client/1.0",
			"X-GitHub-Api-Version": "2022-11-28",
		},
	}

	configData, _ := json.MarshalIndent(config, "", "  ")
	fmt.Printf("GitHub Config:\n%s\n", string(configData))

	// Example usage code
	usageCode := `
// Usage example:
client, _ := NewRestClient("github-config.json")

// Get user info
resp, err := client.Get("/user", nil)

// List repositories
resp, err = client.Get("/user/repos", map[string]string{
    "sort": "updated",
    "per_page": "10",
})

// Create an issue
issue := map[string]interface{}{
    "title": "Bug found",
    "body": "Description of the bug",
    "labels": []string{"bug", "priority-high"},
}
resp, err = client.Post("/repos/owner/repo/issues", issue, nil)
`
	fmt.Println(usageCode)
}

// Example 3: AWS API Gateway with API Key
func exampleAWSAPIGateway() {
	fmt.Println("\n=== AWS API Gateway Example ===")

	config := Config{
		BaseURL:  "https://your-api-id.execute-api.region.amazonaws.com/stage",
		Timeout:  30,
		AuthType: "none", // Using API Key in headers instead
		DefaultHeaders: map[string]string{
			"x-api-key":    "your-api-key-here",
			"Content-Type": "application/json",
			"User-Agent":   "MyApp-AWS-Client/1.0",
		},
	}

	configData, _ := json.MarshalIndent(config, "", "  ")
	fmt.Printf("AWS API Gateway Config:\n%s\n", string(configData))
}

// Example 4: Microservice Communication Pattern
type UserService struct {
	client *RestClient
}

type OrderService struct {
	client *RestClient
}

type NotificationService struct {
	client *RestClient
}

func exampleMicroservicePattern() {
	fmt.Println("\n=== Microservice Communication Pattern ===")

	// User Service Client
	userConfig := Config{
		BaseURL:  "http://user-service:8080",
		Timeout:  10,
		AuthType: "bearer",
		BearerToken: "service-to-service-token",
		DefaultHeaders: map[string]string{
			"Service-Name": "api-gateway",
			"Content-Type": "application/json",
		},
	}

	// Order Service Client
	orderConfig := Config{
		BaseURL:  "http://order-service:8080",
		Timeout:  15,
		AuthType: "bearer",
		BearerToken: "service-to-service-token",
		DefaultHeaders: map[string]string{
			"Service-Name": "api-gateway",
			"Content-Type": "application/json",
		},
	}

	fmt.Printf("Microservice configs ready for:\n- User Service\n- Order Service\n- Notification Service\n")
}

// Example 5: Rate Limiting and Retry Logic
type RateLimitedClient struct {
	client    *RestClient
	rateLimit chan struct{}
	mu        sync.Mutex
}

func NewRateLimitedClient(client *RestClient, requestsPerSecond int) *RateLimitedClient {
	rateLimitChan := make(chan struct{}, requestsPerSecond)

	// Fill the channel initially
	for i := 0; i < requestsPerSecond; i++ {
		rateLimitChan <- struct{}{}
	}

	// Refill the channel every second
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for range ticker.C {
			for i := 0; i < requestsPerSecond; i++ {
				select {
				case rateLimitChan <- struct{}{}:
				default:
					// Channel is full, skip
				}
			}
		}
	}()

	return &RateLimitedClient{
		client:    client,
		rateLimit: rateLimitChan,
	}
}

func (rlc *RateLimitedClient) Get(path string, headers map[string]string) (*Response, error) {
	// Wait for rate limit token
	<-rlc.rateLimit

	return rlc.client.Get(path, headers)
}

func (rlc *RateLimitedClient) GetWithRetry(path string, headers map[string]string, maxRetries int) (*Response, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			time.Sleep(time.Duration(attempt*attempt) * time.Second)
		}

		resp, err := rlc.Get(path, headers)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		// Don't retry on client errors (4xx)
		if resp != nil && resp.StatusCode >= 400 && resp.StatusCode < 500 {
			break
		}
	}

	return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

func exampleRateLimitingAndRetry() {
	fmt.Println("\n=== Rate Limiting and Retry Example ===")

	// Create base client
	config := Config{
		BaseURL:  "https://jsonplaceholder.typicode.com",
		Timeout:  30,
		AuthType: "none",
	}

	// Simulate creating client (in real usage, you'd load from config)
	fmt.Printf("Rate-limited client configured for 10 requests/second with retry logic\n")

	// Example usage code
	usageCode := `
// Usage:
baseClient, _ := NewRestClient("config.json")
rateLimitedClient := NewRateLimitedClient(baseClient, 10) // 10 requests per second

// Make request with retry
resp, err := rateLimitedClient.GetWithRetry("/posts/1", nil, 3)
`
	fmt.Println(usageCode)
}

// Example 6: Context and Timeout Management
func exampleContextManagement() {
	fmt.Println("\n=== Context and Timeout Management ===")

	// Example showing how to add context support to the client
	contextCode := `
// Enhanced client with context support
func (c *RestClient) ExecuteWithContext(ctx context.Context, req Request) (*Response, error) {
    // Build HTTP request as before...
    httpReq, err := http.NewRequestWithContext(ctx, req.Method, fullURL, bodyReader)
    if err != nil {
        return nil, err
    }

    // Rest of the implementation...
    return c.httpClient.Do(httpReq)
}

// Usage with timeout context
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

resp, err := client.ExecuteWithContext(ctx, Request{
    Method: "GET",
    Path:   "/slow-endpoint",
})
`
	fmt.Println(contextCode)
}

// Example 7: Configuration Management for Different Environments
func exampleEnvironmentConfigs() {
	fmt.Println("\n=== Environment Configuration Management ===")

	environments := map[string]Config{
		"development": {
			BaseURL:  "http://localhost:8080",
			Timeout:  60,
			AuthType: "none",
			DefaultHeaders: map[string]string{
				"Environment": "development",
				"Debug":       "true",
			},
		},
		"staging": {
			BaseURL:  "https://staging-api.yourcompany.com",
			Timeout:  30,
			AuthType: "bearer",
			BearerToken: "staging-bearer-token",
			DefaultHeaders: map[string]string{
				"Environment": "staging",
			},
		},
		"production": {
			BaseURL:  "https://api.yourcompany.com",
			Timeout:  15,
			AuthType: "oauth2",
			OAuth2: OAuth2Config{
				ClientID:     "prod-client-id",
				ClientSecret: "prod-client-secret",
				TokenURL:     "https://auth.yourcompany.com/token",
				Scopes:       []string{"api:read", "api:write"},
			},
			DefaultHeaders: map[string]string{
				"Environment": "production",
			},
		},
	}

	for env, config := range environments {
		configData, _ := json.MarshalIndent(config, "", "  ")
		fmt.Printf("\n%s Config:\n%s\n", env, string(configData))
	}
}

// Example 8: Structured Logging Integration
type LoggingClient struct {
	client *RestClient
	logger *log.Logger
}

func NewLoggingClient(client *RestClient, logger *log.Logger) *LoggingClient {
	return &LoggingClient{
		client: client,
		logger: logger,
	}
}

func (lc *LoggingClient) Get(path string, headers map[string]string) (*Response, error) {
	start := time.Now()
	lc.logger.Printf("Starting GET request to %s", path)

	resp, err := lc.client.Get(path, headers)

	duration := time.Since(start)
	if err != nil {
		lc.logger.Printf("GET request to %s failed after %v: %v", path, duration, err)
		return nil, err
	}

	lc.logger.Printf("GET request to %s completed in %v with status %d",
		path, duration, resp.StatusCode)

	return resp, nil
}

func exampleStructuredLogging() {
	fmt.Println("\n=== Structured Logging Example ===")

	loggingCode := `
// Create logger
logger := log.New(os.Stdout, "API_CLIENT: ", log.LstdFlags|log.Lshortfile)

// Create base client
baseClient, _ := NewRestClient("config.json")

// Wrap with logging
loggingClient := NewLoggingClient(baseClient, logger)

// All requests are now logged
resp, err := loggingClient.Get("/api/users", nil)
`
	fmt.Println(loggingCode)
}

// Example 9: Health Check and Circuit Breaker Pattern
type HealthChecker struct {
	client         *RestClient
	healthEndpoint string
	isHealthy      bool
	mu             sync.RWMutex
}

func NewHealthChecker(client *RestClient, healthEndpoint string) *HealthChecker {
	hc := &HealthChecker{
		client:         client,
		healthEndpoint: healthEndpoint,
		isHealthy:      true,
	}

	// Start health checking goroutine
	go hc.startHealthChecking()

	return hc
}

func (hc *HealthChecker) startHealthChecking() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		hc.checkHealth()
	}
}

func (hc *HealthChecker) checkHealth() {
	resp, err := hc.client.Get(hc.healthEndpoint, nil)

	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.isHealthy = err == nil && resp.StatusCode == 200
}

func (hc *HealthChecker) IsHealthy() bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.isHealthy
}

func example