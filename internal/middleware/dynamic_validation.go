package middleware

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Payphone-Digital/gateway/internal/constants"
	"github.com/Payphone-Digital/gateway/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// VariableValidation represents validation rules for a variable
type VariableValidation struct {
	Value          interface{}            `json:"value"`
	Encoding       string                 `json:"encoding"`
	DataType       string                 `json:"data_type"`
	IsRequired         bool                   `json:"is_required"`
	Validations        map[string]interface{} `json:"validations"`
	ValidationMessages map[string]string      `json:"validation_messages"`
	CustomMessage      string                 `json:"custom_message"`
}

// DynamicValidator validates requests based on variables configuration
type DynamicValidator struct {
	variables map[string]VariableValidation
}

// NewDynamicValidator creates a new dynamic validator from variables
func NewDynamicValidator(variables map[string]interface{}) *DynamicValidator {
	validationVars := make(map[string]VariableValidation)
	
	for name, value := range variables {
		// Convert to VariableValidation
		jsonBytes, _ := json.Marshal(value)
		var varVal VariableValidation
		if err := json.Unmarshal(jsonBytes, &varVal); err == nil {
			// Only add if has validation rules or is required
			if varVal.IsRequired || len(varVal.Validations) > 0 {
				validationVars[name] = varVal
			}
		}
	}
	
	return &DynamicValidator{variables: validationVars}
}

// Validate returns a middleware that validates the request body
func (dv *DynamicValidator) Validate() gin.HandlerFunc {
	return func(c *gin.Context) {
		if dv.ValidateRequest(c) {
			c.Next()
		}
	}
}

// ValidateRequest validates the request and returns true if valid
func (dv *DynamicValidator) ValidateRequest(c *gin.Context) bool {
	// Skip validation if no rules
	if len(dv.variables) == 0 {
		return true
	}

	// Read body
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.GetLogger().Error("Failed to read request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Failed to read request body",
		})
		c.Abort()
		return false
	}

	// Restore body for further handlers
	c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	// Parse JSON body
	var requestData map[string]interface{}
	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &requestData); err != nil {
			logger.GetLogger().Error("Invalid JSON", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			c.Abort()
			return false
		}
	} else {
		requestData = make(map[string]interface{})
	}

	// Validate against rules
	errors := dv.ValidateData(requestData)

	if len(errors) > 0 {
		logger.GetLogger().Warn("Request validation failed",
			zap.String("path", c.Request.URL.Path),
			zap.Any("validation_errors", errors),
		)

		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"message": "Unprocessable Entity",
			"errors":  errors,
		})
		c.Abort()
		return false
	}

	return true
}

