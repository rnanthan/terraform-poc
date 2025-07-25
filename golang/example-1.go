package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// Example 1: Basic Authentication with JSONPlaceholder API
func exampleBasicAuth() {
	fmt.Println("=== Example 1: Basic Authentication ===")

	// Create config programmatically (you can also use config file)
	config := Config{
		BaseURL:  "https://jsonplaceholder.typicode.com",
		Timeout:  30,
		AuthType: "basic",
		BasicAuth: BasicAuthConfig{
			Username: "testuser",
			Password: "testpass",
		},
		DefaultHeaders: map[string]string{
			"User-Agent": "GoRestClient/1.0",
			"Accept":     "application/json",
		},
	}

	// Save config to file
	configFile, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile("temp-basic-config.json", configFile, 0644)
	defer os.Remove("temp-basic-config.json")

	// Create client
	client, err := NewRestClient("temp-basic-config.json")
	if err != nil {
		log.Printf("Error creating client: %v", err)
		return
	}

	// Get all posts
	resp, err := client.Get("/posts", nil)
	if err != nil {
		log.Printf("Error fetching posts: %v", err)
		return
	}

	fmt.Printf("Status: %d\n", resp.StatusCode)

	// Parse response
	var posts []Post
	if err := json.Unmarshal(resp.Body, &posts); err != nil {
		log.Printf("Error parsing posts: %v", err)
		return
	}

	fmt.Printf("Retrieved %d posts\n", len(posts))
	if len(posts) > 0 {
		fmt.Printf("First post: %s\n", posts[0].Title)
	}
}

// Example 2: OAuth2 with GitHub API
func exampleOAuth2() {
	fmt.Println("\n=== Example 2: OAuth2 (Simulated) ===")

	// Note: This is a simulation since we don't have real OAuth2 credentials
	config := Config{
		BaseURL:  "https://api.github.com",
		Timeout:  30,
		AuthType: "oauth2",
		OAuth2: OAuth2Config{
			ClientID:     "your_github_client_id",
			ClientSecret: "your_github_client_secret",
			TokenURL:     "https://github.com/login/oauth/access_token",
			Scopes:       []string{"repo", "user"},
		},
		DefaultHeaders: map[string]string{
			"Accept":     "application/vnd.github.v3+json",
			"User-Agent": "GoRestClient/1.0",
		},
	}

	configFile, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile("temp-oauth2-config.json", configFile, 0644)
	defer os.Remove("temp-oauth2-config.json")

	fmt.Println("OAuth2 config created (would need real credentials to work)")
	fmt.Println("Config saved to temp-oauth2-config.json")
}

// Example 3: Bearer Token with JSONPlaceholder
func exampleBearerToken() {
	fmt.Println("\n=== Example 3: Bearer Token ===")

	config := Config{
		BaseURL:     "https://jsonplaceholder.typicode.com",
		Timeout:     30,
		AuthType:    "bearer",
		BearerToken: "your-bearer-token-here",
		DefaultHeaders: map[string]string{
			"User-Agent": "GoRestClient/1.0",
			"Accept":     "application/json",
		},
	}

	configFile, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile("temp-bearer-config.json", configFile, 0644)
	defer os.Remove("temp-bearer-config.json")

	client, err := NewRestClient("temp-bearer-config.json")
	if err != nil {
		log.Printf("Error creating client: %v", err)
		return
	}

	// Create a new post
	newPost := Post{
		Title:  "My New Post",
		Body:   "This is the content of my new post",
		UserID: 1,
	}

	resp, err := client.Post("/posts", newPost, nil)
	if err != nil {
		log.Printf("Error creating post: %v", err)
		return
	}

	fmt.Printf("Create Post Status: %d\n", resp.StatusCode)

	var createdPost Post
	if err := json.Unmarshal(resp.Body, &createdPost); err != nil {
		log.Printf("Error parsing created post: %v", err)
		return
	}

	fmt.Printf("Created post ID: %d, Title: %s\n", createdPost.ID, createdPost.Title)
}

