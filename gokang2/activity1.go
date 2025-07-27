package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/log"

	"myproject/restclient" // Replace with your actual module path
)

// RESTServiceRequest represents input for REST service activities
type RESTServiceRequest struct {
	ServiceName string                     `json:"service_name"`
	BaseURL     string                     `json:"base_url"`
	Auth        restclient.AuthConfig      `json:"auth"`
	Request     restclient.RESTRequest     `json:"request"`
	Retry       *RetryConfig               `json:"retry,omitempty"`
	Timeout     time.Duration              `json:"timeout,omitempty"`
}

// RESTServiceResponse represents output from REST service activities
type RESTServiceResponse struct {
	ServiceName   string                  `json:"service_name"`
	StatusCode    int                     `json:"status_code"`
	Status        string                  `json:"status"`
	Headers       map[string][]string     `json:"headers"`
	Body          string                  `json:"body"`
	ContentType   string                  `json:"content_type"`
	Duration      time.Duration           `json:"duration"`
	URL           string                  `json:"url"`
	Success       bool                    `json:"success"`
	ErrorMessage  string                  `json:"error_message,omitempty"`
	Retries       int                     `json:"retries,omitempty"`
}

// RetryConfig defines retry behavior for REST calls
type RetryConfig struct {
	MaxAttempts        int           `json:"max_attempts"`
	InitialBackoff     time.Duration `json:"initial_backoff"`
	BackoffMultiplier  float64       `json:"backoff_multiplier"`
	MaxBackoff         time.Duration `json:"max_backoff"`
	RetryableStatusCodes []int       `json:"retryable_status_codes,omitempty"` // Default: 5xx errors
}

// RESTServiceActivities contains REST service related activities
type RESTServiceActivities struct {
	logger log.Logger
}

// NewRESTServiceActivities creates new instance of REST service activities
func NewRESTServiceActivities(logger log.Logger) *RESTServiceActivities {
	return &RESTServiceActivities{
		logger: logger,
	}
}

// InvokeRESTService executes a REST API call
func (a *RESTServiceActivities) InvokeRESTService(ctx context.Context, req RESTServiceRequest) (*RESTServiceResponse, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Invoking REST service",
		"service", req.ServiceName,
		"method", req.Request.Method,
		"endpoint", req.Request.Endpoint)

	// Create REST client
	client, err := restclient.NewRESTClient(req.BaseURL, req.Auth)
	if err != nil {
		logger.Error("Failed to create REST client", "error", err)
		return &RESTServiceResponse{
			ServiceName:  req.ServiceName,
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to create REST client: %v", err),
		}, err
	}

	// Set timeout if specified
	if req.Timeout > 0 {
		req.Request.Timeout = req.Timeout
	}

	// Execute REST call
	resp, err := client.Execute(ctx, req.Request)
	if err != nil {
		logger.Error("REST call failed", "error", err)
		return &RESTServiceResponse{
			ServiceName:  req.ServiceName,
			Success:      false,
			ErrorMessage: fmt.Sprintf("REST call failed: %v", err),
		}, err
	}

	// Build response
	result := &RESTServiceResponse{
		ServiceName: req.ServiceName,
		StatusCode:  resp.StatusCode,
		Status:      resp.Status,
		Headers:     resp.Headers,
		Body:        string(resp.Body),
		ContentType: resp.ContentType,
		Duration:    resp.Duration,
		URL:         resp.URL,
		Success:     resp.IsSuccess(),
	}

	if !result.Success {
		result.ErrorMessage = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status)
		logger.Warn("REST service call failed",
			"service", req.ServiceName,
			"status_code", resp.StatusCode,
			"status", resp.Status)
	} else {
		logger.Info("REST service call successful",
			"service", req.ServiceName,
			"status_code", resp.StatusCode,
			"duration", resp.Duration)
	}

	return result, nil
}

