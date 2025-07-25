# go.mod
module rest-client-template

go 1.21

require golang.org/x/oauth2 v0.15.0

require (
    github.com/golang/protobuf v1.5.3 // indirect
    golang.org/x/net v0.19.0 // indirect
    google.golang.org/appengine v1.6.7 // indirect
    google.golang.org/protobuf v1.31.0 // indirect
)

---

# config-basic.json - Basic Authentication Example
{
  "base_url": "https://api.example.com",
  "timeout_seconds": 30,
  "auth_type": "basic",
  "basic_auth": {
    "username": "your_username",
    "password": "your_password"
  },
  "default_headers": {
    "User-Agent": "RestClient/1.0",
    "Accept": "application/json"
  }
}

---

# config-oauth2.json - OAuth2 Client Credentials Example
{
  "base_url": "https://api.example.com",
  "timeout_seconds": 30,
  "auth_type": "oauth2",
  "oauth2": {
    "client_id": "your_client_id",
    "client_secret": "your_client_secret",
    "token_url": "https://auth.example.com/oauth/token",
    "scopes": ["read", "write"],
    "extra_params": {
      "audience": "https://api.example.com"
    }
  },
  "default_headers": {
    "User-Agent": "RestClient/1.0",
    "Accept": "application/json"
  }
}

---

# config-bearer.json - Bearer Token Example
{
  "base_url": "https://api.example.com",
  "timeout_seconds": 30,
  "auth_type": "bearer",
  "bearer_token": "your_bearer_token_here",
  "default_headers": {
    "User-Agent": "RestClient/1.0",
    "Accept": "application/json"
  }
}

---

# .env - Environment Variables Example
REST_BASE_URL=https://api.example.com
REST_TIMEOUT=30
REST_AUTH_TYPE=basic
REST_BASIC_USERNAME=your_username
REST_BASIC_PASSWORD=your_password
# OR for OAuth2:
# REST_AUTH_TYPE=oauth2
# REST_OAUTH2_CLIENT_ID=your_client_id
# REST_OAUTH2_CLIENT_SECRET=your_client_secret
# REST_OAUTH2_TOKEN_URL=https://auth.example.com/oauth/token
# OR for Bearer:
# REST_AUTH_TYPE=bearer
# REST_BEARER_TOKEN=your_bearer_token

---

# example_usage.go - Extended Usage Examples
package main

import (
    "fmt"
    "log"
)

// UserService demonstrates service-specific methods
type UserService struct {
    client *RestClient
}

func NewUserService(client *RestClient) *UserService {
    return &UserService{client: client}
}

func (s *UserService) GetUser(id string) (*User, error) {
    resp, err := s.client.Get(fmt.Sprintf("/users/%s", id), nil)
    if err != nil {
        return nil, err
    }

    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(resp.Body))
    }

    var user User
    if err := json.Unmarshal(resp.Body, &user); err != nil {
        return nil, fmt.Errorf("failed to parse user: %w", err)
    }

    return &user, nil
}

func (s *UserService) CreateUser(user CreateUserRequest) (*User, error) {
    resp, err := s.client.Post("/users", user, nil)
    if err != nil {
        return nil, err
    }

    if resp.StatusCode != 201 {
        return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(resp.Body))
    }

    var createdUser User
    if err := json.Unmarshal(resp.Body, &createdUser); err != nil {
        return nil, fmt.Errorf("failed to parse created user: %w", err)
    }

    return &createdUser, nil
}

type User struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

func exampleUsage() {
    // Example 1: Using config file
    client, err := NewRestClient("config-basic.json")
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }

    userService := NewUserService(client)

    // Get a user
    user, err := userService.GetUser("123")
    if err != nil {
        log.Printf("Failed to get user: %v", err)
    } else {
        fmt.Printf("Retrieved user: %+v\n", user)
    }

    // Create a user
    newUser := CreateUserRequest{
        Name:  "Jane Doe",
        Email: "jane@example.com",
    }

    createdUser, err := userService.CreateUser(newUser)
    if err != nil {
        log.Printf("Failed to create user: %v", err)
    } else {
        fmt.Printf("Created user: %+v\n", createdUser)
    }

    // Example 2: Using environment variables (no config file)
    envClient, err := NewRestClient("")
    if err != nil {
        log.Fatalf("Failed to create env client: %v", err)
    }

    // Make direct API calls
    resp, err := envClient.Get("/health", map[string]string{
        "X-Custom-Header": "custom-value",
    })
    if err != nil {
        log.Printf("Health check failed: %v", err)
    } else {
        fmt.Printf("Health check status: %d\n", resp.StatusCode)
    }
}