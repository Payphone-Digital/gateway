package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Payphone-Digital/gateway/internal/constants"
	"github.com/Payphone-Digital/gateway/internal/dto"
	apperrors "github.com/Payphone-Digital/gateway/internal/errors"
	"github.com/Payphone-Digital/gateway/internal/service"
	ctxutil "github.com/Payphone-Digital/gateway/pkg/context"
	"github.com/Payphone-Digital/gateway/pkg/logger"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(service *service.UserService) *UserHandler {
	return &UserHandler{userService: service}
}

func (h *UserHandler) GetByID(c *gin.Context) {
	// Create context with request information
	ctx := ctxutil.NewContextWithRequest(c.Request.Context(), c.Request, "handler", "GetByID")

	id := c.Param("id")

	logger.InfoWithContext(ctx, "Get user by ID request").
		String("raw_id", id).
		Log()

	userID, err := strconv.Atoi(id)
	if err != nil {
		logger.WarnWithContext(ctx, "Invalid user ID format").
			String("raw_id", id).
			Err(err).
			Log()
		c.JSON(http.StatusBadRequest, constants.BuildErrorResponse("Invalid user ID", ""))
		return
	}

	logger.DebugWithContext(ctx, "Processing get user by ID").
		Int("user_id", userID).
		Log()

	user, err := h.userService.GetByID(ctx, uint(userID))
	if err != nil {
		status := apperrors.ToHTTPStatus(err)
		logger.ErrorWithContext(ctx, "Failed to fetch user").
			Int("user_id", userID).
			Int("http_status", status).
			Err(err).
			Log()
		c.JSON(status, constants.BuildErrorResponse("Failed to fetch user", err.Error()))
		return
	}

	logger.InfoWithContext(ctx, "User fetched successfully").
		Int("user_id", userID).
		Bool("user_exists", user != nil).
		Log()

	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) GetAll(c *gin.Context) {
	// Create context with request information
	ctx := ctxutil.NewContextWithRequest(c.Request.Context(), c.Request, "handler", "GetAll")

	// Parse pagination parameters
	pagination := constants.ParsePaginationParams(c)

	// Get search parameter directly from query for basic pagination
	search := c.DefaultQuery(constants.QueryParamSearch, constants.DefaultSearch)

	logger.InfoWithContext(ctx, "Get all users request").
		Int("page", pagination.Page).
		Int("limit", pagination.Limit).
		Int("offset", pagination.Offset).
		String("search", search).
		Log()

	res, total, total_verify, total_unverify, pageTotal, err := h.userService.GetAll(ctx, pagination.Limit, pagination.Offset, search)
	if err != nil {
		status := apperrors.ToHTTPStatus(err)
		logger.ErrorWithContext(ctx, "Failed to fetch all users").
			Int("page", pagination.Page).
			Int("limit", pagination.Limit).
			String("search", search).
			Int("http_status", status).
			Err(err).
			Log()
		c.JSON(status, constants.BuildErrorResponse("Failed to fetch pages", err.Error()))
	} else {
		logger.InfoWithContext(ctx, "Users fetched successfully").
			Int("page", pagination.Page).
			Int("limit", pagination.Limit).
			String("search", search).
			Int64("total", total).
			Int64("total_verify", total_verify).
			Int64("total_unverify", total_unverify).
			Int("page_total", pageTotal).
			Int("returned_count", len(res)).
			Log()

		c.JSON(http.StatusOK, constants.BuildListResponse(total, pagination.Page, pageTotal, res))
	}
}

// CreateUser creates a new user
func (h *UserHandler) CreateUser(c *gin.Context) {
	ctx := ctxutil.NewContextWithRequest(c.Request.Context(), c.Request, "handler", "CreateUser")

	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.WarnWithContext(ctx, "Invalid request body for user creation").
			Err(err).
			Log()
		c.JSON(http.StatusBadRequest, constants.BuildErrorResponse("Invalid request format", err.Error()))
		return
	}

	logger.InfoWithContext(ctx, "Create user request").
		String("email", req.Email).
		String("first_name", req.FirstName).
		String("last_name", req.LastName).
		Log()

	user, err := h.userService.CreateUser(ctx, &req)
	if err != nil {
		status := apperrors.ToHTTPStatus(err)
		logger.ErrorWithContext(ctx, "Failed to create user").
			String("email", req.Email).
			Int("http_status", status).
			Err(err).
			Log()
		c.JSON(status, constants.BuildErrorResponse("Failed to create user", err.Error()))
		return
	}

	logger.InfoWithContext(ctx, "User created successfully").
		String("email", req.Email).
		Int("user_id", int(user.ID)).
		Log()

	c.JSON(http.StatusCreated, user)
}

// UpdateUser updates user information (excluding email)
func (h *UserHandler) UpdateUser(c *gin.Context) {
	ctx := ctxutil.NewContextWithRequest(c.Request.Context(), c.Request, "handler", "UpdateUser")

	idStr := c.Param("id")
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.WarnWithContext(ctx, "Invalid user ID format").
			String("raw_id", idStr).
			Err(err).
			Log()
		c.JSON(http.StatusBadRequest, constants.BuildErrorResponse("Invalid user ID", ""))
		return
	}
	id := uint(id64)

	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.WarnWithContext(ctx, "Invalid request body for user update").
			Int("user_id", int(id)).
			Err(err).
			Log()
		c.JSON(http.StatusBadRequest, constants.BuildErrorResponse("Invalid request format", err.Error()))
		return
	}

	logger.InfoWithContext(ctx, "Update user request").
		Int("user_id", int(id)).
		String("first_name", req.FirstName).
		String("last_name", req.LastName).
		String("phone", req.Phone).
		Log()

	user, err := h.userService.UpdateUser(ctx, id, &req)
	if err != nil {
		status := apperrors.ToHTTPStatus(err)
		logger.ErrorWithContext(ctx, "Failed to update user").
			Int("user_id", int(id)).
			Int("http_status", status).
			Err(err).
			Log()
		c.JSON(status, constants.BuildErrorResponse("Failed to update user", err.Error()))
		return
	}

	logger.InfoWithContext(ctx, "User updated successfully").
		Int("user_id", int(id)).
		Log()

	c.JSON(http.StatusOK, user)
}