// InvokeRESTServiceWithRetry executes REST API call with retry logic
func (a *RESTServiceActivities) InvokeRESTServiceWithRetry(ctx context.Context, req RESTServiceRequest) (*RESTServiceResponse, error) {
	logger := activity.GetLogger(ctx)

	// Set default retry config
	retryConfig := &RetryConfig{
		MaxAttempts:          3,
		InitialBackoff:       1 * time.Second,
		BackoffMultiplier:    2.0,
		MaxBackoff:           30 * time.Second,
		RetryableStatusCodes: []int{500, 502, 503, 504, 429}, // Server errors and rate limiting
	}

	if req.Retry != nil {
		if req.Retry.MaxAttempts > 0 {
			retryConfig.MaxAttempts = req.Retry.MaxAttempts
		}
		if req.Retry.InitialBackoff > 0 {
			retryConfig.InitialBackoff = req.Retry.InitialBackoff
		}
		if req.Retry.BackoffMultiplier > 0 {
			retryConfig.BackoffMultiplier = req.Retry.BackoffMultiplier
		}
		if req.Retry.MaxBackoff > 0 {
			retryConfig.MaxBackoff = req.Retry.MaxBackoff
		}
		if len(req.Retry.RetryableStatusCodes) > 0 {
			retryConfig.RetryableStatusCodes = req.Retry.RetryableStatusCodes
		}
	}

	logger.Info("Invoking REST service with retry",
		"service", req.ServiceName,
		"max_attempts", retryConfig.MaxAttempts,
		"initial_backoff", retryConfig.InitialBackoff)

	var lastResponse *RESTServiceResponse
	var lastError error
	backoff := retryConfig.InitialBackoff

	for attempt := 1; attempt <= retryConfig.MaxAttempts; attempt++ {
		logger.Info("REST service attempt",
			"service", req.ServiceName,
			"attempt", attempt,
			"of", retryConfig.MaxAttempts)

		// Execute the request
		resp, err := a.InvokeRESTService(ctx, req)

		if err == nil && resp.Success {
			resp.Retries = attempt - 1
			logger.Info("REST service call successful",
				"service", req.ServiceName,
				"attempt", attempt)
			return resp, nil
		}

		// Check if error is retryable
		if err == nil && resp != nil && !a.isRetryableStatus(resp.StatusCode, retryConfig.RetryableStatusCodes) {
			logger.Warn("Non-retryable error, stopping",
				"service", req.ServiceName,
				"status_code", resp.StatusCode)
			resp.Retries = attempt - 1
			return resp, nil
		}

		lastResponse = resp
		lastError = err

		// Don't sleep after the last attempt
		if attempt < retryConfig.MaxAttempts {
			logger.Warn("Attempt failed, retrying",
				"service", req.ServiceName,
				"attempt", attempt,
				"backoff", backoff)

			select {
			case <-time.After(backoff):
				// Continue to next attempt
			case <-ctx.Done():
				return nil, ctx.Err()
			}

			// Calculate next backoff with cap
			backoff = time.Duration(float64(backoff) * retryConfig.BackoffMultiplier)
			if backoff > retryConfig.MaxBackoff {
				backoff = retryConfig.MaxBackoff
			}
		}
	}

	logger.Error("All retry attempts failed",
		"service", req.ServiceName,
		"attempts", retryConfig.MaxAttempts)

	if lastResponse != nil {
		lastResponse.ErrorMessage = fmt.Sprintf("All %d attempts failed. Last error: %s",
			retryConfig.MaxAttempts, lastResponse.ErrorMessage)
		lastResponse.Retries = retryConfig.MaxAttempts - 1
		return lastResponse, fmt.Errorf("all retry attempts failed")
	}

	return &RESTServiceResponse{
		ServiceName:  req.ServiceName,
		Success:      false,
		ErrorMessage: fmt.Sprintf("All %d attempts failed. Last error: %v", retryConfig.MaxAttempts, lastError),
		Retries:      retryConfig.MaxAttempts - 1,
	}, lastError
}

// GetResource performs HTTP GET operation
func (a *RESTServiceActivities) GetResource(ctx context.Context, serviceName, baseURL, endpoint string, auth restclient.AuthConfig, queryParams map[string]string) (*RESTServiceResponse, error) {
	req := RESTServiceRequest{
		ServiceName: serviceName,
		BaseURL:     baseURL,
		Auth:        auth,
		Request: restclient.RESTRequest{
			Method:      restclient.GET,
			Endpoint:    endpoint,
			QueryParams: queryParams,
		},
	}

	return a.InvokeRESTService(ctx, req)
}

