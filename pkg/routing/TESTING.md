# Dynamic URI Registry - Test Cases

## Test Coverage: Multi-Method Support

### Test 1: Same Path, Different Methods (GET & POST)
```
URI: /users
Methods: GET, POST

GET /users → get-users config ✅
POST /users → create-user config ✅
```

### Test 2: Parameter Path, Different Methods (GET & PUT)
```
URI: /users/{id}
Methods: GET, PUT

GET /users/123 → get-user config ✅
PUT /users/456 → update-user config ✅
```

### Test 3: Nested Parameters
```
URI: /users/{id}/posts
Method: GET

GET /users/789/posts → get-user-posts config ✅
Params: {id: "789"} ✅
```

### Test 4: Method Not Allowed
```  
DELETE /users → ErrMethodNotAllowed ✅
(Only GET & POST available)
```

### Test 5: Route Not Found
```
GET /products → ErrRouteNotFound ✅
```

## Key Fix Applied

**Before**: Single config per path, last config overwrites previous
**After**: Map of configs by method - each method gets its own config

```go
// Old structure
type TrieNode struct {
    config interface{}  // ❌ Only one config
}

// New structure  
type TrieNode struct {
    configs map[string]*dto.APIConfigResponse  // ✅ Config per method
}
```

## Result
✅ **Proper multi-method support on same URI**
✅ **Each method returns correct config**
✅ **Slug used only for tracking, not matching**
✅ **URI + Method = Primary matching key**