// Example 4: Environment Variables Configuration
func exampleEnvironmentConfig() {
	fmt.Println("\n=== Example 4: Environment Variables ===")

	// Set environment variables
	os.Setenv("REST_BASE_URL", "https://jsonplaceholder.typicode.com")
	os.Setenv("REST_TIMEOUT", "30")
	os.Setenv("REST_AUTH_TYPE", "none")

	// Clean up after example
	defer func() {
		os.Unsetenv("REST_BASE_URL")
		os.Unsetenv("REST_TIMEOUT")
		os.Unsetenv("REST_AUTH_TYPE")
	}()

	// Create client without config file (uses env vars)
	client, err := NewRestClient("")
	if err != nil {
		log.Printf("Error creating client: %v", err)
		return
	}

	// Get a specific post
	resp, err := client.Get("/posts/1", map[string]string{
		"X-Custom-Header": "custom-value",
	})
	if err != nil {
		log.Printf("Error fetching post: %v", err)
		return
	}

	fmt.Printf("Status: %d\n", resp.StatusCode)

	var post Post
	if err := json.Unmarshal(resp.Body, &post); err != nil {
		log.Printf("Error parsing post: %v", err)
		return
	}

	fmt.Printf("Post: %s\n", post.Title)
}

// Example 5: Service Wrapper Pattern
type PostService struct {
	client *RestClient
}

func NewPostService(client *RestClient) *PostService {
	return &PostService{client: client}
}

func (s *PostService) GetPost(id int) (*Post, error) {
	resp, err := s.client.Get(fmt.Sprintf("/posts/%d", id), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get post: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error: status %d", resp.StatusCode)
	}

	var post Post
	if err := json.Unmarshal(resp.Body, &post); err != nil {
		return nil, fmt.Errorf("failed to parse post: %w", err)
	}

	return &post, nil
}

func (s *PostService) GetAllPosts() ([]Post, error) {
	resp, err := s.client.Get("/posts", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get posts: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error: status %d", resp.StatusCode)
	}

	var posts []Post
	if err := json.Unmarshal(resp.Body, &posts); err != nil {
		return nil, fmt.Errorf("failed to parse posts: %w", err)
	}

	return posts, nil
}

func (s *PostService) CreatePost(post Post) (*Post, error) {
	resp, err := s.client.Post("/posts", post, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create post: %w", err)
	}

	if resp.StatusCode != 201 {
		return nil, fmt.Errorf("API error: status %d", resp.StatusCode)
	}

	var createdPost Post
	if err := json.Unmarshal(resp.Body, &createdPost); err != nil {
		return nil, fmt.Errorf("failed to parse created post: %w", err)
	}

	return &createdPost, nil
}

func (s *PostService) UpdatePost(id int, post Post) (*Post, error) {
	resp, err := s.client.Put(fmt.Sprintf("/posts/%d", id), post, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to update post: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error: status %d", resp.StatusCode)
	}

	var updatedPost Post
	if err := json.Unmarshal(resp.Body, &updatedPost); err != nil {
		return nil, fmt.Errorf("failed to parse updated post: %w", err)
	}

	return &updatedPost, nil
}

