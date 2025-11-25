package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/surdiana/gateway/internal/dto"
	apperrors "github.com/surdiana/gateway/internal/errors"
	"github.com/surdiana/gateway/internal/model"
	"github.com/surdiana/gateway/internal/repository"
	ctxutil "github.com/surdiana/gateway/pkg/context"
	"github.com/surdiana/gateway/pkg/logger"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	repoUser   *repository.UserRepository
	jwtService *JWTService
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repoUser: repo}
}

func NewUserServiceWithJWT(repo *repository.UserRepository, jwtService *JWTService) *UserService {
	return &UserService{
		repoUser:   repo,
		jwtService: jwtService,
	}
}

func (s *UserService) GetByID(ctx context.Context, id uint) (*dto.UserResponse, error) {
	// Add function info to context
	ctx = context.WithValue(ctx, ctxutil.FunctionKey, "GetByID")
	ctx = context.WithValue(ctx, ctxutil.ModuleKey, "service")

	logger.InfoWithContext(ctx, "Get user by ID").
		Int("user_id", int(id)).
		Log()

	user, err := s.repoUser.GetByID(ctx, int(id))
	if err != nil {
		logger.ErrorWithContext(ctx, "Failed to get user by ID").
			Int("user_id", int(id)).
			Err(err).
			Log()
		return nil, apperrors.WrapError(apperrors.ErrInternal, err)
	}

	if user == nil {
		logger.InfoWithContext(ctx, "User not found").
			Int("user_id", int(id)).
			Log()
		return nil, apperrors.ErrUserNotFound
	}

	logger.InfoWithContext(ctx, "User retrieved successfully").
		Int("user_id", int(id)).
		String("email", user.Email).
		Log()

	response := &dto.UserResponse{
		ID:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Phone:     user.Phone,
		LastLogin: user.LastLogin,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	return response, nil
}

func (s *UserService) GetAll(ctx context.Context, limit, offset int, search string) ([]dto.UserResponse, int64, int64, int64, int, error) {
	// Add function info to context
	ctx = context.WithValue(ctx, ctxutil.FunctionKey, "GetAll")
	ctx = context.WithValue(ctx, ctxutil.ModuleKey, "service")

	logger.InfoWithContext(ctx, "Get all users").
		Int("limit", limit).
		Int("offset", offset).
		String("search", search).
		Log()

	pages, total, total_verify, total_unverify, err := s.repoUser.GetAll(ctx, limit, offset, search)
	if err != nil {
		logger.ErrorWithContext(ctx, "Failed to get all users").
			Int("limit", limit).
			Int("offset", offset).
			String("search", search).
			Err(err).
			Log()
		return nil, 0, 0, 0, 0, apperrors.WrapError(apperrors.ErrInternal, err)
	}

	pageTotal := int(math.Ceil(float64(total) / float64(limit)))
	var res []dto.UserResponse
	for _, data := range pages {
		res = append(res, dto.UserResponse{
			ID:        data.ID,
			FirstName: data.FirstName,
			LastName:  data.LastName,
			Email:     data.Email,
			Phone:     data.Phone,
			LastLogin: data.LastLogin,
			CreatedAt: data.CreatedAt,
			UpdatedAt: data.UpdatedAt,
		})
	}

	logger.InfoWithContext(ctx, "Users retrieved successfully").
		Int("limit", limit).
		Int("offset", offset).
		String("search", search).
		Int64("total", total).
		Int64("total_verify", total_verify).
		Int64("total_unverify", total_unverify).
		Int("page_total", pageTotal).
		Int("returned_count", len(res)).
		Log()

	return res, total, total_verify, total_unverify, pageTotal, nil
}

// hashPassword hashes password using bcrypt
func (s *UserService) hashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedPassword), nil
}

