package routing

import (
	"errors"
	"fmt"
	"sync"

	"github.com/Payphone-Digital/gateway/internal/dto"
	"go.uber.org/zap"
)

var (
	// ErrRouteNotFound is returned when no route matches the request
	ErrRouteNotFound = errors.New("route not found")

	// ErrMethodNotAllowed is returned when route exists but method is not allowed
	ErrMethodNotAllowed = errors.New("method not allowed")

	// ErrRouteAlreadyExists is returned when trying to add a duplicate route
	ErrRouteAlreadyExists = errors.New("route already exists")
)

// RouteRegistry maintains an in-memory trie of all routes
type RouteRegistry struct {
	mu     sync.RWMutex
	root   *TrieNode
	routes map[string]*dto.APIConfigResponse // slug -> config for quick lookup
	logger *zap.Logger
}

// NewRouteRegistry creates a new route registry
func NewRouteRegistry(logger *zap.Logger) *RouteRegistry {
	return &RouteRegistry{
		root:   NewTrieNode(""),
		routes: make(map[string]*dto.APIConfigResponse),
		logger: logger,
	}
}

// AddRoute adds a route to the registry
func (r *RouteRegistry) AddRoute(config *dto.APIConfigResponse) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Create unique key for path+method combination
	routeKey := config.Path + ":" + config.Method

	// Check for duplicate path+method combination
	if _, exists := r.routes[routeKey]; exists {
		return fmt.Errorf("%w: path=%s, method=%s", ErrRouteAlreadyExists, config.Path, config.Method)
	}

	// Parse Path into segments (Path is the public URL exposed to clients like /v1/products/{id})
	// URI is the backend target path which is used for forwarding, not for matching
	segments := ParseURI(config.Path)

	r.logger.Debug("Adding route to registry",
		zap.String("path", config.Path),
		zap.String("target_uri", config.URI),
		zap.String("method", config.Method),
		zap.Int("segments", len(segments)),
	)

	// Navigate/create trie path
	node := r.root
	for _, segment := range segments {
		node = node.AddChild(segment)
	}

	// Initialize configs map if nil
	if node.configs == nil {
		node.configs = make(map[string]*dto.APIConfigResponse)
	}

	// Set config for this specific method
	node.configs[config.Method] = config
	node.methods[config.Method] = true

	// Store in routes map with path:method key for quick lookup
	r.routes[routeKey] = config

	r.logger.Info("Route added successfully",
		zap.String("path", config.Path),
		zap.String("target_uri", config.URI),
		zap.String("method", config.Method),
	)

	return nil
}

// Match finds a matching route for the given path and method
// Returns: (config, params, error)
func (r *RouteRegistry) Match(path, method string) (*dto.APIConfigResponse, map[string]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Parse request path
	segments := ParseURI(path)
	params := make(map[string]string)

	r.logger.Debug("Matching route",
		zap.String("path", path),
		zap.String("method", method),
		zap.Int("segments", len(segments)),
	)

	// Traverse trie
	node := r.root
	for i, segment := range segments {
		child := node.FindChild(segment, params)
		if child == nil {
			r.logger.Debug("No matching child found",
				zap.String("path", path),
				zap.Int("segment_index", i),
				zap.String("segment", segment),
			)
			return nil, nil, ErrRouteNotFound
		}
		node = child
	}

	// Check if we reached a configured endpoint
	if node.configs == nil || len(node.configs) == 0 {
		r.logger.Debug("Path matched but no configs found",
			zap.String("path", path),
		)
		return nil, nil, ErrRouteNotFound
	}

	// Get config for specific method
	config, exists := node.configs[method]
	if !exists {
		// Check if any method exists (for better error message)
		if len(node.configs) > 0 {
			r.logger.Debug("Method not allowed",
				zap.String("path", path),
				zap.String("method", method),
				zap.Any("available_methods", getKeys(node.configs)),
			)
			return nil, nil, ErrMethodNotAllowed
		}
		return nil, nil, ErrRouteNotFound
	}

	r.logger.Info("Route matched successfully",
		zap.String("slug", config.Path),
		zap.String("path", path),
		zap.String("method", method),
		zap.Int("params_count", len(params)),
		zap.Any("params", params),
	)

	return config, params, nil
}

// Helper function to get map keys
func getKeys(m map[string]*dto.APIConfigResponse) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// GetBySlug retrieves a route configuration by slug
func (r *RouteRegistry) GetBySlug(slug string) (*dto.APIConfigResponse, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	config, exists := r.routes[slug]
	return config, exists
}

// RemoveRoute removes a route from the registry by path and method
func (r *RouteRegistry) RemoveRoute(path, method string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	routeKey := path + ":" + method
	config, exists := r.routes[routeKey]
	if !exists {
		return fmt.Errorf("route not found: path=%s, method=%s", path, method)
	}

	// Remove from routes map
	delete(r.routes, routeKey)

	// Also remove from trie node's configs map
	segments := ParseURI(path)
	node := r.root
	for _, segment := range segments {
		child := node.FindChild(segment, nil)
		if child == nil {
			break
		}
		node = child
	}
	if node.configs != nil {
		delete(node.configs, method)
		delete(node.methods, method)
	}

	r.logger.Info("Route removed successfully",
		zap.String("path", path),
		zap.String("method", method),
		zap.String("uri", config.URI),
	)

	return nil
}

// Count returns the total number of routes
func (r *RouteRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.routes)
}

// List returns all registered route slugs
func (r *RouteRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	slugs := make([]string, 0, len(r.routes))
	for slug := range r.routes {
		slugs = append(slugs, slug)
	}

	return slugs
}

// Clear removes all routes from the registry
func (r *RouteRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.root = NewTrieNode("")
	r.routes = make(map[string]*dto.APIConfigResponse)

	r.logger.Info("Registry cleared")
}
