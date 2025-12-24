package handler

import (
	"fmt"
	"net/http"

	"github.com/Payphone-Digital/gateway/internal/constants"
	"github.com/Payphone-Digital/gateway/internal/dto"
	apperrors "github.com/Payphone-Digital/gateway/internal/errors"
	"github.com/Payphone-Digital/gateway/internal/service"
	ctxutil "github.com/Payphone-Digital/gateway/pkg/context"
	"github.com/Payphone-Digital/gateway/pkg/logger"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	userService *service.UserService
}

func NewAuthHandler(userService *service.UserService) *AuthHandler {
	return &AuthHandler{
		userService: userService,
	}
}

// Login handles user authentication
func (h *AuthHandler) Login(c *gin.Context) {
	ctx := ctxutil.NewContextWithRequest(c.Request.Context(), c.Request, "handler", "Login")

	var req dto.UserLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.WarnWithContext(ctx, "Invalid login request").
			Err(err).
			Log()
		c.JSON(http.StatusBadRequest, constants.BuildErrorResponse("Invalid request format", err.Error()))
		return
	}

	logger.InfoWithContext(ctx, "User login attempt").
		String("email", req.Email).
		Log()

	response, err := h.userService.LoginUser(ctx, req.Email, req.Password)
	if err != nil {
		logger.WarnWithContext(ctx, "Login failed").
			String("email", req.Email).
			Err(err).
			Log()
		status := apperrors.ToHTTPStatus(err)
		c.JSON(status, constants.BuildErrorResponse("Authentication failed", apperrors.GetErrorMessage(err)))
		return
	}

	logger.InfoWithContext(ctx, "User logged in successfully").
		String("email", req.Email).
		Int("user_id", int(response.User.ID)).
		Log()

	c.JSON(http.StatusOK, response)
}

// RefreshToken handles JWT token refresh using refresh token
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	ctx := ctxutil.NewContextWithRequest(c.Request.Context(), c.Request, "handler", "RefreshToken")

	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.WarnWithContext(ctx, "Invalid refresh token request").
			Err(err).
			Log()
		c.JSON(http.StatusBadRequest, constants.BuildErrorResponse("Invalid request format", err.Error()))
		return
	}

	logger.InfoWithContext(ctx, "Token refresh attempt").
		String("token_length", fmt.Sprintf("%d", len(req.RefreshToken))).
		Log()

	response, err := h.userService.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		logger.WarnWithContext(ctx, "Token refresh failed").
			Err(err).
			Log()
		status := apperrors.ToHTTPStatus(err)
		c.JSON(status, constants.BuildErrorResponse("Token refresh failed", apperrors.GetErrorMessage(err)))
		return
	}

	logger.InfoWithContext(ctx, "Token refreshed successfully").
		Int("user_id", int(response.User.ID)).
		Log()

	c.JSON(http.StatusOK, response)
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *gin.Context) {
	ctx := ctxutil.NewContextWithRequest(c.Request.Context(), c.Request, "handler", "Logout")

	// Get user info from context (set by JWT middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		logger.WarnWithContext(ctx, "User not found in context during logout").
			Log()
		c.JSON(http.StatusUnauthorized, constants.BuildErrorResponse("Unauthorized", "User not found in context"))
		return
	}

	// Convert userID to uint
	var userIDUint uint
	switch v := userID.(type) {
	case uint:
		userIDUint = v
	case float64:
		userIDUint = uint(v)
	default:
		logger.ErrorWithContext(ctx, "Invalid user ID type in context").
			String("type", fmt.Sprintf("%T", userID)).
			Log()
		c.JSON(http.StatusUnauthorized, constants.BuildErrorResponse("Unauthorized", "Invalid user type"))
		return
	}

	logger.InfoWithContext(ctx, "User logout attempt").
		Int("user_id", int(userIDUint)).
		Log()

	// Invalidate user tokens (increment version and clear refresh token)
	if err := h.userService.LogoutUser(ctx, userIDUint); err != nil {
		logger.ErrorWithContext(ctx, "Failed to logout user").
			Int("user_id", int(userIDUint)).
			Err(err).
			Log()
		status := apperrors.ToHTTPStatus(err)
		c.JSON(status, constants.BuildErrorResponse("Logout failed", apperrors.GetErrorMessage(err)))
		return
	}

	logger.InfoWithContext(ctx, "User logged out successfully").
		Int("user_id", int(userIDUint)).
		Log()

	c.JSON(http.StatusOK, constants.BuildSuccessResponse("Logout successful"))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
