package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/surdiana/gateway/internal/model"
	ctxutil "github.com/surdiana/gateway/pkg/context"
	"github.com/surdiana/gateway/pkg/logger"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByID(ctx context.Context, id int) (*model.User, error) {
	// Add function info to context
	ctx = context.WithValue(ctx, ctxutil.FunctionKey, "GetByID")
	ctx = context.WithValue(ctx, ctxutil.ModuleKey, "repository")

	logger.DebugWithContext(ctx, "Getting user by ID").
		Int("user_id", id).
		Log()

	// Check if context is cancelled
	if err := ctx.Err(); err != nil {
		logger.WarnWithContext(ctx, "Context cancelled before query").
			Err(err).
			Log()
		return nil, err
	}

	start := time.Now()
	var user model.User

	// Use context in database query
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&user)
	duration := time.Since(start)

	if result.Error != nil {
		logger.ErrorWithContext(ctx, "Failed to get user by ID").
			Int("user_id", id).
			Duration(duration).
			Err(result.Error).
			Log()
	} else {
		logger.DebugWithContext(ctx, "User retrieved successfully").
			Int("user_id", id).
			String("email", user.Email).
			Duration(duration).
			Log()
	}

	return &user, result.Error
}

func (r *UserRepository) GetAll(ctx context.Context, limit, offset int, search string) ([]model.User, int64, int64, int64, error) {
	// Add function info to context
	ctx = context.WithValue(ctx, ctxutil.FunctionKey, "GetAll")
	ctx = context.WithValue(ctx, ctxutil.ModuleKey, "repository")

	logger.DebugWithContext(ctx, "Getting all users").
		Int("limit", limit).
		Int("offset", offset).
		String("search", search).
		Log()

	// Check if context is cancelled
	if err := ctx.Err(); err != nil {
		logger.WarnWithContext(ctx, "Context cancelled before query").
			Err(err).
			Log()
		return nil, 0, 0, 0, err
	}

	start := time.Now()
	var pages []model.User
	var total, verifiedCount, unverifiedCount int64

	query := r.db.WithContext(ctx).Model(&model.User{})

	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where(
			"first_name ILIKE ? OR last_name ILIKE ? OR email ILIKE ? OR phone ILIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern,
		)
		logger.DebugWithContext(ctx, "Applied search filter").
			String("search", search).
			String("search_pattern", searchPattern).
			Log()
	}

	if err := query.Count(&total).Error; err != nil {
		logger.ErrorWithContext(ctx, "Failed to count total users").
			Err(err).
			Log()
		return nil, 0, 0, 0, err
	}

	// Set verified counts to 0 since is_verified column doesn't exist
	verifiedCount = int64(0)
	unverifiedCount = int64(0)

	if err := query.Limit(limit).Offset(offset).Find(&pages).Error; err != nil {
		logger.ErrorWithContext(ctx, "Failed to fetch users").
			Int("limit", limit).
			Int("offset", offset).
			String("search", search).
			Duration(time.Since(start)).
			Err(err).
			Log()
		return nil, 0, 0, 0, err
	}

	queryDuration := time.Since(start)
	logger.InfoWithContext(ctx, "Users retrieved successfully").
		Int("limit", limit).
		Int("offset", offset).
		String("search", search).
		Int64("total", total).
		Int64("verified_count", verifiedCount).
		Int64("unverified_count", unverifiedCount).
		Int("returned_count", len(pages)).
		Duration(queryDuration).
		Log()

	return pages, total, verifiedCount, unverifiedCount, nil
}

// GetByEmail finds user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	ctx = context.WithValue(ctx, ctxutil.FunctionKey, "GetByEmail")
	ctx = context.WithValue(ctx, ctxutil.ModuleKey, "repository")

	logger.DebugWithContext(ctx, "Getting user by email").
		String("email", email).
		Log()

	start := time.Now()
	var user model.User

	result := r.db.WithContext(ctx).Where("email = ?", email).First(&user)
	duration := time.Since(start)

	if result.Error != nil {
		logger.ErrorWithContext(ctx, "Failed to get user by email").
			String("email", email).
			Duration(duration).
			Err(result.Error).
			Log()
		return nil, result.Error
	}

	logger.DebugWithContext(ctx, "User retrieved successfully by email").
		String("email", email).
		Int("user_id", int(user.ID)).
		Duration(duration).
		Log()

	return &user, nil
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	ctx = context.WithValue(ctx, ctxutil.FunctionKey, "Create")
	ctx = context.WithValue(ctx, ctxutil.ModuleKey, "repository")

	logger.DebugWithContext(ctx, "Creating new user").
		String("email", user.Email).
		String("first_name", user.FirstName).
		String("last_name", user.LastName).
		Log()

	start := time.Now()
	result := r.db.WithContext(ctx).Create(user)
	duration := time.Since(start)

	if result.Error != nil {
		logger.ErrorWithContext(ctx, "Failed to create user").
			String("email", user.Email).
			Duration(duration).
			Err(result.Error).
			Log()
		return result.Error
	}

	logger.InfoWithContext(ctx, "User created successfully").
		String("email", user.Email).
		Int("user_id", int(user.ID)).
		Duration(duration).
		Log()

	return nil
}

