package constants

// Pagination Query Parameters
const (
	QueryParamPage   = "page"
	QueryParamLimit  = "limit"
	QueryParamSearch = "search"
	QueryParamSort   = "sort"
	QueryParamOrder  = "order"
)

// Default Pagination Values (as strings for query parsing)
const (
	DefaultPage   = "1"
	DefaultLimit  = "10"
	DefaultSearch = ""
	DefaultSort   = "id"
	DefaultOrder  = "asc"
)

// Pagination Limits (as integers for validation)
const (
	MinPage       = 1
	MinLimit      = 1
	MaxLimit      = 100
	DefaultOffset = 0
)

// Sort Orders
const (
	OrderAsc  = "asc"
	OrderDesc = "desc"
)