// checkPassword verifies password against hash
func (s *UserService) checkPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// validateEmail checks if email is valid and not already used
func (s *UserService) validateEmail(ctx context.Context, email string, excludeID *uint) error {
	// Basic email format validation is handled by binding tags
	// Check if email already exists
	existingUser, err := s.repoUser.GetByEmail(ctx, email)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil // Email is available
		}
		return fmt.Errorf("failed to check email availability: %w", err)
	}

	// If we're updating a user, exclude their own email from check
	if excludeID != nil && existingUser.ID == *excludeID {
		return nil // Email belongs to the same user
	}

	return fmt.Errorf("email already exists")
}

// CreateUser creates a new user with hashed password
func (s *UserService) CreateUser(ctx context.Context, req *dto.CreateUserRequest) (*dto.UserResponse, error) {
	ctx = context.WithValue(ctx, ctxutil.FunctionKey, "CreateUser")
	ctx = context.WithValue(ctx, ctxutil.ModuleKey, "service")

	logger.InfoWithContext(ctx, "Creating new user").
		String("email", req.Email).
		String("first_name", req.FirstName).
		String("last_name", req.LastName).
		Log()

	// Validate email uniqueness
	if err := s.validateEmail(ctx, req.Email, nil); err != nil {
		logger.WarnWithContext(ctx, "Email validation failed").
			String("email", req.Email).
			Err(err).
			Log()
		return nil, apperrors.ErrEmailExists
	}

	// Hash password
	hashedPassword, err := s.hashPassword(req.Password)
	if err != nil {
		logger.ErrorWithContext(ctx, "Failed to hash password").
			String("email", req.Email).
			Err(err).
			Log()
		return nil, apperrors.WrapError(apperrors.ErrInternal, err)
	}

	// Create user model
	user := &model.User{
		FirstName: strings.TrimSpace(req.FirstName),
		LastName:  strings.TrimSpace(req.LastName),
		Email:     strings.ToLower(strings.TrimSpace(req.Email)),
		Phone:     strings.TrimSpace(req.Phone),
		Password:  hashedPassword,
	}

	// Save to database
	if err := s.repoUser.Create(ctx, user); err != nil {
		logger.ErrorWithContext(ctx, "Failed to create user").
			String("email", req.Email).
			Err(err).
			Log()
		return nil, apperrors.WrapError(apperrors.ErrInternal, err)
	}

	logger.InfoWithContext(ctx, "User created successfully").
		String("email", user.Email).
		Int("user_id", int(user.ID)).
		Log()

	response := &dto.UserResponse{
		ID:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Phone:     user.Phone,
		LastLogin: user.LastLogin,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	return response, nil
}

// UpdateUser updates user information (excluding email)
func (s *UserService) UpdateUser(ctx context.Context, id uint, req *dto.UpdateUserRequest) (*dto.UserResponse, error) {
	ctx = context.WithValue(ctx, ctxutil.FunctionKey, "UpdateUser")
	ctx = context.WithValue(ctx, ctxutil.ModuleKey, "service")

	logger.InfoWithContext(ctx, "Updating user").
		Int("user_id", int(id)).
		Log()

	// Check if user exists
	_, err := s.repoUser.GetByID(ctx, int(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.InfoWithContext(ctx, "User not found for update").
				Int("user_id", int(id)).
				Log()
			return nil, apperrors.ErrUserNotFound
		}
		logger.ErrorWithContext(ctx, "Failed to get user for update").
			Int("user_id", int(id)).
			Err(err).
			Log()
		return nil, apperrors.WrapError(apperrors.ErrInternal, err)
	}

	// Update only non-empty fields
	updateUser := &model.User{}
	if req.FirstName != "" {
		updateUser.FirstName = strings.TrimSpace(req.FirstName)
	}
	if req.LastName != "" {
		updateUser.LastName = strings.TrimSpace(req.LastName)
	}
	if req.Phone != "" {
		updateUser.Phone = strings.TrimSpace(req.Phone)
	}

	// Save updates
	if err := s.repoUser.Update(ctx, id, updateUser); err != nil {
		logger.ErrorWithContext(ctx, "Failed to update user").
			Int("user_id", int(id)).
			Err(err).
			Log()
		return nil, apperrors.WrapError(apperrors.ErrInternal, err)
	}

	// Get updated user
	updatedUser, err := s.repoUser.GetByID(ctx, int(id))
	if err != nil {
		logger.ErrorWithContext(ctx, "Failed to get updated user").
			Int("user_id", int(id)).
			Err(err).
			Log()
		return nil, apperrors.WrapError(apperrors.ErrInternal, err)
	}

	logger.InfoWithContext(ctx, "User updated successfully").
		Int("user_id", int(id)).
		String("email", updateUser.Email).
		Log()

	response := &dto.UserResponse{
		ID:        updatedUser.ID,
		FirstName: updatedUser.FirstName,
		LastName:  updatedUser.LastName,
		Email:     updatedUser.Email,
		Phone:     updatedUser.Phone,
		LastLogin: updatedUser.LastLogin,
		CreatedAt: updatedUser.CreatedAt,
		UpdatedAt: updatedUser.UpdatedAt,
	}

	return response, nil
}

// UpdatePassword updates user password with current password verification
func (s *UserService) UpdatePassword(ctx context.Context, id uint, req *dto.UpdatePasswordRequest) error {
	ctx = context.WithValue(ctx, ctxutil.FunctionKey, "UpdatePassword")
	ctx = context.WithValue(ctx, ctxutil.ModuleKey, "service")

	logger.InfoWithContext(ctx, "Updating user password").
		Int("user_id", int(id)).
		Log()

	// Validate new password confirmation
	if req.NewPassword != req.ConfirmPassword {
		logger.WarnWithContext(ctx, "New password confirmation mismatch").
			Int("user_id", int(id)).
			Log()
		return apperrors.ErrPasswordMismatch
	}

	// Get current user
	user, err := s.repoUser.GetByID(ctx, int(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.InfoWithContext(ctx, "User not found for password update").
				Int("user_id", int(id)).
				Log()
			return apperrors.ErrUserNotFound
		}
		logger.ErrorWithContext(ctx, "Failed to get user for password update").
			Int("user_id", int(id)).
			Err(err).
			Log()
		return apperrors.WrapError(apperrors.ErrInternal, err)
	}

	// Verify current password
	if !s.checkPassword(user.Password, req.CurrentPassword) {
		logger.WarnWithContext(ctx, "Current password verification failed").
			Int("user_id", int(id)).
			Log()
		return apperrors.ErrIncorrectPassword
	}

	// Hash new password
	hashedPassword, err := s.hashPassword(req.NewPassword)
	if err != nil {
		logger.ErrorWithContext(ctx, "Failed to hash new password").
			Int("user_id", int(id)).
			Err(err).
			Log()
		return apperrors.WrapError(apperrors.ErrInternal, err)
	}

	// Update password in database
	if err := s.repoUser.UpdatePassword(ctx, id, hashedPassword); err != nil {
		logger.ErrorWithContext(ctx, "Failed to update password in database").
			Int("user_id", int(id)).
			Err(err).
			Log()
		return apperrors.WrapError(apperrors.ErrInternal, err)
	}

	logger.InfoWithContext(ctx, "User password updated successfully").
		Int("user_id", int(id)).
		String("email", user.Email).
		Log()

	return nil
}

// AuthenticateUser verifies user credentials and returns user data
func (s *UserService) AuthenticateUser(ctx context.Context, email, password string) (*dto.UserResponse, error) {
	ctx = context.WithValue(ctx, ctxutil.FunctionKey, "AuthenticateUser")
	ctx = context.WithValue(ctx, ctxutil.ModuleKey, "service")

	logger.DebugWithContext(ctx, "Authenticating user").
		String("email", email).
		Log()

	// Get user by email
	user, err := s.repoUser.GetByEmail(ctx, email)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.InfoWithContext(ctx, "Authentication failed: user not found").
				String("email", email).
				Log()
			return nil, apperrors.ErrInvalidCredentials
		}
		logger.ErrorWithContext(ctx, "Failed to get user for authentication").
			String("email", email).
			Err(err).
			Log()
		return nil, apperrors.WrapError(apperrors.ErrInternal, err)
	}

	// Verify password
	if !s.checkPassword(user.Password, password) {
		logger.WarnWithContext(ctx, "Authentication failed: incorrect password").
			String("email", email).
			Int("user_id", int(user.ID)).
			Log()
		return nil, apperrors.ErrInvalidCredentials
	}

	// Update last login timestamp
	if err := s.repoUser.UpdateLastLogin(ctx, user.ID); err != nil {
		logger.WarnWithContext(ctx, "Failed to update last login timestamp").
			Int("user_id", int(user.ID)).
			Err(err).
			Log()
		// Continue even if update fails
	}

	logger.InfoWithContext(ctx, "User authenticated successfully").
		String("email", email).
		Int("user_id", int(user.ID)).
		Log()

	response := &dto.UserResponse{
		ID:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Phone:     user.Phone,
		LastLogin: user.LastLogin,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	return response, nil
}