// Update updates user information (excluding email)
func (r *UserRepository) Update(ctx context.Context, id uint, user *model.User) error {
	ctx = context.WithValue(ctx, ctxutil.FunctionKey, "Update")
	ctx = context.WithValue(ctx, ctxutil.ModuleKey, "repository")

	logger.DebugWithContext(ctx, "Updating user").
		Int("user_id", int(id)).
		String("first_name", user.FirstName).
		String("last_name", user.LastName).
		String("phone", user.Phone).
		Log()

	start := time.Now()
	result := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"first_name": user.FirstName,
		"last_name":  user.LastName,
		"phone":      user.Phone,
	})
	duration := time.Since(start)

	if result.Error != nil {
		logger.ErrorWithContext(ctx, "Failed to update user").
			Int("user_id", int(id)).
			Duration(duration).
			Err(result.Error).
			Log()
		return result.Error
	}

	if result.RowsAffected == 0 {
		logger.WarnWithContext(ctx, "No user found to update").
			Int("user_id", int(id)).
			Log()
		return gorm.ErrRecordNotFound
	}

	logger.InfoWithContext(ctx, "User updated successfully").
		Int("user_id", int(id)).
		Int64("rows_affected", result.RowsAffected).
		Duration(duration).
		Log()

	return nil
}

// UpdatePassword updates user password
func (r *UserRepository) UpdatePassword(ctx context.Context, id uint, hashedPassword string) error {
	ctx = context.WithValue(ctx, ctxutil.FunctionKey, "UpdatePassword")
	ctx = context.WithValue(ctx, ctxutil.ModuleKey, "repository")

	logger.DebugWithContext(ctx, "Updating user password").
		Int("user_id", int(id)).
		Log()

	start := time.Now()
	result := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("password", hashedPassword)
	duration := time.Since(start)

	if result.Error != nil {
		logger.ErrorWithContext(ctx, "Failed to update user password").
			Int("user_id", int(id)).
			Duration(duration).
			Err(result.Error).
			Log()
		return result.Error
	}

	if result.RowsAffected == 0 {
		logger.WarnWithContext(ctx, "No user found to update password").
			Int("user_id", int(id)).
			Log()
		return gorm.ErrRecordNotFound
	}

	logger.InfoWithContext(ctx, "User password updated successfully").
		Int("user_id", int(id)).
		Duration(duration).
		Log()

	return nil
}

// UpdateLastLogin updates the last login timestamp
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id uint) error {
	ctx = context.WithValue(ctx, ctxutil.FunctionKey, "UpdateLastLogin")
	ctx = context.WithValue(ctx, ctxutil.ModuleKey, "repository")

	start := time.Now()
	result := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("last_login", time.Now())
	duration := time.Since(start)

	if result.Error != nil {
		logger.ErrorWithContext(ctx, "Failed to update last login").
			Int("user_id", int(id)).
			Duration(duration).
			Err(result.Error).
			Log()
		return result.Error
	}

	logger.DebugWithContext(ctx, "Last login updated successfully").
		Int("user_id", int(id)).
		Duration(duration).
		Log()

	return nil
}

// UpdateRefreshToken updates user's refresh token and expiry
func (r *UserRepository) UpdateRefreshToken(ctx context.Context, id uint, refreshTokenHash string, expiresAt *time.Time) error {
	ctx = context.WithValue(ctx, ctxutil.FunctionKey, "UpdateRefreshToken")
	ctx = context.WithValue(ctx, ctxutil.ModuleKey, "repository")

	logger.DebugWithContext(ctx, "Updating refresh token").
		Int("user_id", int(id)).
		Bool("has_token", refreshTokenHash != "").
		Log()

	start := time.Now()
	result := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"refresh_token_hash":    refreshTokenHash,
		"refresh_token_expires_at": expiresAt,
	})
	duration := time.Since(start)

	if result.Error != nil {
		logger.ErrorWithContext(ctx, "Failed to update refresh token").
			Int("user_id", int(id)).
			Duration(duration).
			Err(result.Error).
			Log()
		return result.Error
	}

	if result.RowsAffected == 0 {
		logger.WarnWithContext(ctx, "No user found to update refresh token").
			Int("user_id", int(id)).
			Log()
		return gorm.ErrRecordNotFound
	}

	logger.DebugWithContext(ctx, "Refresh token updated successfully").
		Int("user_id", int(id)).
		Duration(duration).
		Log()

	return nil
}