// getFloat safely gets a float64 from an interface
func getFloat(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case float32:
		return float64(val), true
	case string:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

// ValidateData validates the request data against all variable rules
func (dv *DynamicValidator) ValidateData(data map[string]interface{}) map[string][]map[string]string {
	errors := make(map[string][]map[string]string)

	for fieldName, varDef := range dv.variables {
		fieldErrors := dv.validateField(data, fieldName, varDef)
		if len(fieldErrors) > 0 {
			errors[toSnakeCase(fieldName)] = fieldErrors
		}
	}

	return errors
}

// validateField validates a single field
func (dv *DynamicValidator) validateField(data map[string]interface{}, fieldName string, varDef VariableValidation) []map[string]string {
	var errors []map[string]string

	value, exists := data[fieldName]

	// Check required
	if varDef.IsRequired && (!exists || value == nil || value == "") {
		msg := varDef.CustomMessage
		if msg == "" {
			msg = fmt.Sprintf("%s wajib diisi", fieldName)
		}
		errors = append(errors, map[string]string{
			"code":    constants.ValidationRequired,
			"message": msg,
		})
		return errors
	}

	// Skip validation if field doesn't exist and not required
	if !exists || value == nil {
		return errors
	}

	// Validate based on data type
	switch varDef.DataType {
	case constants.DataTypeString:
		errors = append(errors, dv.validateString(value, fieldName, varDef)...)
	case constants.DataTypeNumber:
		errors = append(errors, dv.validateNumber(value, fieldName, varDef)...)
	case constants.DataTypeBoolean:
		errors = append(errors, dv.validateBoolean(value, fieldName, varDef)...)
	case constants.DataTypeArray:
		errors = append(errors, dv.validateArray(value, fieldName, varDef)...)
	case constants.DataTypeObject:
		errors = append(errors, dv.validateObject(value, fieldName, varDef)...)
	case constants.DataTypeDate:
		errors = append(errors, dv.validateString(value, fieldName, varDef)...)
	}

	// Generic validations for all types
	errors = append(errors, dv.validateEnum(value, fieldName, varDef)...)

	return errors
}

// validateString validates string fields
func (dv *DynamicValidator) validateString(value interface{}, fieldName string, varDef VariableValidation) []map[string]string {
	var errors []map[string]string

	str, ok := value.(string)
	if !ok {
		errors = append(errors, map[string]string{
			"code":    "type",
			"message": fmt.Sprintf("%s harus berupa string", fieldName),
		})
		return errors
	}

	// Min length
	if minVal, ok := getFloat(varDef.Validations[constants.ValidationMin]); ok && minVal > 0 && len(str) < int(minVal) {
		msg := varDef.ValidationMessages[constants.ValidationMin]
		if msg == "" {
			msg = fmt.Sprintf("%s minimal %d karakter", fieldName, int(minVal))
		}
		errors = append(errors, map[string]string{
			"code":    constants.ValidationMin,
			"message": msg,
		})
	}

	// Max length
	if maxVal, ok := getFloat(varDef.Validations[constants.ValidationMax]); ok && maxVal > 0 && len(str) > int(maxVal) {
		msg := varDef.ValidationMessages[constants.ValidationMax]
		if msg == "" {
			msg = fmt.Sprintf("%s maksimal %d karakter", fieldName, int(maxVal))
		}
		errors = append(errors, map[string]string{
			"code":    constants.ValidationMax,
			"message": msg,
		})
	}

	// Email validation
	if email, ok := varDef.Validations[constants.ValidationEmail].(bool); ok && email {
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(str) {
			msg := varDef.ValidationMessages[constants.ValidationEmail]
			if msg == "" {
				msg = fmt.Sprintf("%s harus berupa email yang valid", fieldName)
			}
			errors = append(errors, map[string]string{
				"code":    constants.ValidationEmail,
				"message": msg,
			})
		}
	}

	// URL validation
	if urlCheck, ok := varDef.Validations[constants.ValidationURL].(bool); ok && urlCheck {
		if !strings.HasPrefix(str, "http://") && !strings.HasPrefix(str, "https://") {
			errors = append(errors, map[string]string{
				"code":    constants.ValidationURL,
				"message": fmt.Sprintf("%s harus berupa URL yang valid", fieldName),
			})
		}
	}

	// UUID validation
	if uuidCheck, ok := varDef.Validations[constants.ValidationUUID].(bool); ok && uuidCheck {
		uuidRegex := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
		if !uuidRegex.MatchString(str) {
			msg := varDef.ValidationMessages[constants.ValidationUUID]
			if msg == "" {
				msg = fmt.Sprintf("%s harus berupa UUID yang valid", fieldName)
			}
			errors = append(errors, map[string]string{
				"code":    constants.ValidationUUID,
				"message": msg,
			})
		}
	}

	// Date validation (ISO8601/RFC3339)
	dateCheck, hasDateRule := varDef.Validations[constants.ValidationDate].(bool)
	if (hasDateRule && dateCheck) || varDef.DataType == constants.DataTypeDate {
		// Try parsing YYYY-MM-DD and RFC3339
		_, err1 := time.Parse("2006-01-02", str)
		_, err2 := time.Parse(time.RFC3339, str)
		if err1 != nil && err2 != nil {
			msg := varDef.ValidationMessages[constants.ValidationDate]
			if msg == "" {
				msg = fmt.Sprintf("%s harus berupa tanggal yang valid (YYYY-MM-DD atau ISO8601)", fieldName)
			}
			errors = append(errors, map[string]string{
				"code":    constants.ValidationDate,
				"message": msg,
			})
		}
	}

	// IP validation
	if ipCheck, ok := varDef.Validations[constants.ValidationIP].(bool); ok && ipCheck {
		ip := net.ParseIP(str)
		if ip == nil {
			msg := varDef.ValidationMessages[constants.ValidationIP]
			if msg == "" {
				msg = fmt.Sprintf("%s harus berupa IP Address yang valid", fieldName)
			}
			errors = append(errors, map[string]string{
				"code":    constants.ValidationIP,
				"message": msg,
			})
		}
	}

	// Numeric validation (string contains only numbers)
	if numeric, ok := varDef.Validations[constants.ValidationNumeric].(bool); ok && numeric {
		numericRegex := regexp.MustCompile(`^[0-9]+$`)
		if !numericRegex.MatchString(str) {
			errors = append(errors, map[string]string{
				"code":    constants.ValidationNumeric,
				"message": fmt.Sprintf("%s harus berupa angka", fieldName),
			})
		}
	}

	// Alpha validation (string contains only letters)
	if alpha, ok := varDef.Validations[constants.ValidationAlpha].(bool); ok && alpha {
		alphaRegex := regexp.MustCompile(`^[a-zA-Z]+$`)
		if !alphaRegex.MatchString(str) {
			errors = append(errors, map[string]string{
				"code":    constants.ValidationAlpha,
				"message": fmt.Sprintf("%s harus berupa huruf saja", fieldName),
			})
		}
	}

	// Alphanumeric validation (string contains only letters and numbers)
	if alphanumeric, ok := varDef.Validations[constants.ValidationAlphanumeric].(bool); ok && alphanumeric {
		alphanumericRegex := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
		if !alphanumericRegex.MatchString(str) {
			errors = append(errors, map[string]string{
				"code":    constants.ValidationAlphanumeric,
				"message": fmt.Sprintf("%s harus berupa huruf dan angka saja", fieldName),
			})
		}
	}

	// Pattern validation
	if pattern, ok := varDef.Validations[constants.ValidationPattern].(string); ok && pattern != "" {
		re, err := regexp.Compile(pattern)
		if err == nil && !re.MatchString(str) {
			msg := varDef.ValidationMessages[constants.ValidationPattern]
			if msg == "" {
				msg = fmt.Sprintf("%s format tidak sesuai", fieldName)
			}
			errors = append(errors, map[string]string{
				"code":    constants.ValidationPattern,
				"message": msg,
			})
		}
	}

	return errors
}

// validateEnum checks if value is in the allowed list
func (dv *DynamicValidator) validateEnum(value interface{}, fieldName string, varDef VariableValidation) []map[string]string {
	var errors []map[string]string

	// Enum validation
	if enumValues, ok := varDef.Validations[constants.ValidationEnum].([]interface{}); ok {
		found := false
		for _, enumVal := range enumValues {
			// Simple equality check
			if fmt.Sprintf("%v", enumVal) == fmt.Sprintf("%v", value) {
				found = true
				break
			}
		}

		if !found {
			allowed := make([]string, len(enumValues))
			for i, v := range enumValues {
				allowed[i] = fmt.Sprintf("%v", v)
			}
			msg := varDef.ValidationMessages[constants.ValidationEnum]
			if msg == "" {
				msg = fmt.Sprintf("%s harus salah satu dari: %s", fieldName, strings.Join(allowed, ", "))
			}
			errors = append(errors, map[string]string{
				"code":    constants.ValidationEnum,
				"message": msg,
			})
		}
	}
	return errors
}

// validateNumber validates numeric fields
func (dv *DynamicValidator) validateNumber(value interface{}, fieldName string, varDef VariableValidation) []map[string]string {
	var errors []map[string]string

	var num float64
	switch v := value.(type) {
	case float64:
		num = v
	case int:
		num = float64(v)
	case string:
		// Try to parse as number
		if _, err := fmt.Sscanf(v, "%f", &num); err != nil {
			errors = append(errors, map[string]string{
				"code":    "type",
				"message": fmt.Sprintf("%s harus berupa angka", fieldName),
			})
			return errors
		}
	default:
		errors = append(errors, map[string]string{
			"code":    "type",
			"message": fmt.Sprintf("%s harus berupa angka", fieldName),
		})
		return errors
	}

	// Min value
	if minVal, ok := getFloat(varDef.Validations[constants.ValidationMin]); ok && num < minVal {
		errors = append(errors, map[string]string{
			"code":    constants.ValidationMin,
			"message": fmt.Sprintf("%s minimal %v", fieldName, minVal),
		})
	}

	// Max value
	if maxVal, ok := getFloat(varDef.Validations[constants.ValidationMax]); ok && num > maxVal {
		errors = append(errors, map[string]string{
			"code":    constants.ValidationMax,
			"message": fmt.Sprintf("%s maksimal %v", fieldName, maxVal),
		})
	}

	return errors
}

// validateBoolean validates boolean fields
func (dv *DynamicValidator) validateBoolean(value interface{}, fieldName string, varDef VariableValidation) []map[string]string {
	var errors []map[string]string

	_, ok := value.(bool)
	if !ok {
		errors = append(errors, map[string]string{
			"code":    "type",
			"message": fmt.Sprintf("%s harus berupa boolean", fieldName),
		})
	}

	return errors
}

// validateArray validates array fields
func (dv *DynamicValidator) validateArray(value interface{}, fieldName string, varDef VariableValidation) []map[string]string {
	var errors []map[string]string

	arr, ok := value.([]interface{})
	if !ok {
		errors = append(errors, map[string]string{
			"code":    "type",
			"message": fmt.Sprintf("%s harus berupa array", fieldName),
		})
		return errors
	}

	// Min items
	if minItems, ok := getFloat(varDef.Validations[constants.ValidationMinItems]); ok && len(arr) < int(minItems) {
		errors = append(errors, map[string]string{
			"code":    constants.ValidationMinItems,
			"message": fmt.Sprintf("%s harus memiliki minimal %d item", fieldName, int(minItems)),
		})
	}

	// Max items
	if maxItems, ok := getFloat(varDef.Validations[constants.ValidationMaxItems]); ok && len(arr) > int(maxItems) {
		errors = append(errors, map[string]string{
			"code":    constants.ValidationMaxItems,
			"message": fmt.Sprintf("%s maksimal %d item",fieldName, int(maxItems)),
		})
	}

	return errors
}

// validateObject validates object fields
func (dv *DynamicValidator) validateObject(value interface{}, fieldName string, varDef VariableValidation) []map[string]string {
	var errors []map[string]string

	_, ok := value.(map[string]interface{})
	if !ok {
		errors = append(errors, map[string]string{
			"code":    "type",
			"message": fmt.Sprintf("%s harus berupa object", fieldName),
		})
	}

	return errors
}
