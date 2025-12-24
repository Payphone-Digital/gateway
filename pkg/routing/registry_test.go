package routing

import (
	"testing"

	"github.com/Payphone-Digital/gateway/internal/dto"
	"go.uber.org/zap"
)

func setupTestLogger() *zap.Logger {
	// Use no-op logger for tests
	return zap.NewNop()
}

func createTestConfig(slug, uri, method string) *dto.APIConfigResponse {
	return &dto.APIConfigResponse{
		Slug:   slug,
		URI:    uri,
		Method: method,
	}
}

func TestNewRouteRegistry(t *testing.T) {
	logger := setupTestLogger()
	registry := NewRouteRegistry(logger)

	if registry == nil {
		t.Fatal("Expected registry, got nil")
	}

	if registry.Count() != 0 {
		t.Errorf("Expected count 0, got %d", registry.Count())
	}
}

func TestRouteRegistry_AddRoute(t *testing.T) {
	logger := setupTestLogger()
	registry := NewRouteRegistry(logger)

	tests := []struct {
		name        string
		config      *dto.APIConfigResponse
		shouldError bool
	}{
		{
			name:        "Add simple route",
			config:      createTestConfig("get-users", "/users", "GET"),
			shouldError: false,
		},
		{
			name:        "Add route with parameter",
			config:      createTestConfig("get-user", "/users/{id}", "GET"),
			shouldError: false,
		},
		{
			name:        "Add duplicate slug",
			config:      createTestConfig("get-users", "/users", "POST"),
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.AddRoute(tt.config)

			if tt.shouldError && err == nil {
				t.Error("Expected error, got nil")
			}

			if !tt.shouldError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

func TestRouteRegistry_Match(t *testing.T) {
	logger := setupTestLogger()
	registry := NewRouteRegistry(logger)

	// Add test routes - now properly supports multiple methods on same path
	registry.AddRoute(createTestConfig("get-users", "/users", "GET"))
	registry.AddRoute(createTestConfig("create-user", "/users", "POST"))
	registry.AddRoute(createTestConfig("get-user", "/users/{id}", "GET"))
	registry.AddRoute(createTestConfig("update-user", "/users/{id}", "PUT"))
	registry.AddRoute(createTestConfig("get-user-posts", "/users/{id}/posts", "GET"))

	tests := []struct {
		name           string
		path           string
		method         string
		shouldMatch    bool
		expectedSlug   string
		expectedParams map[string]string
		expectedError  error
	}{
		{
			name:           "Match exact path GET",
			path:           "/users",
			method:         "GET",
			shouldMatch:    true,
			expectedSlug:   "get-users",
			expectedParams: map[string]string{},
		},
		{
			name:           "Match exact path POST",
			path:           "/users",
			method:         "POST",
			shouldMatch:    true,
			expectedSlug:   "create-user", // Now correctly returns create-user for POST
			expectedParams: map[string]string{},
		},
		{
			name:           "Match parameter path GET",
			path:           "/users/123",
			method:         "GET",
			shouldMatch:    true,
			expectedSlug:   "get-user",
			expectedParams: map[string]string{"id": "123"},
		},
		{
			name:           "Match parameter path PUT",
			path:           "/users/456",
			method:         "PUT",
			shouldMatch:    true,
			expectedSlug:   "update-user", // Different config for PUT
			expectedParams: map[string]string{"id": "456"},
		},
		{
			name:           "Match nested parameter",
			path:           "/users/789/posts",
			method:         "GET",
			shouldMatch:    true,
			expectedSlug:   "get-user-posts",
			expectedParams: map[string]string{"id": "789"},
		},
		{
			name:          "No route found",
			path:          "/products",
			method:        "GET",
			shouldMatch:   false,
			expectedError: ErrRouteNotFound,
		},
		{
			name:          "Method not allowed",
			path:          "/users",
			method:        "DELETE",
			shouldMatch:   false,
			expectedError: ErrMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, params, err := registry.Match(tt.path, tt.method)

			if tt.shouldMatch {
				if err != nil {
					t.Errorf("Expected match, got error: %v", err)
					return
				}

				if config == nil {
					t.Error("Expected config, got nil")
					return
				}

				if config.Slug != tt.expectedSlug {
					t.Errorf("Expected slug %s, got %s", tt.expectedSlug, config.Slug)
				}

				if tt.expectedParams != nil {
					for key, val := range tt.expectedParams {
						if params[key] != val {
							t.Errorf("Expected param[%s] = %s, got %s", key, val, params[key])
						}
					}
				}
			} else {
				if err != tt.expectedError {
					t.Errorf("Expected error %v, got %v", tt.expectedError, err)
				}
			}
		})
	}
}

func TestRouteRegistry_GetBySlug(t *testing.T) {
	logger := setupTestLogger()
	registry := NewRouteRegistry(logger)

	testConfig := createTestConfig("test-route", "/test", "GET")
	registry.AddRoute(testConfig)

	// Test existing slug
	config, exists := registry.GetBySlug("test-route")
	if !exists {
		t.Error("Expected to find route")
	}
	if config.Slug != "test-route" {
		t.Errorf("Expected slug 'test-route', got %s", config.Slug)
	}

	// Test non-existing slug
	_, exists = registry.GetBySlug("non-existent")
	if exists {
		t.Error("Expected not to find route")
	}
}

func TestRouteRegistry_RemoveRoute(t *testing.T) {
	logger := setupTestLogger()
	registry := NewRouteRegistry(logger)

	testConfig := createTestConfig("test-route", "/test", "GET")
	registry.AddRoute(testConfig)

	if registry.Count() != 1 {
		t.Errorf("Expected count 1, got %d", registry.Count())
	}

	// Remove route
	err := registry.RemoveRoute("test-route")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if registry.Count() != 0 {
		t.Errorf("Expected count 0 after removal, got %d", registry.Count())
	}

	// Remove non-existing
	err = registry.RemoveRoute("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent route")
	}
}

func TestRouteRegistry_Clear(t *testing.T) {
	logger := setupTestLogger()
	registry := NewRouteRegistry(logger)

	// Add multiple routes
	registry.AddRoute(createTestConfig("route1", "/route1", "GET"))
	registry.AddRoute(createTestConfig("route2", "/route2", "POST"))
	registry.AddRoute(createTestConfig("route3", "/route3", "PUT"))

	if registry.Count() != 3 {
		t.Errorf("Expected count 3, got %d", registry.Count())
	}

	// Clear all
	registry.Clear()

	if registry.Count() != 0 {
		t.Errorf("Expected count 0 after clear, got %d", registry.Count())
	}
}

func TestRouteRegistry_List(t *testing.T) {
	logger := setupTestLogger()
	registry := NewRouteRegistry(logger)

	// Add routes
	registry.AddRoute(createTestConfig("route1", "/route1", "GET"))
	registry.AddRoute(createTestConfig("route2", "/route2", "POST"))

	slugs := registry.List()

	if len(slugs) != 2 {
		t.Errorf("Expected 2 slugs, got %d", len(slugs))
	}

	// Check if both slugs are present
	hasRoute1 := false
	hasRoute2 := false
	for _, slug := range slugs {
		if slug == "route1" {
			hasRoute1 = true
		}
		if slug == "route2" {
			hasRoute2 = true
		}
	}

	if !hasRoute1 || !hasRoute2 {
		t.Error("Expected both routes in list")
	}
}

// Benchmark tests
func BenchmarkRouteRegistry_Match(b *testing.B) {
	logger := setupTestLogger()
	registry := NewRouteRegistry(logger)

	// Add 100 routes
	for i := 0; i < 100; i++ {
		slug := "route-" + string(rune(i))
		path := "/api/v1/resource/" + string(rune(i))
		registry.AddRoute(createTestConfig(slug, path, "GET"))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		registry.Match("/api/v1/resource/50", "GET")
	}
}

func BenchmarkRouteRegistry_AddRoute(b *testing.B) {
	logger := setupTestLogger()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		registry := NewRouteRegistry(logger)
		for j := 0; j < 100; j++ {
			slug := "route-" + string(rune(j))
			path := "/api/resource/" + string(rune(j))
			registry.AddRoute(createTestConfig(slug, path, "GET"))
		}
	}
}
