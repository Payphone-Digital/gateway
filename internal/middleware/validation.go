package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/Payphone-Digital/gateway/pkg/logger"
	"github.com/Payphone-Digital/gateway/pkg/validation"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type ValidationMiddleware struct {
	validate *validator.Validate
}

func NewValidationMiddleware() *ValidationMiddleware {
	validate := validator.New()
	return &ValidationMiddleware{validate: validate}
}

func (m *ValidationMiddleware) ValidateRequestBody(factory func() interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		userAgent := c.GetHeader("User-Agent")
		contentType := c.GetHeader("Content-Type")

		logger.GetLogger().Debug("Middleware: Validation request processing",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("client_ip", clientIP),
			zap.String("user_agent", userAgent),
			zap.String("content_type", contentType),
		)

		var bodyBytes []byte
		if c.Request.Body != nil {
			var err error
			bodyBytes, err = io.ReadAll(c.Request.Body)
			if err != nil {
				logger.GetLogger().Error("Middleware: Failed to read request body",
					zap.String("client_ip", clientIP),
					zap.String("path", c.Request.URL.Path),
					zap.Error(err),
				)
				c.JSON(http.StatusBadRequest, gin.H{
					"message": "Gagal membaca request body",
				})
				c.Abort()
				return
			}
		}

		// Restore body untuk dapat dibaca kembali
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		request := factory()

		if err := json.Unmarshal(bodyBytes, request); err != nil {
			logger.GetLogger().Error("Middleware: JSON unmarshaling failed",
				zap.String("client_ip", clientIP),
				zap.String("path", c.Request.URL.Path),
				zap.Int("body_size", len(bodyBytes)),
				zap.Error(err),
			)
			c.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			c.Abort()
			return
		}

		logger.GetLogger().Debug("Middleware: JSON unmarshaling successful",
			zap.String("client_ip", clientIP),
			zap.String("path", c.Request.URL.Path),
			zap.Int("body_size", len(bodyBytes)),
		)

		if err := m.validate.Struct(request); err != nil {
			validationErrors := make(map[string][]map[string]string)

			for _, e := range err.(validator.ValidationErrors) {
				logger.GetLogger().Debug("Middleware: Validation error occurred",
					zap.String("client_ip", clientIP),
					zap.String("field", e.Field()),
					zap.String("tag", e.Tag()),
					zap.String("param", e.Param()),
				)

				// Determine field name (convert to snake_case equivalent or use lowercase)
				fieldName := toSnakeCase(e.Field())

				// Get error message
				var message string
				if fieldMessages := validation.CustomMessage(e.Field()); fieldMessages != nil {
					if msg, exists := fieldMessages[e.Tag()]; exists {
						message = msg
					}
				}
				if message == "" {
					message = validation.DefaultMessage(e.Field(), e.Tag())
				}

				errorDetail := map[string]string{
					"code":    e.Tag(),
					"message": message,
				}

				validationErrors[fieldName] = append(validationErrors[fieldName], errorDetail)
			}

			logger.GetLogger().Warn("Middleware: Request validation failed",
				zap.String("client_ip", clientIP),
				zap.String("path", c.Request.URL.Path),
				zap.Any("validation_errors", validationErrors),
				zap.Int("error_count", len(validationErrors)),
			)

			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"message": "Unprocessable Entity",
				"errors":  validationErrors,
			})
			c.Abort()
			return
		}

		logger.GetLogger().Debug("Middleware: Request validation successful",
			zap.String("client_ip", clientIP),
			zap.String("path", c.Request.URL.Path),
		)

		c.Next()
	}
}

func toSnakeCase(str string) string {
	var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