// CreateResource performs HTTP POST operation
func (a *RESTServiceActivities) CreateResource(ctx context.Context, serviceName, baseURL, endpoint string, auth restclient.AuthConfig, body interface{}) (*RESTServiceResponse, error) {
	req := RESTServiceRequest{
		ServiceName: serviceName,
		BaseURL:     baseURL,
		Auth:        auth,
		Request: restclient.RESTRequest{
			Method:   restclient.POST,
			Endpoint: endpoint,
			Body:     body,
		},
	}

	return a.InvokeRESTService(ctx, req)
}

// UpdateResource performs HTTP PUT operation
func (a *RESTServiceActivities) UpdateResource(ctx context.Context, serviceName, baseURL, endpoint string, auth restclient.AuthConfig, body interface{}) (*RESTServiceResponse, error) {
	req := RESTServiceRequest{
		ServiceName: serviceName,
		BaseURL:     baseURL,
		Auth:        auth,
		Request: restclient.RESTRequest{
			Method:   restclient.PUT,
			Endpoint: endpoint,
			Body:     body,
		},
	}

	return a.InvokeRESTService(ctx, req)
}

// PatchResource performs HTTP PATCH operation
func (a *RESTServiceActivities) PatchResource(ctx context.Context, serviceName, baseURL, endpoint string, auth restclient.AuthConfig, body interface{}) (*RESTServiceResponse, error) {
	req := RESTServiceRequest{
		ServiceName: serviceName,
		BaseURL:     baseURL,
		Auth:        auth,
		Request: restclient.RESTRequest{
			Method:   restclient.PATCH,
			Endpoint: endpoint,
			Body:     body,
		},
	}

	return a.InvokeRESTService(ctx, req)
}

// DeleteResource performs HTTP DELETE operation
func (a *RESTServiceActivities) DeleteResource(ctx context.Context, serviceName, baseURL, endpoint string, auth restclient.AuthConfig) (*RESTServiceResponse, error) {
	req := RESTServiceRequest{
		ServiceName: serviceName,
		BaseURL:     baseURL,
		Auth:        auth,
		Request: restclient.RESTRequest{
			Method:   restclient.DELETE,
			Endpoint: endpoint,
		},
	}

	return a.InvokeRESTService(ctx, req)
}

// BatchRESTCalls executes multiple REST calls in sequence
func (a *RESTServiceActivities) BatchRESTCalls(ctx context.Context, requests []RESTServiceRequest) ([]*RESTServiceResponse, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Executing batch REST calls", "count", len(requests))

	responses := make([]*RESTServiceResponse, len(requests))

	for i, req := range requests {
		logger.Info("Executing batch request",
			"index", i+1,
			"of", len(requests),
			"service", req.ServiceName,
			"endpoint", req.Request.Endpoint)

		resp, err := a.InvokeRESTService(ctx, req)
		if err != nil {
			logger.Error("Batch request failed",
				"index", i+1,
				"service", req.ServiceName,
				"error", err)
			responses[i] = &RESTServiceResponse{
				ServiceName:  req.ServiceName,
				Success:      false,
				ErrorMessage: err.Error(),
			}
		} else {
			responses[i] = resp
		}
	}

	// Count results
	successful := 0
	failed := 0
	for _, resp := range responses {
		if resp.Success {
			successful++
		} else {
			failed++
		}
	}

	logger.Info("Batch REST calls completed",
		"total", len(requests),
		"successful", successful,
		"failed", failed)

	return responses, nil
}

// ValidateRESTResponse validates REST response against expected criteria
func (a *RESTServiceActivities) ValidateRESTResponse(ctx context.Context, response *RESTServiceResponse, expectedStatusCode int, requiredFields []string) error {
	logger := activity.GetLogger(ctx)

	// Check status code
	if expectedStatusCode > 0 && response.StatusCode != expectedStatusCode {
		return fmt.Errorf("expected status code %d, got %d", expectedStatusCode, response.StatusCode)
	}

	// Check required fields in JSON response
	if len(requiredFields) > 0 {
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(response.Body), &jsonData); err != nil {
			return fmt.Errorf("failed to parse JSON response: %v", err)
		}

		for _, field := range requiredFields {
			if _, exists := jsonData[field]; !exists {
				return fmt.Errorf("required field '%s' not found in response", field)
			}
		}
	}

	logger.Info("REST response validation successful",
		"service", response.ServiceName,
		"status_code", response.StatusCode)

	return nil
}