// LoginUser authenticates user and returns JWT + refresh token
func (s *UserService) LoginUser(ctx context.Context, email, password string) (*dto.UserLoginResponse, error) {
	ctx = context.WithValue(ctx, ctxutil.FunctionKey, "LoginUser")
	ctx = context.WithValue(ctx, ctxutil.ModuleKey, "service")

	logger.InfoWithContext(ctx, "User login attempt").
		String("email", email).
		Log()

	// Authenticate user
	user, err := s.AuthenticateUser(ctx, email, password)
	if err != nil {
		return nil, err
	}

	// Generate JWT token and refresh token
	if s.jwtService == nil {
		logger.ErrorWithContext(ctx, "JWT service not initialized").
			String("email", email).
			Log()
		return nil, apperrors.ErrServiceUnavailable
	}

	// Get current user to get token version
	fullUser, err := s.repoUser.GetByID(ctx, int(user.ID))
	if err != nil {
		logger.ErrorWithContext(ctx, "Failed to get user for token generation").
			String("email", email).
			Err(err).
			Log()
		return nil, apperrors.WrapError(apperrors.ErrInternal, err)
	}

	// Generate access token
	token, err := s.jwtService.GenerateToken(user, fullUser.TokenVersion)
	if err != nil {
		logger.ErrorWithContext(ctx, "Failed to generate JWT token").
			String("email", email).
			Err(err).
			Log()
		return nil, apperrors.WrapError(apperrors.ErrInternal, err)
	}

	// Generate refresh token
	refreshToken, err := s.jwtService.GenerateRefreshToken()
	if err != nil {
		logger.ErrorWithContext(ctx, "Failed to generate refresh token").
			String("email", email).
			Err(err).
			Log()
		return nil, apperrors.WrapError(apperrors.ErrInternal, err)
	}

	// Hash refresh token for storage
	refreshTokenHash, err := s.jwtService.HashRefreshToken(refreshToken)
	if err != nil {
		logger.ErrorWithContext(ctx, "Failed to hash refresh token").
			String("email", email).
			Err(err).
			Log()
		return nil, apperrors.WrapError(apperrors.ErrInternal, err)
	}

	// Store refresh token in database
	refreshTokenExpires := time.Now().Add(7 * 24 * time.Hour) // 7 days
	if err := s.repoUser.UpdateRefreshToken(ctx, user.ID, refreshTokenHash, &refreshTokenExpires); err != nil {
		logger.ErrorWithContext(ctx, "Failed to store refresh token").
			String("email", email).
			Err(err).
			Log()
		return nil, apperrors.WrapError(apperrors.ErrInternal, err)
	}

	response := &dto.UserLoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresIn:    15 * 60, // 15 minutes in seconds
		User:         *user,
	}

	logger.InfoWithContext(ctx, "User logged in successfully").
		String("email", email).
		Int("user_id", int(user.ID)).
		Log()

	return response, nil
}

