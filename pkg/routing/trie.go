package routing

import (
	"strings"

	"github.com/Payphone-Digital/gateway/internal/dto"
)

// TrieNode represents a node in the route trie
type TrieNode struct {
	segment       string                            // Path segment ("products", "{id}", etc.)
	isParam       bool                              // True if this segment is a parameter
	paramName     string                            // Parameter name ("id" if segment is "{id}")
	children      map[string]*TrieNode              // Static children (exact matches)
	paramChild    *TrieNode                         // Dynamic parameter child ("{id}", "{slug}", etc.)
	wildcardChild *TrieNode                         // Wildcard child ("*")
	configs       map[string]*dto.APIConfigResponse // Method -> Config mapping
	methods       map[string]bool                   // Allowed HTTP methods at this node
}

// NewTrieNode creates a new trie node
func NewTrieNode(segment string) *TrieNode {
	node := &TrieNode{
		segment:  segment,
		children: make(map[string]*TrieNode),
		methods:  make(map[string]bool),
	}

	// Check if segment is a parameter
	if len(segment) > 2 && segment[0] == '{' && segment[len(segment)-1] == '}' {
		node.isParam = true
		node.paramName = segment[1 : len(segment)-1] // Extract param name without {}
	}

	return node
}

// AddChild adds or retrieves a child node
func (n *TrieNode) AddChild(segment string) *TrieNode {
	// Wildcard
	if segment == "*" {
		if n.wildcardChild == nil {
			n.wildcardChild = NewTrieNode(segment)
		}
		return n.wildcardChild
	}

	// Parameter (e.g., "{id}")
	if len(segment) > 2 && segment[0] == '{' && segment[len(segment)-1] == '}' {
		if n.paramChild == nil {
			n.paramChild = NewTrieNode(segment)
		}
		return n.paramChild
	}

	// Static segment
	if child, exists := n.children[segment]; exists {
		return child
	}

	child := NewTrieNode(segment)
	n.children[segment] = child
	return child
}

// FindChild finds the best matching child for a segment
// Priority: exact match > parameter > wildcard
func (n *TrieNode) FindChild(segment string, params map[string]string) *TrieNode {
	// 1. Try exact match first
	if child, exists := n.children[segment]; exists {
		return child
	}

	// 2. Try parameter match
	if n.paramChild != nil {
		if params != nil {
			params[n.paramChild.paramName] = segment
		}
		return n.paramChild
	}

	// 3. Try wildcard match
	if n.wildcardChild != nil {
		if params != nil {
			params["wildcard"] = segment
		}
		return n.wildcardChild
	}

	return nil
}

// ParseURI splits a URI path into segments
// Example: "/api/users/123" -> ["api", "users", "123"]
func ParseURI(uri string) []string {
	// Remove leading and trailing slashes
	uri = strings.Trim(uri, "/")

	if uri == "" {
		return []string{}
	}

	return strings.Split(uri, "/")
}

// BuildURIPattern builds a URI pattern from segments
// Example: ["api", "users", "{id}"] -> "/api/users/{id}"
func BuildURIPattern(segments []string) string {
	if len(segments) == 0 {
		return "/"
	}
	return "/" + strings.Join(segments, "/")
}

// IsParameterSegment checks if a segment is a parameter
func IsParameterSegment(segment string) bool {
	return len(segment) > 2 && segment[0] == '{' && segment[len(segment)-1] == '}'
}

// IsWildcardSegment checks if a segment is a wildcard
func IsWildcardSegment(segment string) bool {
	return segment == "*"
}
