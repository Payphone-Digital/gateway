package constants

import (
	"reflect"
	"strconv"
	"unicode"

	"github.com/gin-gonic/gin"
)

// Standard Response Field Keys
const (
	// Pagination fields
	ResponseFieldTotal     = "total"
	ResponseFieldPage      = "page"
	ResponseFieldPageTotal = "page_total"
	ResponseFieldData      = "data"

	// Common response fields
	ResponseFieldMessage = "message"
	ResponseFieldDetails = "details"
	ResponseFieldError   = "error"
	ResponseFieldSuccess = "success"
)

// Pagination Parameters Struct - Core pagination only
type PaginationParams struct {
	Page   int // Page number from user request (default: 1)
	Limit  int // Limit per page from user request (default: 10)
	Offset int // Calculated offset (page - 1) * limit
	All    any // Contains: page, limit, offset, search + all filter data
}

// ParsePaginationParams parses basic pagination parameters (page, limit only)
func ParsePaginationParams(c *gin.Context) PaginationParams {
	// Parse basic pagination parameters
	pageStr := c.DefaultQuery(QueryParamPage, DefaultPage)
	limitStr := c.DefaultQuery(QueryParamLimit, DefaultLimit)

	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)

	// Validate pagination parameters
	if page < MinPage {
		page = MinPage
	}
	if limit < MinLimit {
		limit = MinLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}

	// Calculate offset
	offset := (page - 1) * limit

	return PaginationParams{
		Page:   page,
		Limit:  limit,
		Offset: offset,
	}
}

// ParsePaginationParamsWithFilter parses pagination parameters with custom filter based on DTO struct
// Uses reflection to automatically parse query parameters based on struct fields
func ParsePaginationParamsWithFilter(c *gin.Context, filterStruct any) PaginationParams {
	// Parse basic pagination parameters first
	pagination := ParsePaginationParams(c)

	// Parse search (common parameter)
	search := c.DefaultQuery(QueryParamSearch, DefaultSearch)

	// Parse dynamic parameters based on struct fields
	dynamicParams := make(map[string]string)
	dynamicTypedParams := make(map[string]any)

	// Use reflection to get struct fields
	structValue := reflect.ValueOf(filterStruct)
	if structValue.Kind() == reflect.Ptr {
		structValue = structValue.Elem()
	}

	if structValue.Kind() == reflect.Struct {
		structType := structValue.Type()

		for i := 0; i < structType.NumField(); i++ {
			field := structType.Field(i)
			fieldName := field.Name
			queryTag := field.Tag.Get("query")

			if queryTag == "" {
				// Convert field name to snake_case for query parameter
				queryTag = toSnakeCase(fieldName)
			}

			value := c.Query(queryTag)
			if value != "" {
				dynamicParams[queryTag] = value

				// Parse based on field type
				switch field.Type.Kind() {
				case reflect.String:
					dynamicTypedParams[queryTag] = value
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					if intVal, err := strconv.Atoi(value); err == nil {
						dynamicTypedParams[queryTag] = intVal
					}
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					if uintVal, err := strconv.ParseUint(value, 10, 64); err == nil {
						dynamicTypedParams[queryTag] = uintVal
					}
				case reflect.Bool:
					dynamicTypedParams[queryTag] = value == "true" || value == "1"
				}
			}
		}
	}

	// Merge core pagination with dynamic filter data into All field
	// All field will contain: page, limit, offset, search + all DTO fields
	allParams := make(map[string]any)
	allParams["page"] = pagination.Page
	allParams["limit"] = pagination.Limit
	allParams["offset"] = pagination.Offset
	allParams["search"] = search

	// Add all dynamic params to All
	for key, value := range dynamicParams {
		allParams[key] = value
	}
	for key, value := range dynamicTypedParams {
		allParams[key] = value
	}

	return PaginationParams{
		All: allParams,
	}
}

// Helper function to convert PascalCase to snake_case
func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			result = append(result, '_')
		}
		result = append(result, unicode.ToLower(r))
	}
	return string(result)
}

// NOTE: Helper methods for dynamic parameter access have been removed
// because parsing is now handled via DTO-based approach in service layer
// All filter values are accessed directly from the parsed filter struct

// Response Format Functions
func BuildListResponse(total int64, page int, pageTotal int, data any) map[string]any {
	return map[string]any{
		ResponseFieldTotal:     total,
		ResponseFieldPage:      page,
		ResponseFieldPageTotal: pageTotal,
		ResponseFieldData:      data,
	}
}

func BuildErrorResponse(message string, details any) map[string]any {
	response := map[string]any{
		ResponseFieldMessage: message,
	}

	if details != nil {
		response[ResponseFieldDetails] = details
	}

	return response
}

func BuildSuccessResponse(message string) map[string]any {
	return map[string]any{
		ResponseFieldMessage: message,
	}
}
