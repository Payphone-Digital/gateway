package routing

import (
	"testing"
)

func TestNewTrieNode(t *testing.T) {
	tests := []struct {
		name      string
		segment   string
		isParam   bool
		paramName string
	}{
		{
			name:      "Static segment",
			segment:   "users",
			isParam:   false,
			paramName: "",
		},
		{
			name:      "Parameter segment",
			segment:   "{id}",
			isParam:   true,
			paramName: "id",
		},
		{
			name:      "Complex parameter",
			segment:   "{user_id}",
			isParam:   true,
			paramName: "user_id",
		},
		{
			name:      "Wildcard",
			segment:   "*",
			isParam:   false,
			paramName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := NewTrieNode(tt.segment)

			if node.segment != tt.segment {
				t.Errorf("Expected segment %s, got %s", tt.segment, node.segment)
			}

			if node.isParam != tt.isParam {
				t.Errorf("Expected isParam %v, got %v", tt.isParam, node.isParam)
			}

			if node.paramName != tt.paramName {
				t.Errorf("Expected paramName %s, got %s", tt.paramName, node.paramName)
			}
		})
	}
}

func TestParseURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected []string
	}{
		{
			name:     "Simple path",
			uri:      "/users/123",
			expected: []string{"users", "123"},
		},
		{
			name:     "Root path",
			uri:      "/",
			expected: []string{},
		},
		{
			name:     "Path with trailing slash",
			uri:      "/api/v1/",
			expected: []string{"api", "v1"},
		},
		{
			name:     "Deep nested path",
			uri:      "/api/v1/users/123/posts/456",
			expected: []string{"api", "v1", "users", "123", "posts", "456"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseURI(tt.uri)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d segments, got %d", len(tt.expected), len(result))
				return
			}

			for i, seg := range tt.expected {
				if result[i] != seg {
					t.Errorf("Expected segment[%d] = %s, got %s", i, seg, result[i])
				}
			}
		})
	}
}

func TestTrieNode_AddChild(t *testing.T) {
	root := NewTrieNode("")

	// Test adding static children
	child1 := root.AddChild("users")
	if child1.segment != "users" {
		t.Errorf("Expected segment 'users', got %s", child1.segment)
	}

	// Test idempotent - adding same child returns existing
	child1_again := root.AddChild("users")
	if child1_again != child1 {
		t.Error("Expected same child instance for duplicate add")
	}

	// Test parameter child
	paramChild := root.AddChild("{id}")
	if !paramChild.isParam {
		t.Error("Expected parameter child")
	}
	if paramChild.paramName != "id" {
		t.Errorf("Expected paramName 'id', got %s", paramChild.paramName)
	}

	// Test wildcard
	wildcardChild := root.AddChild("*")
	if root.wildcardChild != wildcardChild {
		t.Error("Expected wildcard child to be set")
	}
}

func TestTrieNode_FindChild(t *testing.T) {
	root := NewTrieNode("")

	// Add some children
	usersNode := root.AddChild("users")
	root.AddChild("{id}")
	root.AddChild("*")

	tests := []struct {
		name           string
		segment        string
		shouldFind     bool
		expectedNode   *TrieNode
		expectedParams map[string]string
	}{
		{
			name:           "Find exact match",
			segment:        "users",
			shouldFind:     true,
			expectedNode:   usersNode,
			expectedParams: map[string]string{},
		},
		{
			name:           "Find via parameter",
			segment:        "123",
			shouldFind:     true,
			expectedParams: map[string]string{"id": "123"},
		},
		{
			name:           "Find via wildcard",
			segment:        "anything",
			shouldFind:     true,
			expectedParams: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := make(map[string]string)
			result := root.FindChild(tt.segment, params)

			if tt.shouldFind && result == nil {
				t.Error("Expected to find child, got nil")
			}

			if tt.expectedNode != nil && result != tt.expectedNode {
				t.Error("Expected specific node, got different")
			}

			for key, val := range tt.expectedParams {
				if params[key] != val {
					t.Errorf("Expected param[%s] = %s, got %s", key, val, params[key])
				}
			}
		})
	}
}

func TestIsParameterSegment(t *testing.T) {
	tests := []struct {
		segment  string
		expected bool
	}{
		{"{id}", true},
		{"{user_id}", true},
		{"users", false},
		{"{", false},
		{"}", false},
		{"{}", false},
		{"*", false},
	}

	for _, tt := range tests {
		t.Run(tt.segment, func(t *testing.T) {
			result := IsParameterSegment(tt.segment)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for segment %s", tt.expected, result, tt.segment)
			}
		})
	}
}

func TestIsWildcardSegment(t *testing.T) {
	tests := []struct {
		segment  string
		expected bool
	}{
		{"*", true},
		{"users", false},
		{"{id}", false},
		{"**", false},
	}

	for _, tt := range tests {
		t.Run(tt.segment, func(t *testing.T) {
			result := IsWildcardSegment(tt.segment)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for segment %s", tt.expected, result, tt.segment)
			}
		})
	}
}