// RefreshToken generates new access and refresh tokens
func (s *UserService) RefreshToken(ctx context.Context, refreshToken string) (*dto.RefreshTokenResponse, error) {
	ctx = context.WithValue(ctx, ctxutil.FunctionKey, "RefreshToken")
	ctx = context.WithValue(ctx, ctxutil.ModuleKey, "service")

	logger.InfoWithContext(ctx, "Token refresh attempt").
		String("token_length", fmt.Sprintf("%d", len(refreshToken))).
		Log()

	if s.jwtService == nil {
		logger.ErrorWithContext(ctx, "JWT service not initialized").
			Log()
		return nil, apperrors.ErrServiceUnavailable
	}

	// Find user by refresh token
	user, err := s.repoUser.FindByRefreshToken(ctx, refreshToken)
	if err != nil {
		logger.WarnWithContext(ctx, "Invalid refresh token").
			Err(err).
			Log()
		return nil, apperrors.ErrInvalidRefreshToken
	}

	// Check if refresh token is expired
	if user.RefreshTokenExpires != nil && user.RefreshTokenExpires.Before(time.Now()) {
		logger.WarnWithContext(ctx, "Refresh token expired").
			Int("user_id", int(user.ID)).
			Log()
		// Cleanup expired token
		s.repoUser.UpdateRefreshToken(ctx, user.ID, "", nil)
		return nil, apperrors.ErrTokenExpired
	}

	// Increment token version for security
	newTokenVersion := user.TokenVersion + 1
	if err := s.repoUser.UpdateTokenVersion(ctx, user.ID, newTokenVersion); err != nil {
		logger.ErrorWithContext(ctx, "Failed to update token version").
			Int("user_id", int(user.ID)).
			Err(err).
			Log()
		return nil, apperrors.WrapError(apperrors.ErrInternal, err)
	}

	// Create user response
	userResponse := &dto.UserResponse{
		ID:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Phone:     user.Phone,
		LastLogin: user.LastLogin,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	// Generate new access token
	newAccessToken, err := s.jwtService.GenerateToken(userResponse, newTokenVersion)
	if err != nil {
		logger.ErrorWithContext(ctx, "Failed to generate new access token").
			Int("user_id", int(user.ID)).
			Err(err).
			Log()
		return nil, apperrors.WrapError(apperrors.ErrInternal, err)
	}

	// Generate new refresh token (rotation)
	newRefreshToken, err := s.jwtService.GenerateRefreshToken()
	if err != nil {
		logger.ErrorWithContext(ctx, "Failed to generate new refresh token").
			Int("user_id", int(user.ID)).
			Err(err).
			Log()
		return nil, apperrors.WrapError(apperrors.ErrInternal, err)
	}

	// Hash new refresh token
	newRefreshTokenHash, err := s.jwtService.HashRefreshToken(newRefreshToken)
	if err != nil {
		logger.ErrorWithContext(ctx, "Failed to hash new refresh token").
			Int("user_id", int(user.ID)).
			Err(err).
			Log()
		return nil, apperrors.WrapError(apperrors.ErrInternal, err)
	}

	// Store new refresh token (replaces old one)
	newRefreshTokenExpires := time.Now().Add(7 * 24 * time.Hour)
	if err := s.repoUser.UpdateRefreshToken(ctx, user.ID, newRefreshTokenHash, &newRefreshTokenExpires); err != nil {
		logger.ErrorWithContext(ctx, "Failed to store new refresh token").
			Int("user_id", int(user.ID)).
			Err(err).
			Log()
		return nil, apperrors.WrapError(apperrors.ErrInternal, err)
	}

	response := &dto.RefreshTokenResponse{
		Token:        newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    15 * 60, // 15 minutes
		User:         *userResponse,
	}

	logger.InfoWithContext(ctx, "Token refreshed successfully").
		Int("user_id", int(user.ID)).
		Int("new_token_version", newTokenVersion).
		Log()

	return response, nil
}

// LogoutUser invalidates refresh token
func (s *UserService) LogoutUser(ctx context.Context, userID uint) error {
	ctx = context.WithValue(ctx, ctxutil.FunctionKey, "LogoutUser")
	ctx = context.WithValue(ctx, ctxutil.ModuleKey, "service")

	logger.InfoWithContext(ctx, "User logout attempt").
		Int("user_id", int(userID)).
		Log()

	// Get current user to retrieve their token version
	user, err := s.repoUser.GetByID(ctx, int(userID))
	if err != nil {
		logger.ErrorWithContext(ctx, "Failed to get user for logout").
			Int("user_id", int(userID)).
			Err(err).
			Log()
		return apperrors.WrapError(apperrors.ErrInternal, err)
	}

	// Increment token version to invalidate all existing tokens
	newTokenVersion := user.TokenVersion + 1
	if err := s.repoUser.UpdateTokenVersion(ctx, userID, newTokenVersion); err != nil {
		logger.ErrorWithContext(ctx, "Failed to update token version on logout").
			Int("user_id", int(userID)).
			Err(err).
			Log()
		return apperrors.WrapError(apperrors.ErrInternal, err)
	}

	// Clear refresh token
	if err := s.repoUser.UpdateRefreshToken(ctx, userID, "", nil); err != nil {
		logger.WarnWithContext(ctx, "Failed to clear refresh token on logout").
			Int("user_id", int(userID)).
			Err(err).
			Log()
		// Continue even if refresh token cleanup fails
	}

	logger.InfoWithContext(ctx, "User logged out successfully").
		Int("user_id", int(userID)).
		Log()

	return nil
}

// DeleteUser performs hard delete on user with security validations
func (s *UserService) DeleteUser(ctx context.Context, id uint, requestingUserID uint) error {
	ctx = context.WithValue(ctx, ctxutil.FunctionKey, "DeleteUser")
	ctx = context.WithValue(ctx, ctxutil.ModuleKey, "service")

	logger.DebugWithContext(ctx, "Attempting to delete user").
		Int("target_user_id", int(id)).
		Int("requesting_user_id", int(requestingUserID)).
		Log()

	// Prevent users from deleting themselves
	if id == requestingUserID {
		logger.WarnWithContext(ctx, "User attempted to delete themselves").
			Int("user_id", int(id)).
			Log()
		return apperrors.ErrSelfDeletion
	}

	// Check if user exists before deletion
	user, err := s.repoUser.GetByID(ctx, int(id))
	if err != nil {
		logger.ErrorWithContext(ctx, "Failed to get user for deletion").
			Int("user_id", int(id)).
			Err(err).
			Log()
		return apperrors.WrapError(apperrors.ErrInternal, err)
	}

	if user == nil {
		logger.WarnWithContext(ctx, "User not found for deletion").
			Int("user_id", int(id)).
			Log()
		return apperrors.ErrUserNotFound
	}

	// Perform hard delete (permanent deletion)
	if err := s.repoUser.Delete(ctx, id); err != nil {
		logger.ErrorWithContext(ctx, "Failed to delete user").
			Int("user_id", int(id)).
			Err(err).
			Log()
		return apperrors.WrapError(apperrors.ErrInternal, err)
	}

	logger.InfoWithContext(ctx, "User deleted successfully").
		Int("target_user_id", int(id)).
		Int("requesting_user_id", int(requestingUserID)).
		String("deleted_user_email", user.Email).
		Log()

	return nil
}
