package constants

// Field Length Limits
const (
	MinPasswordLength = 8
	MaxPasswordLength = 100
	MinNameLength     = 2
	MaxNameLength     = 50
	MinPhoneLength    = 10
	MaxPhoneLength    = 15
	MaxEmailLength    = 255
	MaxDescLength     = 500
	MaxURLLength      = 2048
)

// Token Settings (in seconds)
const (
	AccessTokenExpiry  = 15 * 60          // 15 minutes
	RefreshTokenExpiry = 7 * 24 * 60 * 60 // 7 days
)

// Validation Patterns
const (
	EmailPattern = `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	PhonePattern = `^\+?[1-9]\d{1,14}$` // E.164 format
)

// Dynamic Validation Rules
const (
	ValidationRequired     = "required"
	ValidationMin          = "min"
	ValidationMax          = "max"
	ValidationEmail        = "email"
	ValidationURL          = "url"
	ValidationPattern      = "pattern"
	ValidationNumeric      = "numeric"
	ValidationAlpha        = "alpha"
	ValidationAlphanumeric = "alphanumeric"
	ValidationMinItems     = "minItems"
	ValidationMaxItems     = "maxItems"
	ValidationUUID         = "uuid"
	ValidationDate         = "date"
	ValidationIP           = "ip"
	ValidationEnum         = "enum"
)