// isRetryableStatus checks if status code is retryable
func (a *RESTServiceActivities) isRetryableStatus(statusCode int, retryableStatusCodes []int) bool {
	for _, code := range retryableStatusCodes {
		if statusCode == code {
			return true
		}
	}
	return false
}

// GetRESTActivityOptions returns recommended activity options for REST service calls
func GetRESTActivityOptions() *activity.Options {
	return &activity.Options{
		StartToCloseTimeout: 60 * time.Second,
		HeartbeatTimeout:    10 * time.Second,
		RetryPolicy: &activity.RetryPolicy{
			InitialInterval:    2 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    3,
			NonRetryableErrorTypes: []string{
				"InvalidArgument",
				"PermissionDenied",
				"Unauthenticated",
			},
		},
	}
}

// GetLongRunningRESTActivityOptions returns activity options for long-running REST operations
func GetLongRunningRESTActivityOptions() *activity.Options {
	return &activity.Options{
		StartToCloseTimeout: 300 * time.Second, // 5 minutes
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &activity.RetryPolicy{
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    60 * time.Second,
			MaximumAttempts:    5,
			NonRetryableErrorTypes: []string{
				"InvalidArgument",
				"PermissionDenied",
				"Unauthenticated",
			},
		},
	}
}

// GetNoRetryRESTActivityOptions returns activity options with no retry policy
func GetNoRetryRESTActivityOptions() *activity.Options {
	return &activity.Options{
		StartToCloseTimeout: 60 * time.Second,
		HeartbeatTimeout:    10 * time.Second,
		RetryPolicy:         nil, // No retries
	}
}

// HealthCheckRequest represents a health check request
type HealthCheckRequest struct {
	ServiceName string                `json:"service_name"`
	BaseURL     string                `json:"base_url"`
	Auth        restclient.AuthConfig `json:"auth"`
	Endpoint    string                `json:"endpoint,omitempty"` // Default: /health
	Timeout     time.Duration         `json:"timeout,omitempty"`   // Default: 10s
}

// HealthCheckResponse represents a health check response
type HealthCheckResponse struct {
	ServiceName string        `json:"service_name"`
	IsHealthy   bool          `json:"is_healthy"`
	StatusCode  int           `json:"status_code"`
	Duration    time.Duration `json:"duration"`
	ErrorMessage string       `json:"error_message,omitempty"`
}

// HealthCheck performs a health check on a REST service
func (a *RESTServiceActivities) HealthCheck(ctx context.Context, req HealthCheckRequest) (*HealthCheckResponse, error) {
	logger := activity.GetLogger(ctx)

	// Set defaults
	endpoint := req.Endpoint
	if endpoint == "" {
		endpoint = "/health"
	}

	timeout := req.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	logger.Info("Performing health check",
		"service", req.ServiceName,
		"endpoint", endpoint)

	restReq := RESTServiceRequest{
		ServiceName: req.ServiceName,
		BaseURL:     req.BaseURL,
		Auth:        req.Auth,
		Request: restclient.RESTRequest{
			Method:   restclient.GET,
			Endpoint: endpoint,
			Timeout:  timeout,
		},
	}

	resp, err := a.InvokeRESTService(ctx, restReq)

	healthResp := &HealthCheckResponse{
		ServiceName: req.ServiceName,
		IsHealthy:   false,
		Duration:    0,
	}

	if err != nil {
		healthResp.ErrorMessage = err.Error()
		logger.Error("Health check failed", "service", req.ServiceName, "error", err)
		return healthResp, nil // Don't return error, just indicate unhealthy
	}

	healthResp.StatusCode = resp.StatusCode
	healthResp.Duration = resp.Duration
	healthResp.IsHealthy = resp.Success && (resp.StatusCode >= 200 && resp.StatusCode < 300)

	if !healthResp.IsHealthy {
		healthResp.ErrorMessage = resp.ErrorMessage
	}

	logger.Info("Health check completed",
		"service", req.ServiceName,
		"healthy", healthResp.IsHealthy,
		"status_code", healthResp.StatusCode,
		"duration", healthResp.Duration)

	return healthResp, nil
}