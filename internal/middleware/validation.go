package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/surdiana/gateway/pkg/logger"
	"github.com/surdiana/gateway/pkg/validation"
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
				"message": "Format JSON tidak valid",
				"details": err.Error(),
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
			var validationErrors []string

			for _, e := range err.(validator.ValidationErrors) {
				logger.GetLogger().Debug("Middleware: Validation error occurred",
					zap.String("client_ip", clientIP),
					zap.String("field", e.Field()),
					zap.String("tag", e.Tag()),
					zap.String("param", e.Param()),
				)
				// Ambil custom message jika ada
				fmt.Println("fieldMessages", validation.CustomMessage(e.Field()))
				if fieldMessages := validation.CustomMessage(e.Field()); fieldMessages != nil {
					if msg, exists := fieldMessages[e.Tag()]; exists {
						validationErrors = append(validationErrors, msg)
					}
				} else {
					validationErrors = append(validationErrors, validation.DefaultMessage(e.Field(), e.Tag()))
				}
			}

			logger.GetLogger().Warn("Middleware: Request validation failed",
				zap.String("client_ip", clientIP),
				zap.String("path", c.Request.URL.Path),
				zap.Strings("validation_errors", validationErrors),
				zap.Int("error_count", len(validationErrors)),
			)

			c.JSON(http.StatusBadRequest, gin.H{
				"message": "Validasi gagal",
				"details": validationErrors,
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

// func formatValidationError(e validator.FieldError) string {
// 	fieldName := strings.ToLower(e.Field())

// 	switch e.Tag() {
// 	case "required":
// 		return fmt.Sprintf("%s tidak boleh kosong", fieldName)
// 	case "email":
// 		return "Format email tidak valid"
// 	case "min":
// 		return fmt.Sprintf("%s minimal harus %s karakter", fieldName, e.Param())
// 	case "max":
// 		return fmt.Sprintf("%s maksimal boleh %s karakter", fieldName, e.Param())
// 	case "len":
// 		return fmt.Sprintf("%s harus tepat %s karakter", fieldName, e.Param())
// 	case "password":
// 		return "Password harus minimal 8 karakter dan mengandung huruf besar, huruf kecil, angka, dan karakter khusus"
// 	case "phone":
// 		return "Nomor telepon tidak valid (contoh format: +628123456789, 08123456789)"
// 	case "name":
// 		return "Nama hanya boleh berisi huruf, spasi, dan tanda hubung (2-50 karakter)"
// 	case "referral_code":
// 		return "Kode referral harus 6-8 karakter, hanya huruf dan angka, dan dalam huruf besar"
// 	case "eqfield":
// 		compareField := strings.ToLower(e.Param())
// 		return fmt.Sprintf("%s harus sama dengan %s", fieldName, compareField)
// 	default:
// 		return fmt.Sprintf("%s tidak valid: %s", fieldName, e.Tag())
// 	}
// }