// UpdatePassword updates user password with current password verification
func (h *UserHandler) UpdatePassword(c *gin.Context) {
	ctx := ctxutil.NewContextWithRequest(c.Request.Context(), c.Request, "handler", "UpdatePassword")

	idStr := c.Param("id")
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.WarnWithContext(ctx, "Invalid user ID format").
			String("raw_id", idStr).
			Err(err).
			Log()
		c.JSON(http.StatusBadRequest, constants.BuildErrorResponse("Invalid user ID", ""))
		return
	}
	id := uint(id64)

	var req dto.UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.WarnWithContext(ctx, "Invalid request body for password update").
			Int("user_id", int(id)).
			Err(err).
			Log()
		c.JSON(http.StatusBadRequest, constants.BuildErrorResponse("Invalid request format", err.Error()))
		return
	}

	logger.InfoWithContext(ctx, "Update password request").
		Int("user_id", int(id)).
		Log()

	if err := h.userService.UpdatePassword(ctx, id, &req); err != nil {
		status := apperrors.ToHTTPStatus(err)
		logger.ErrorWithContext(ctx, "Failed to update password").
			Int("user_id", int(id)).
			Err(err).
			Log()

		// Map domain errors to user-friendly messages
		var errorMessage string
		switch {
		case errors.Is(err, apperrors.ErrIncorrectPassword):
			errorMessage = "Current password is incorrect"
		case errors.Is(err, apperrors.ErrPasswordMismatch):
			errorMessage = "New password and confirmation do not match"
		case errors.Is(err, apperrors.ErrUserNotFound):
			errorMessage = "User not found"
		default:
			errorMessage = "Failed to update password"
		}

		c.JSON(status, constants.BuildErrorResponse(errorMessage, apperrors.GetErrorMessage(err)))
		return
	}

	logger.InfoWithContext(ctx, "Password updated successfully").
		Int("user_id", int(id)).
		Log()

	c.JSON(http.StatusOK, constants.BuildSuccessResponse("Password updated successfully"))
}

// DeleteUser performs hard delete on user with security validations
func (h *UserHandler) DeleteUser(c *gin.Context) {
	ctx := ctxutil.NewContextWithRequest(c.Request.Context(), c.Request, "handler", "DeleteUser")

	// Get target user ID from URL parameter
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		logger.WarnWithContext(ctx, "Invalid user ID in delete request").
			String("id_param", idParam).
			Err(err).
			Log()
		c.JSON(http.StatusBadRequest, constants.BuildErrorResponse("Invalid user ID", err.Error()))
		return
	}

	// Get requesting user ID from JWT context
	requestingUserIDInterface, exists := c.Get("user_id")
	if !exists {
		logger.ErrorWithContext(ctx, "User ID not found in JWT context").
			Log()
		c.JSON(http.StatusUnauthorized, constants.BuildErrorResponse("User authentication failed", ""))
		return
	}

	var requestingUserID uint
	switch v := requestingUserIDInterface.(type) {
	case uint:
		requestingUserID = v
	case float64:
		requestingUserID = uint(v)
	default:
		logger.ErrorWithContext(ctx, "Invalid user ID type in JWT context").
			String("type", fmt.Sprintf("%T", requestingUserIDInterface)).
			Log()
		c.JSON(http.StatusUnauthorized, constants.BuildErrorResponse("User authentication failed", ""))
		return
	}

	logger.InfoWithContext(ctx, "Delete user request received").
		Int("target_user_id", int(id)).
		Int("requesting_user_id", int(requestingUserID)).
		Log()

	// Perform hard delete with security validations
	if err := h.userService.DeleteUser(ctx, uint(id), requestingUserID); err != nil {
		logger.ErrorWithContext(ctx, "Failed to delete user").
			Int("target_user_id", int(id)).
			Int("requesting_user_id", int(requestingUserID)).
			Err(err).
			Log()

		var errorMessage string
		switch {
		case errors.Is(err, apperrors.ErrSelfDeletion):
			errorMessage = "Users cannot delete themselves"
			status := http.StatusForbidden
			c.JSON(status, constants.BuildErrorResponse(errorMessage, ""))
			return
		case errors.Is(err, apperrors.ErrUserNotFound):
			errorMessage = "User not found"
			status := http.StatusNotFound
			c.JSON(status, constants.BuildErrorResponse(errorMessage, ""))
			return
		default:
			errorMessage = "Failed to delete user"
			status := apperrors.ToHTTPStatus(err)
			c.JSON(status, constants.BuildErrorResponse(errorMessage, apperrors.GetErrorMessage(err)))
			return
		}
	}

	logger.InfoWithContext(ctx, "User deleted successfully").
		Int("target_user_id", int(id)).
		Int("requesting_user_id", int(requestingUserID)).
		Log()

	c.JSON(http.StatusOK, constants.BuildSuccessResponse("User deleted successfully"))
}