func (s *PostService) DeletePost(id int) error {
	resp, err := s.client.Delete(fmt.Sprintf("/posts/%d", id), nil)
	if err != nil {
		return fmt.Errorf("failed to delete post: %w", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("API error: status %d", resp.StatusCode)
	}

	return nil
}

func exampleServiceWrapper() {
	fmt.Println("\n=== Example 5: Service Wrapper Pattern ===")

	// Create client with no auth for JSONPlaceholder
	config := Config{
		BaseURL:  "https://jsonplaceholder.typicode.com",
		Timeout:  30,
		AuthType: "none",
		DefaultHeaders: map[string]string{
			"User-Agent": "GoRestClient/1.0",
			"Accept":     "application/json",
		},
	}

	configFile, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile("temp-service-config.json", configFile, 0644)
	defer os.Remove("temp-service-config.json")

	client, err := NewRestClient("temp-service-config.json")
	if err != nil {
		log.Printf("Error creating client: %v", err)
		return
	}

	// Create service wrapper
	postService := NewPostService(client)

	// Get a single post
	post, err := postService.GetPost(1)
	if err != nil {
		log.Printf("Error getting post: %v", err)
		return
	}
	fmt.Printf("Retrieved post: %s\n", post.Title)

	// Create a new post
	newPost := Post{
		Title:  "Test Post from Service",
		Body:   "This post was created using the service wrapper",
		UserID: 1,
	}

	createdPost, err := postService.CreatePost(newPost)
	if err != nil {
		log.Printf("Error creating post: %v", err)
		return
	}
	fmt.Printf("Created post ID: %d\n", createdPost.ID)

	// Update the post
	createdPost.Title = "Updated Test Post"
	updatedPost, err := postService.UpdatePost(createdPost.ID, *createdPost)
	if err != nil {
		log.Printf("Error updating post: %v", err)
		return
	}
	fmt.Printf("Updated post title: %s\n", updatedPost.Title)

	// Delete the post
	err = postService.DeletePost(createdPost.ID)
	if err != nil {
		log.Printf("Error deleting post: %v", err)
		return
	}
	fmt.Printf("Post %d deleted successfully\n", createdPost.ID)
}

// Example 6: Error Handling and Retry Logic
func exampleErrorHandling() {
	fmt.Println("\n=== Example 6: Error Handling ===")

	config := Config{
		BaseURL:  "https://httpstat.us", // Service for testing HTTP status codes
		Timeout:  5,
		AuthType: "none",
	}

	configFile, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile("temp-error-config.json", configFile, 0644)
	defer os.Remove("temp-error-config.json")

	client, err := NewRestClient("temp-error-config.json")
	if err != nil {
		log.Printf("Error creating client: %v", err)
		return
	}

	// Test different HTTP status codes
	statusCodes := []int{200, 404, 500, 503}

	for _, code := range statusCodes {
		fmt.Printf("Testing status code %d: ", code)

		resp, err := client.Get(fmt.Sprintf("/%d", code), nil)
		if err != nil {
			fmt.Printf("Request failed: %v\n", err)
			continue
		}

		switch {
		case resp.StatusCode >= 200 && resp.StatusCode < 300:
			fmt.Printf("Success - %s\n", string(resp.Body))
		case resp.StatusCode >= 400 && resp.StatusCode < 500:
			fmt.Printf("Client error - %s\n", string(resp.Body))
		case resp.StatusCode >= 500:
			fmt.Printf("Server error - %s\n", string(resp.Body))
		}
	}
}

// Example 7: Multiple API Integration
func exampleMultipleAPIs() {
	fmt.Println("\n=== Example 7: Multiple API Integration ===")

	// JSONPlaceholder API
	jsonClient, err := createJSONPlaceholderClient()
	if err != nil {
		log.Printf("Error creating JSON client: %v", err)
		return
	}

	// Get posts from JSONPlaceholder
	resp, err := jsonClient.Get("/posts", nil)
	if err != nil {
		log.Printf("Error fetching from JSONPlaceholder: %v", err)
	} else {
		var posts []Post
		json.Unmarshal(resp.Body, &posts)
		fmt.Printf("JSONPlaceholder: Retrieved %d posts\n", len(posts))
	}

	// HTTPBin API (for testing different HTTP methods)
	httpbinClient, err := createHTTPBinClient()
	if err != nil {
		log.Printf("Error creating HTTPBin client: %v", err)
		return
	}

	// Test POST with HTTPBin
	testData := map[string]interface{}{
		"message": "Hello from Go REST Client",
		"timestamp": "2024-01-01T00:00:00Z",
	}

	resp, err = httpbinClient.Post("/post", testData, nil)
	if err != nil {
		log.Printf("Error posting to HTTPBin: %v", err)
	} else {
		fmt.Printf("HTTPBin POST Status: %d\n", resp.StatusCode)
	}
}

func createJSONPlaceholderClient() (*RestClient, error) {
	config := Config{
		BaseURL:  "https://jsonplaceholder.typicode.com",
		Timeout:  30,
		AuthType: "none",
		DefaultHeaders: map[string]string{
			"User-Agent": "GoRestClient-JSONPlaceholder/1.0",
		},
	}

	configFile, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile("temp-json-config.json", configFile, 0644)
	defer os.Remove("temp-json-config.json")

	return NewRestClient("temp-json-config.json")
}

func createHTTPBinClient() (*RestClient, error) {
	config := Config{
		BaseURL:  "https://httpbin.org",
		Timeout:  30,
		AuthType: "none",
		DefaultHeaders: map[string]string{
			"User-Agent": "GoRestClient-HTTPBin/1.0",
		},
	}

	configFile, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile("temp-httpbin-config.json", configFile, 0644)
	defer os.Remove("temp-httpbin-config.json")

	return NewRestClient("temp-httpbin-config.json")
}

// Data structures for examples
type Post struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	UserID int    `json:"userId"`
}

// Main function to run all examples
func main() {
	fmt.Println("REST Client Template - Usage Examples")
	fmt.Println("=====================================")

	// Run all examples
	exampleBasicAuth()
	exampleOAuth2()
	exampleBearerToken()
	exampleEnvironmentConfig()
	exampleServiceWrapper()
	exampleErrorHandling()
	exampleMultipleAPIs()

	fmt.Println("\n=== All Examples Completed ===")
}