// FindByRefreshToken finds user by refresh token (optimized with bcrypt comparison)
func (r *UserRepository) FindByRefreshToken(ctx context.Context, refreshToken string) (*model.User, error) {
	ctx = context.WithValue(ctx, ctxutil.FunctionKey, "FindByRefreshToken")
	ctx = context.WithValue(ctx, ctxutil.ModuleKey, "repository")

	logger.DebugWithContext(ctx, "Finding user by refresh token").
		String("token_length", fmt.Sprintf("%d", len(refreshToken))).
		Log()

	start := time.Now()
	var users []model.User

	// Get all users with refresh tokens (optimized query)
	result := r.db.WithContext(ctx).
		Where("refresh_token_hash IS NOT NULL AND refresh_token_hash != ''").
		Find(&users)
	duration := time.Since(start)

	if result.Error != nil {
		logger.ErrorWithContext(ctx, "Failed to query users with refresh tokens").
			Duration(duration).
			Err(result.Error).
			Log()
		return nil, result.Error
	}

	// Check each user's refresh token hash using bcrypt directly
	for _, user := range users {
		err := bcrypt.CompareHashAndPassword([]byte(user.RefreshTokenHash), []byte(refreshToken))
		if err == nil {
			logger.DebugWithContext(ctx, "Refresh token verified successfully").
				Int("user_id", int(user.ID)).
				Duration(duration).
				Log()
			return &user, nil
		}
	}

	logger.DebugWithContext(ctx, "No valid refresh token found").
		Duration(duration).
		Log()

	return nil, gorm.ErrRecordNotFound
}

// UpdateTokenVersion increments user's token version
func (r *UserRepository) UpdateTokenVersion(ctx context.Context, id uint, newVersion int) error {
	ctx = context.WithValue(ctx, ctxutil.FunctionKey, "UpdateTokenVersion")
	ctx = context.WithValue(ctx, ctxutil.ModuleKey, "repository")

	logger.DebugWithContext(ctx, "Updating token version").
		Int("user_id", int(id)).
		Int("new_version", newVersion).
		Log()

	start := time.Now()
	result := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("token_version", newVersion)
	duration := time.Since(start)

	if result.Error != nil {
		logger.ErrorWithContext(ctx, "Failed to update token version").
			Int("user_id", int(id)).
			Duration(duration).
			Err(result.Error).
			Log()
		return result.Error
	}

	if result.RowsAffected == 0 {
		logger.WarnWithContext(ctx, "No user found to update token version").
			Int("user_id", int(id)).
			Log()
		return gorm.ErrRecordNotFound
	}

	logger.DebugWithContext(ctx, "Token version updated successfully").
		Int("user_id", int(id)).
		Duration(duration).
		Log()

	return nil
}

// CleanupExpiredRefreshTokens removes expired refresh tokens (batch operation)
func (r *UserRepository) CleanupExpiredRefreshTokens(ctx context.Context) (int64, error) {
	ctx = context.WithValue(ctx, ctxutil.FunctionKey, "CleanupExpiredRefreshTokens")
	ctx = context.WithValue(ctx, ctxutil.ModuleKey, "repository")

	logger.DebugWithContext(ctx, "Cleaning up expired refresh tokens").
		Log()

	start := time.Now()
	result := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("refresh_token_expires_at IS NOT NULL AND refresh_token_expires_at < ?", time.Now()).
		Updates(map[string]interface{}{
			"refresh_token_hash":     nil,
			"refresh_token_expires_at": nil,
		})
	duration := time.Since(start)

	if result.Error != nil {
		logger.ErrorWithContext(ctx, "Failed to cleanup expired refresh tokens").
			Duration(duration).
			Err(result.Error).
			Log()
		return 0, result.Error
	}

	logger.InfoWithContext(ctx, "Expired refresh tokens cleaned up successfully").
		Int64("cleaned_count", result.RowsAffected).
		Duration(duration).
		Log()

	return result.RowsAffected, nil
}

// Delete performs hard delete on user (permanent deletion)
func (r *UserRepository) Delete(ctx context.Context, id uint) error {
	ctx = context.WithValue(ctx, ctxutil.FunctionKey, "Delete")
	ctx = context.WithValue(ctx, ctxutil.ModuleKey, "repository")

	logger.DebugWithContext(ctx, "Hard deleting user").
		Int("user_id", int(id)).
		Log()

	start := time.Now()
	result := r.db.WithContext(ctx).Delete(&model.User{}, id)
	duration := time.Since(start)

	if result.Error != nil {
		logger.ErrorWithContext(ctx, "Failed to delete user").
			Int("user_id", int(id)).
			Duration(duration).
			Err(result.Error).
			Log()
		return result.Error
	}

	if result.RowsAffected == 0 {
		logger.WarnWithContext(ctx, "No user found to delete").
			Int("user_id", int(id)).
			Log()
		return gorm.ErrRecordNotFound
	}

	logger.InfoWithContext(ctx, "User deleted successfully").
		Int("user_id", int(id)).
		Duration(duration).
		Log()

	return nil
}
