# API Response Format Standardization

This document defines the standardized response format used across all API endpoints in this application.

## Response Types

### 1. List Response
Used for endpoints that return paginated data collections.

```go
// Usage
constants.BuildListResponse(total, page, pageTotal, data)
```

**Response Structure:**
```json
{
  "total": 150,
  "page": 1,
  "page_total": 15,
  "data": [
    {
      "id": 1,
      "slug": "example-api",
      "method": "GET",
      "url": "https://api.example.com/users",
      // ... other fields
    }
  ]
}
```

**Field Descriptions:**
- `total`: Total number of records
- `page`: Current page number (1-based)
- `page_total`: Total number of pages
- `data`: Array of data records

### 2. Success Response
Used for endpoints that perform successful operations (create, update, delete).

```go
// Usage
constants.BuildSuccessResponse("Operation completed successfully")
```

**Response Structure:**
```json
{
  "message": "Create successful"
}
```

### 3. Error Response
Used for endpoints that encounter errors during operation.

```go
// Usage
constants.BuildErrorResponse("Operation failed", "Detailed error message")
```

**Response Structure:**
```json
{
  "message": "Create failed",
  "details": "Validation error: Invalid field value"
}
```

## Constants Reference

### Field Keys
```go
const (
    ResponseFieldTotal     = "total"
    ResponseFieldPage      = "page"
    ResponseFieldPageTotal = "page_total"
    ResponseFieldData      = "data"
    ResponseFieldMessage  = "message"
    ResponseFieldDetails  = "details"
    ResponseFieldError    = "error"
    ResponseFieldRendered = "rendered"
)
```

### Response Builder Functions
```go
// For paginated list responses
func BuildListResponse(total int64, page int, pageTotal int, data any) map[string]any

// For error responses with optional details
func BuildErrorResponse(message string, details any) map[string]any

// For simple success messages
func BuildSuccessResponse(message string) map[string]any
```

## Endpoints Using Standard Format

### URL Config Management
- `GET /api/v1/url-config` - List response with pagination
- `POST /api/v1/url-config` - Success response on creation
- `PUT /api/v1/url-config/:id` - Success response on update
- `DELETE /api/v1/url-config/:id` - Success response on deletion
- `GET /api/v1/url-config/:id` - Direct data response

### Path Config Management
- `GET /api/v1/path-config` - List response with pagination and filtering
- `POST /api/v1/path-config` - Success response on creation
- `PUT /api/v1/path-config/:id` - Success response on update
- `DELETE /api/v1/path-config/:id` - Success response on deletion
- `GET /api/v1/path-config/:id` - Direct data response

### User Management
- `GET /users` - List response with pagination and search
- `GET /users/:id` - Direct data response (user object)
- `POST /users` - Direct data response (created user object)
- `PUT /users/:id` - Direct data response (updated user object)
- `DELETE /users/:id` - Success response on deletion
- `PUT /users/:id/password` - Success response on password update
- All error responses use standardized error format

## Usage Examples

### Handler Implementation (DTO-Based Clean Single Parameter)
```go
func (h *APIConfigHandler) GetAllConfig(c *gin.Context) {
    // Create filter DTO with query tags
    filter := &dto.APIConfigFilter{}

    // Parse pagination parameters with DTO-based filters
    pagination := constants.ParsePaginationParamsWithFilter(c, filter)

    // Clean single parameter call
    res, total, pageTotal, status, err := h.integrasiService.GetAllConfig(pagination.All)
    if err != nil {
        c.JSON(status, constants.BuildErrorResponse("Failed to fetch pages", err.Error()))
    } else {
        c.JSON(http.StatusOK, constants.BuildListResponse(total, pagination.Page, pageTotal, res))
    }
}
```

### DTO-Based Filter Usage
```go
// Define filter DTO with query tags
type APIConfigFilter struct {
    URLConfigID uint   `query:"url_config_id"` // Filter by URL Config ID
    Protocol    string `query:"protocol"`       // Filter by protocol (http/grpc)
    Method      string `query:"method"`         // Filter by HTTP method
    IsAdmin     *bool  `query:"is_admin"`       // Filter by admin status (nil = no filter)
    Status      string `query:"status"`         // Custom status filter
}

// Use in handler - automatically parses query parameters based on DTO fields
filter := &dto.APIConfigFilter{}
pagination := constants.ParsePaginationParamsWithFilter(c, filter)

// Filter values are automatically populated from query string:
// GET /api/v1/path-config?url_config_id=5&protocol=http&is_admin=true
// → filter.URLConfigID = 5
// → filter.Protocol = "http"
// → filter.IsAdmin = true
```

### Basic Pagination (No Filters)
```go
// For endpoints without filters
pagination := constants.ParsePaginationParams(c)
// Only parses: page, limit, offset
```

### Multiple Filter DTOs
```go
// URL Config Filter
type URLConfigFilter struct {
    Protocol string `query:"protocol"`
    IsActive *bool  `query:"is_active"`
}

// User Filter
type UserFilter struct {
    IsVerified *bool  `query:"is_verified"`
    Role       string `query:"role"`
    Status     string `query:"status"`
}

// Usage in different endpoints
urlConfigFilter := &dto.URLConfigFilter{}
urlConfigPagination := constants.ParsePaginationParamsWithFilter(c, urlConfigFilter)

userFilter := &dto.UserFilter{}
userPagination := constants.ParsePaginationParamsWithFilter(c, userFilter)
```

### Error Handling
```go
if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, constants.BuildErrorResponse("Invalid request", err.Error()))
    return
}
```

### Success Response
```go
c.JSON(status, constants.BuildSuccessResponse("Create successful"))
```

## Benefits

1. **Consistency**: All responses follow the same format across the application
2. **Maintainability**: Centralized response builders make changes easier
3. **Type Safety**: Strongly typed response structures
4. **Documentation**: Clear structure for API consumers
5. **Testing**: Easier to mock and test response formats

## Migration Guide

When adding new endpoints:

1. Import the constants package
2. Use appropriate response builder functions
3. Follow the established patterns for error handling
4. Include proper HTTP status codes

```go
import "github.com/surdiana/gateway/internal/constants"

// List response
c.JSON(http.StatusOK, constants.BuildListResponse(total, page, pageTotal, data))

// Error response
c.JSON(status, constants.BuildErrorResponse("Error message", details))

// Success response
c.JSON(status, constants.BuildSuccessResponse("Success message"))
```