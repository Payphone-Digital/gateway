package repository

import (
	"context"
	"strings"
	"time"

	"github.com/surdiana/gateway/internal/model"
	"github.com/surdiana/gateway/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type APIConfigRepository struct {
	db *gorm.DB
}

func NewAPIConfigRepository(db *gorm.DB) *APIConfigRepository {
	return &APIConfigRepository{db: db}
}

func (r *APIConfigRepository) CreateURLConfig(ctx context.Context, req *model.URLConfig) error {
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Repository: Context cancelled before creating URL config",
			zap.String("nama", req.Nama),
			zap.Error(err),
		)
		return err
	}

	logger.GetLogger().Debug("Repository: Creating URL config",
		zap.String("nama", req.Nama),
		zap.String("protocol", req.Protocol),
		zap.String("url", req.URL),
	)

	start := time.Now()
	err := r.db.WithContext(ctx).Create(req).Error
	duration := time.Since(start)

	if err != nil {
		logger.GetLogger().Error("Repository: Failed to create URL config",
			zap.String("nama", req.Nama),
			zap.Duration("query_duration", duration),
			zap.Error(err),
		)
	} else {
		logger.GetLogger().Info("Repository: URL config created successfully",
			zap.String("nama", req.Nama),
			zap.Uint("url_config_id", req.ID),
			zap.Duration("query_duration", duration),
		)
	}

	return err
}

func (r *APIConfigRepository) GetByIDURLConfig(id uint) (*model.URLConfig, error) {
	logger.GetLogger().Debug("Repository: Getting URL config by ID",
		zap.Uint("url_config_id", id),
	)

	start := time.Now()
	var res model.URLConfig
	err := r.db.First(&res, id).Error
	duration := time.Since(start)

	if err != nil {
		logger.GetLogger().Error("Repository: Failed to get URL config by ID",
			zap.Uint("url_config_id", id),
			zap.Duration("query_duration", duration),
			zap.Error(err),
		)
		return nil, err
	}

	logger.GetLogger().Debug("Repository: URL config retrieved successfully",
		zap.Uint("url_config_id", id),
		zap.String("nama", res.Nama),
		zap.String("protocol", res.Protocol),
		zap.Duration("query_duration", duration),
	)

	return &res, nil
}

func (r *APIConfigRepository) GetAllURLConfig(limit, offset int, search string) ([]model.URLConfig, int64, error) {
	logger.GetLogger().Debug("Repository: Getting all URL configs",
		zap.Int("limit", limit),
		zap.Int("offset", offset),
		zap.String("search", search),
	)

	start := time.Now()
	var urlConfigs []model.URLConfig
	var total int64

	query := r.db.Model(&model.URLConfig{})

	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where(
			"nama ILIKE ? OR url ILIKE ? OR deskripsi ILIKE ?",
			searchPattern, searchPattern, searchPattern,
		)
	}

	if err := query.Count(&total).Error; err != nil {
		logger.GetLogger().Error("Repository: Failed to count total URL configs",
			zap.Error(err),
		)
		return nil, 0, err
	}

	if err := query.Limit(limit).Offset(offset).Find(&urlConfigs).Error; err != nil {
		logger.GetLogger().Error("Repository: Failed to fetch URL configs",
			zap.Int("limit", limit),
			zap.Int("offset", offset),
			zap.String("search", search),
			zap.Duration("query_time", time.Since(start)),
			zap.Error(err),
		)
		return nil, 0, err
	}

	queryDuration := time.Since(start)
	logger.GetLogger().Info("Repository: URL configs retrieved successfully",
		zap.Int("limit", limit),
		zap.Int("offset", offset),
		zap.String("search", search),
		zap.Int64("total", total),
		zap.Int("returned_count", len(urlConfigs)),
		zap.Duration("query_duration", queryDuration),
	)

	return urlConfigs, total, nil
}

func (r *APIConfigRepository) UpdateURLConfig(ctx context.Context, req *model.URLConfig) error {
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Repository: Context cancelled before updating URL config",
			zap.Uint("url_config_id", req.ID),
			zap.String("nama", req.Nama),
			zap.Error(err),
		)
		return err
	}

	logger.GetLogger().Debug("Repository: Updating URL config",
		zap.Uint("url_config_id", req.ID),
		zap.String("nama", req.Nama),
		zap.String("protocol", req.Protocol),
	)

	start := time.Now()
	err := r.db.WithContext(ctx).Save(req).Error
	duration := time.Since(start)

	if err != nil {
		logger.GetLogger().Error("Repository: Failed to update URL config",
			zap.Uint("url_config_id", req.ID),
			zap.String("nama", req.Nama),
			zap.Duration("query_duration", duration),
			zap.Error(err),
		)
	} else {
		logger.GetLogger().Info("Repository: URL config updated successfully",
			zap.Uint("url_config_id", req.ID),
			zap.String("nama", req.Nama),
			zap.Duration("query_duration", duration),
		)
	}

	return err
}

func (r *APIConfigRepository) DeleteURLConfig(ctx context.Context, id uint) error {
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Repository: Context cancelled before deleting URL config",
			zap.Uint("url_config_id", id),
			zap.Error(err),
		)
		return err
	}

	logger.GetLogger().Debug("Repository: Deleting URL config",
		zap.Uint("url_config_id", id),
	)

	start := time.Now()
	err := r.db.WithContext(ctx).Delete(&model.URLConfig{}, id).Error
	duration := time.Since(start)

	if err != nil {
		logger.GetLogger().Error("Repository: Failed to delete URL config",
			zap.Uint("url_config_id", id),
			zap.Duration("query_duration", duration),
			zap.Error(err),
		)
	} else {
		logger.GetLogger().Info("Repository: URL config deleted successfully",
			zap.Uint("url_config_id", id),
			zap.Duration("query_duration", duration),
		)
	}

	return err
}

func (r *APIConfigRepository) CreateConfig(ctx context.Context, req *model.APIConfig) error {
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Repository: Context cancelled before creating API config",
			zap.String("slug", req.Slug),
			zap.Error(err),
		)
		return err
	}

	logger.GetLogger().Debug("Repository: Creating API config",
		zap.String("slug", req.Slug),
		zap.String("method", req.Method),
		zap.Uint("url_config_id", req.URLConfigID),
		zap.String("uri", req.URI),
		zap.Int("max_retries", req.MaxRetries),
		zap.Int("timeout", req.Timeout),
	)

	start := time.Now()
	// Use WithContext to pass context to database operation
	err := r.db.WithContext(ctx).Create(req).Error
	duration := time.Since(start)

	if err != nil {
		logger.GetLogger().Error("Repository: Failed to create API config",
			zap.String("slug", req.Slug),
			zap.Duration("query_duration", duration),
			zap.Error(err),
		)
	} else {
		logger.GetLogger().Info("Repository: API config created successfully",
			zap.String("slug", req.Slug),
			zap.Uint("config_id", req.ID),
			zap.Duration("query_duration", duration),
		)
	}

	return err
}

// DISABLED: CreateGroup - Group-related functions are disabled
func (r *APIConfigRepository) CreateGroup(ctx context.Context, req *model.APIGroup) error {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Repository: Context cancelled before creating API group",
			zap.String("slug", req.Slug),
			zap.Error(err),
		)
		return err
	}

	logger.GetLogger().Debug("Repository: Creating API group",
		zap.String("slug", req.Slug),
		zap.String("name", req.Name),
	)

	start := time.Now()
	// Use WithContext to pass context to database operation
	err := r.db.WithContext(ctx).Create(req).Error
	duration := time.Since(start)

	if err != nil {
		logger.GetLogger().Error("Repository: Failed to create API group",
			zap.String("slug", req.Slug),
			zap.Duration("query_duration", duration),
			zap.Error(err),
		)
	} else {
		logger.GetLogger().Info("Repository: API group created successfully",
			zap.String("slug", req.Slug),
			zap.Uint("group_id", req.ID),
			zap.Duration("query_duration", duration),
		)
	}

	return err
}

// DISABLED: CreateGroupStep - Group-related functions are disabled
func (r *APIConfigRepository) CreateGroupStep(ctx context.Context, req *model.APIGroupStep) error {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Repository: Context cancelled before creating API group step",
			zap.String("alias", req.Alias),
			zap.Uint("group_id", req.GroupID),
			zap.Error(err),
		)
		return err
	}

	logger.GetLogger().Debug("Repository: Creating API group step",
		zap.String("alias", req.Alias),
		zap.Uint("group_id", req.GroupID),
		zap.Uint("api_config_id", req.APIConfigID),
		zap.Int("order_index", req.OrderIndex),
	)

	start := time.Now()
	// Use WithContext to pass context to database operation
	err := r.db.WithContext(ctx).Create(req).Error
	duration := time.Since(start)

	if err != nil {
		logger.GetLogger().Error("Repository: Failed to create API group step",
			zap.String("alias", req.Alias),
			zap.Uint("group_id", req.GroupID),
			zap.Duration("query_duration", duration),
			zap.Error(err),
		)
	} else {
		logger.GetLogger().Info("Repository: API group step created successfully",
			zap.String("alias", req.Alias),
			zap.Uint("step_id", req.ID),
			zap.Uint("group_id", req.GroupID),
			zap.Duration("query_duration", duration),
		)
	}

	return err
}

// DISABLED: CreateGroupCron - Group-related functions are disabled
func (r *APIConfigRepository) CreateGroupCron(ctx context.Context, req *model.APIGroupCron) error {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Repository: Context cancelled before creating API group cron",
			zap.String("slug", req.Slug),
			zap.Error(err),
		)
		return err
	}

	logger.GetLogger().Debug("Repository: Creating API group cron",
		zap.String("slug", req.Slug),
		zap.String("schedule", req.Schedule),
		zap.Bool("enabled", req.Enabled),
	)

	start := time.Now()
	// Use WithContext to pass context to database operation
	err := r.db.WithContext(ctx).Create(req).Error
	duration := time.Since(start)

	if err != nil {
		logger.GetLogger().Error("Repository: Failed to create API group cron",
			zap.String("slug", req.Slug),
			zap.Duration("query_duration", duration),
			zap.Error(err),
		)
	} else {
		logger.GetLogger().Info("Repository: API group cron created successfully",
			zap.String("slug", req.Slug),
			zap.Uint("cron_id", req.ID),
			zap.Duration("query_duration", duration),
		)
	}

	return err
}

// End Create

// Start Update
func (r *APIConfigRepository) UpdateConfig(ctx context.Context, req *model.APIConfig) error {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Repository: Context cancelled before updating API config",
			zap.Uint("config_id", req.ID),
			zap.String("slug", req.Slug),
			zap.Error(err),
		)
		return err
	}

	logger.GetLogger().Debug("Repository: Updating API config",
		zap.Uint("config_id", req.ID),
		zap.String("slug", req.Slug),
		zap.String("method", req.Method),
		zap.Uint("url_config_id", req.URLConfigID),
		zap.String("uri", req.URI),
		zap.Int("max_retries", req.MaxRetries),
		zap.Int("timeout", req.Timeout),
	)

	start := time.Now()
	err := r.db.WithContext(ctx).Save(req).Error
	duration := time.Since(start)

	if err != nil {
		logger.GetLogger().Error("Repository: Failed to update API config",
			zap.Uint("config_id", req.ID),
			zap.String("slug", req.Slug),
			zap.Duration("query_duration", duration),
			zap.Error(err),
		)
	} else {
		logger.GetLogger().Info("Repository: API config updated successfully",
			zap.Uint("config_id", req.ID),
			zap.String("slug", req.Slug),
			zap.Duration("query_duration", duration),
		)
	}

	return err
}

// DISABLED: UpdateGroup - Group-related functions are disabled
func (r *APIConfigRepository) UpdateGroup(req *model.APIGroup) error {
	logger.GetLogger().Debug("Repository: Updating API group",
		zap.Uint("group_id", req.ID),
		zap.String("slug", req.Slug),
		zap.String("name", req.Name),
	)

	start := time.Now()
	err := r.db.Save(req).Error
	duration := time.Since(start)

	if err != nil {
		logger.GetLogger().Error("Repository: Failed to update API group",
			zap.Uint("group_id", req.ID),
			zap.String("slug", req.Slug),
			zap.Duration("query_duration", duration),
			zap.Error(err),
		)
	} else {
		logger.GetLogger().Info("Repository: API group updated successfully",
			zap.Uint("group_id", req.ID),
			zap.String("slug", req.Slug),
			zap.Duration("query_duration", duration),
		)
	}

	return err
}

// DISABLED: UpdateGroupStep - Group-related functions are disabled
func (r *APIConfigRepository) UpdateGroupStep(apiConfig *model.APIGroupStep) error {
	logger.GetLogger().Debug("Repository: Updating API group step",
		zap.Uint("step_id", apiConfig.ID),
		zap.String("alias", apiConfig.Alias),
		zap.Uint("group_id", apiConfig.GroupID),
		zap.Uint("api_config_id", apiConfig.APIConfigID),
		zap.Int("order_index", apiConfig.OrderIndex),
	)

	start := time.Now()
	err := r.db.Save(apiConfig).Error
	duration := time.Since(start)

	if err != nil {
		logger.GetLogger().Error("Repository: Failed to update API group step",
			zap.Uint("step_id", apiConfig.ID),
			zap.String("alias", apiConfig.Alias),
			zap.Duration("query_duration", duration),
			zap.Error(err),
		)
	} else {
		logger.GetLogger().Info("Repository: API group step updated successfully",
			zap.Uint("step_id", apiConfig.ID),
			zap.String("alias", apiConfig.Alias),
			zap.Duration("query_duration", duration),
		)
	}

	return err
}

// DISABLED: UpdateGroupCron - Group-related functions are disabled
func (r *APIConfigRepository) UpdateGroupCron(apiGroupCron *model.APIGroupCron) error {
	logger.GetLogger().Debug("Repository: Updating API group cron",
		zap.Uint("cron_id", apiGroupCron.ID),
		zap.String("slug", apiGroupCron.Slug),
		zap.String("schedule", apiGroupCron.Schedule),
		zap.Bool("enabled", apiGroupCron.Enabled),
	)

	start := time.Now()
	err := r.db.Save(apiGroupCron).Error
	duration := time.Since(start)

	if err != nil {
		logger.GetLogger().Error("Repository: Failed to update API group cron",
			zap.Uint("cron_id", apiGroupCron.ID),
			zap.String("slug", apiGroupCron.Slug),
			zap.Duration("query_duration", duration),
			zap.Error(err),
		)
	} else {
		logger.GetLogger().Info("Repository: API group cron updated successfully",
			zap.Uint("cron_id", apiGroupCron.ID),
			zap.String("slug", apiGroupCron.Slug),
			zap.Duration("query_duration", duration),
		)
	}

	return err
}

// End Update

// Start Delete
func (r *APIConfigRepository) DeleteConfig(ctx context.Context, id uint) error {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Repository: Context cancelled before deleting API config",
			zap.Uint("config_id", id),
			zap.Error(err),
		)
		return err
	}

	logger.GetLogger().Debug("Repository: Deleting API config",
		zap.Uint("config_id", id),
	)

	start := time.Now()
	// Use WithContext to pass context to database operation
	err := r.db.WithContext(ctx).Delete(&model.APIConfig{}, id).Error
	duration := time.Since(start)

	if err != nil {
		logger.GetLogger().Error("Repository: Failed to delete API config",
			zap.Uint("config_id", id),
			zap.Duration("query_duration", duration),
			zap.Error(err),
		)
	} else {
		logger.GetLogger().Info("Repository: API config deleted successfully",
			zap.Uint("config_id", id),
			zap.Duration("query_duration", duration),
		)
	}

	return err
}

// DISABLED: DeleteGroup - Group-related functions are disabled
func (r *APIConfigRepository) DeleteGroup(ctx context.Context, id uint) error {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Repository: Context cancelled before deleting API group",
			zap.Uint("group_id", id),
			zap.Error(err),
		)
		return err
	}

	logger.GetLogger().Debug("Repository: Deleting API group",
		zap.Uint("group_id", id),
	)

	start := time.Now()
	// Use WithContext to pass context to database operation
	err := r.db.WithContext(ctx).Delete(&model.APIGroup{}, id).Error
	duration := time.Since(start)

	if err != nil {
		logger.GetLogger().Error("Repository: Failed to delete API group",
			zap.Uint("group_id", id),
			zap.Duration("query_duration", duration),
			zap.Error(err),
		)
	} else {
		logger.GetLogger().Info("Repository: API group deleted successfully",
			zap.Uint("group_id", id),
			zap.Duration("query_duration", duration),
		)
	}

	return err
}

// DISABLED: DeleteGroupStep - Group-related functions are disabled
func (r *APIConfigRepository) DeleteGroupStep(ctx context.Context, id uint) error {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Repository: Context cancelled before deleting API group step",
			zap.Uint("step_id", id),
			zap.Error(err),
		)
		return err
	}

	logger.GetLogger().Debug("Repository: Deleting API group step",
		zap.Uint("step_id", id),
	)

	start := time.Now()
	// Use WithContext to pass context to database operation
	err := r.db.WithContext(ctx).Delete(&model.APIGroupStep{}, id).Error
	duration := time.Since(start)

	if err != nil {
		logger.GetLogger().Error("Repository: Failed to delete API group step",
			zap.Uint("step_id", id),
			zap.Duration("query_duration", duration),
			zap.Error(err),
		)
	} else {
		logger.GetLogger().Info("Repository: API group step deleted successfully",
			zap.Uint("step_id", id),
			zap.Duration("query_duration", duration),
		)
	}

	return err
}

// DISABLED: DeleteGroupCron - Group-related functions are disabled
func (r *APIConfigRepository) DeleteGroupCron(ctx context.Context, id uint) error {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Repository: Context cancelled before deleting API group cron",
			zap.Uint("cron_id", id),
			zap.Error(err),
		)
		return err
	}

	logger.GetLogger().Debug("Repository: Deleting API group cron",
		zap.Uint("cron_id", id),
	)

	start := time.Now()
	// Use WithContext to pass context to database operation
	err := r.db.WithContext(ctx).Delete(&model.APIGroupCron{}, id).Error
	duration := time.Since(start)

	if err != nil {
		logger.GetLogger().Error("Repository: Failed to delete API group cron",
			zap.Uint("cron_id", id),
			zap.Duration("query_duration", duration),
			zap.Error(err),
		)
	} else {
		logger.GetLogger().Info("Repository: API group cron deleted successfully",
			zap.Uint("cron_id", id),
			zap.Duration("query_duration", duration),
		)
	}

	return err
}

// End Delete

// Start Get By ID

func (r *APIConfigRepository) GetByIDConfig(id uint) (*model.APIConfig, error) {
	logger.GetLogger().Debug("Repository: Getting API config by ID",
		zap.Uint("config_id", id),
	)

	start := time.Now()
	var res model.APIConfig
	err := r.db.Preload("URLConfig").First(&res, id).Error
	duration := time.Since(start)

	if err != nil {
		logger.GetLogger().Error("Repository: Failed to get API config by ID",
			zap.Uint("config_id", id),
			zap.Duration("query_duration", duration),
			zap.Error(err),
		)
		return nil, err
	}

	logger.GetLogger().Debug("Repository: API config retrieved successfully",
		zap.Uint("config_id", id),
		zap.String("slug", res.Slug),
		zap.String("method", res.Method),
		zap.Duration("query_duration", duration),
	)

	return &res, nil
}

// DISABLED: GetByIDGroup - Group-related functions are disabled
func (r *APIConfigRepository) GetByIDGroup(id uint) (*model.APIGroup, error) {
	logger.GetLogger().Debug("Repository: Getting API group by ID",
		zap.Uint("group_id", id),
	)

	start := time.Now()
	var res model.APIGroup
	err := r.db.First(&res, id).Error
	duration := time.Since(start)

	if err != nil {
		logger.GetLogger().Error("Repository: Failed to get API group by ID",
			zap.Uint("group_id", id),
			zap.Duration("query_duration", duration),
			zap.Error(err),
		)
		return nil, err
	}

	logger.GetLogger().Debug("Repository: API group retrieved successfully",
		zap.Uint("group_id", id),
		zap.String("slug", res.Slug),
		zap.String("name", res.Name),
		zap.Duration("query_duration", duration),
	)

	return &res, nil
}

// DISABLED: GetByIDGroupStep - Group-related functions are disabled
func (r *APIConfigRepository) GetByIDGroupStep(ctx context.Context, id uint) (*model.APIGroupStep, error) {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Repository: Context cancelled before getting API group step",
			zap.Uint("step_id", id),
			zap.Error(err),
		)
		return nil, err
	}

	logger.GetLogger().Debug("Repository: Getting API group step by ID",
		zap.Uint("step_id", id),
	)

	start := time.Now()
	// Use WithContext to pass context to database operation
	var res model.APIGroupStep
	err := r.db.WithContext(ctx).First(&res, id).Error
	duration := time.Since(start)

	if err != nil {
		logger.GetLogger().Error("Repository: Failed to get API group step by ID",
			zap.Uint("step_id", id),
			zap.Duration("query_duration", duration),
			zap.Error(err),
		)
		return nil, err
	}

	logger.GetLogger().Debug("Repository: API group step retrieved successfully",
		zap.Uint("step_id", id),
		zap.String("alias", res.Alias),
		zap.Uint("group_id", res.GroupID),
		zap.Duration("query_duration", duration),
	)

	return &res, nil
}

// DISABLED: GetByIDGroupCron - Group-related functions are disabled
func (r *APIConfigRepository) GetByIDGroupCron(ctx context.Context, id uint) (*model.APIGroupCron, error) {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Repository: Context cancelled before getting API group cron",
			zap.Uint("cron_id", id),
			zap.Error(err),
		)
		return nil, err
	}

	logger.GetLogger().Debug("Repository: Getting API group cron by ID",
		zap.Uint("cron_id", id),
	)

	start := time.Now()
	// Use WithContext to pass context to database operation
	var res model.APIGroupCron
	err := r.db.WithContext(ctx).First(&res, id).Error
	duration := time.Since(start)

	if err != nil {
		logger.GetLogger().Error("Repository: Failed to get API group cron by ID",
			zap.Uint("cron_id", id),
			zap.Duration("query_duration", duration),
			zap.Error(err),
		)
		return nil, err
	}

	logger.GetLogger().Debug("Repository: API group cron retrieved successfully",
		zap.Uint("cron_id", id),
		zap.String("slug", res.Slug),
		zap.String("schedule", res.Schedule),
		zap.Bool("enabled", res.Enabled),
		zap.Duration("query_duration", duration),
	)

	return &res, nil
}

//End Get By ID

// Start By Slug

func (r *APIConfigRepository) FindBySlugConfig(ctx context.Context, slug string) (*model.APIConfig, error) {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Repository: Context cancelled before finding API config by slug",
			zap.String("slug", slug),
			zap.Error(err),
		)
		return nil, err
	}

	logger.GetLogger().Debug("Repository: Finding API config by slug",
		zap.String("slug", slug),
	)

	start := time.Now()
	var res model.APIConfig
	err := r.db.WithContext(ctx).Preload("URLConfig").Where("slug = ?", slug).First(&res).Error
	duration := time.Since(start)

	if err != nil {
		logger.GetLogger().Error("Repository: Failed to find API config by slug",
			zap.String("slug", slug),
			zap.Duration("query_duration", duration),
			zap.Error(err),
		)
		return nil, err
	}

	logger.GetLogger().Debug("Repository: API config found by slug successfully",
		zap.String("slug", slug),
		zap.Uint("config_id", res.ID),
		zap.String("method", res.Method),
		zap.Duration("query_duration", duration),
	)

	return &res, nil
}

// DISABLED: FindBySlugGroup - Group-related functions are disabled
func (r *APIConfigRepository) FindBySlugGroup(slug string) (*model.APIGroup, error) {
	logger.GetLogger().Debug("Repository: Finding API group by slug",
		zap.String("slug", slug),
	)

	start := time.Now()
	var group model.APIGroup
	err := r.db.Preload("Steps").Where("slug = ?", slug).First(&group).Error
	duration := time.Since(start)

	if err != nil {
		logger.GetLogger().Error("Repository: Failed to find API group by slug",
			zap.String("slug", slug),
			zap.Duration("query_duration", duration),
			zap.Error(err),
		)
		return nil, err
	}

	logger.GetLogger().Debug("Repository: API group found by slug successfully",
		zap.String("slug", slug),
		zap.Uint("group_id", group.ID),
		zap.String("name", group.Name),
		zap.Int("steps_count", len(group.Steps)),
		zap.Duration("query_duration", duration),
	)

	return &group, nil
}

// DISABLED: FindByURIConfig - Dynamic routing functions are disabled
// FindByURIConfig finds API config by custom URI and method
func (r *APIConfigRepository) FindByURIConfig(ctx context.Context, uri, method string) (*model.APIConfig, error) {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Repository: Context cancelled before finding API config by URI",
			zap.String("uri", uri),
			zap.String("method", method),
			zap.Error(err),
		)
		return nil, err
	}

	logger.GetLogger().Debug("Repository: Finding API config by URI and method",
		zap.String("uri", uri),
		zap.String("method", method),
	)

	start := time.Now()
	var res model.APIConfig
	// For HTTP protocol, match URI and method exactly
	// For gRPC protocol, match URI (any method allowed)
	err := r.db.WithContext(ctx).
		Preload("URLConfig").
		Where("uri = ? AND method = ?", uri, method).
		First(&res).Error

	duration := time.Since(start)

	if err != nil {
		logger.GetLogger().Error("Repository: Failed to find API config by URI and method",
			zap.String("uri", uri),
			zap.String("method", method),
			zap.Duration("query_duration", duration),
			zap.Error(err),
		)
		return nil, err
	}

	logger.GetLogger().Debug("Repository: API config found by URI and method successfully",
		zap.String("uri", uri),
		zap.String("method", method),
		zap.Uint("config_id", res.ID),
		zap.String("slug", res.Slug),
		zap.Duration("query_duration", duration),
	)

	return &res, nil
}

// DISABLED: FindByURIConfigWithPattern - Dynamic routing functions are disabled
func (r *APIConfigRepository) FindByURIConfigWithPattern(ctx context.Context, requestURI, method string) (*model.APIConfig, map[string]string, error) {
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Repository: Context cancelled before finding API config by URI pattern",
			zap.String("request_uri", requestURI),
			zap.String("method", method),
			zap.Error(err),
		)
		return nil, nil, err
	}

	logger.GetLogger().Debug("Repository: Finding API config by URI pattern and method",
		zap.String("request_uri", requestURI),
		zap.String("method", method),
	)

	start := time.Now()

	// First try exact match
	exactConfig, err := r.FindByURIConfig(ctx, requestURI, method)
	if err == nil {
		duration := time.Since(start)
		logger.GetLogger().Debug("Repository: Found exact URI match",
			zap.String("request_uri", requestURI),
			zap.String("method", method),
			zap.Uint("config_id", exactConfig.ID),
			zap.Duration("query_duration", duration),
		)
		return exactConfig, nil, nil
	}

	// If no exact match, try pattern matching
	var configs []model.APIConfig
	err = r.db.WithContext(ctx).
		Preload("URLConfig").
		Where("method = ?", method).
		Find(&configs).Error

	if err != nil {
		duration := time.Since(start)
		logger.GetLogger().Error("Repository: Failed to fetch API configs for pattern matching",
			zap.String("request_uri", requestURI),
			zap.String("method", method),
			zap.Duration("query_duration", duration),
			zap.Error(err),
		)
		return nil, nil, err
	}

	// Try to match each config's URI pattern against the request URI
	for _, config := range configs {
		if config.URI == "" {
			continue // Skip empty URIs
		}

		params, isMatch := r.matchURI(config.URI, requestURI)
		if isMatch {
			duration := time.Since(start)
			logger.GetLogger().Debug("Repository: Found pattern match for URI",
				zap.String("config_uri", config.URI),
				zap.String("request_uri", requestURI),
				zap.String("method", method),
				zap.Uint("config_id", config.ID),
				zap.Int("matched_params", len(params)),
				zap.Duration("query_duration", duration),
			)
			return &config, params, nil
		}
	}

	duration := time.Since(start)
	logger.GetLogger().Debug("Repository: No URI pattern match found",
		zap.String("request_uri", requestURI),
		zap.String("method", method),
		zap.Duration("query_duration", duration),
	)

	return nil, nil, gorm.ErrRecordNotFound
}

// matchURI checks if a URI pattern matches the request URI and extracts parameters
// Pattern: /api/v1/user/{id} can match /api/v1/user/123
func (r *APIConfigRepository) matchURI(pattern, requestURI string) (map[string]string, bool) {
	patternParts := r.splitPath(pattern)
	requestParts := r.splitPath(requestURI)

	// Different number of parts means no match
	if len(patternParts) != len(requestParts) {
		return nil, false
	}

	params := make(map[string]string)

	for i, patternPart := range patternParts {
		requestPart := requestParts[i]

		// Check if this part is a parameter (wrapped in {})
		if len(patternPart) > 2 && patternPart[0] == '{' && patternPart[len(patternPart)-1] == '}' {
			// Extract parameter name without {}
			paramName := patternPart[1 : len(patternPart)-1]
			params[paramName] = requestPart
		} else if patternPart != requestPart {
			// Static parts must match exactly
			return nil, false
		}
	}

	return params, true
}

// splitPath splits a URI path into its components
func (r *APIConfigRepository) splitPath(path string) []string {
	if path == "" || path == "/" {
		return []string{}
	}

	// Remove leading slash and split
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return []string{}
	}

	return strings.Split(path, "/")
}

//End By Slug

// Start All Config
func (r *APIConfigRepository) GetAllConfig(limit, offset int, search string, urlConfigID ...uint) ([]model.APIConfig, int64, error) {
	logger.GetLogger().Debug("Repository: Getting all API configs",
		zap.Int("limit", limit),
		zap.Int("offset", offset),
		zap.String("search", search),
		zap.Any("url_config_id", urlConfigID),
	)

	start := time.Now()
	var pages []model.APIConfig
	var total int64

	query := r.db.Model(&model.APIConfig{})

	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where(
			"slug ILIKE ? OR method ILIKE ? OR description ILIKE ?",
			searchPattern, searchPattern, searchPattern,
		)
		logger.GetLogger().Debug("Repository: Applied search filter for API configs",
			zap.String("search", search),
			zap.String("search_pattern", searchPattern),
		)
	}

	// Add URL config ID filter if provided
	if len(urlConfigID) > 0 && urlConfigID[0] > 0 {
		query = query.Where("url_config_id = ?", urlConfigID[0])
		logger.GetLogger().Debug("Repository: Applied URL config ID filter",
			zap.Uint("url_config_id", urlConfigID[0]),
		)
	}

	if err := query.Count(&total).Error; err != nil {
		logger.GetLogger().Error("Repository: Failed to count total API configs",
			zap.Error(err),
		)
		return nil, 0, err
	}

	if err := query.Preload("URLConfig").Limit(limit).Offset(offset).Find(&pages).Error; err != nil {
		logger.GetLogger().Error("Repository: Failed to fetch API configs",
			zap.Int("limit", limit),
			zap.Int("offset", offset),
			zap.String("search", search),
			zap.Duration("query_time", time.Since(start)),
			zap.Error(err),
		)
		return nil, 0, err
	}

	queryDuration := time.Since(start)
	logger.GetLogger().Info("Repository: API configs retrieved successfully",
		zap.Int("limit", limit),
		zap.Int("offset", offset),
		zap.String("search", search),
		zap.Int64("total", total),
		zap.Int("returned_count", len(pages)),
		zap.Duration("query_duration", queryDuration),
	)

	return pages, total, nil
}

// DISABLED: GetAllGroup - Group-related functions are disabled
func (r *APIConfigRepository) GetAllGroup(limit, offset int, search string) ([]model.APIGroup, int64, error) {
	logger.GetLogger().Debug("Repository: Getting all API groups",
		zap.Int("limit", limit),
		zap.Int("offset", offset),
		zap.String("search", search),
	)

	start := time.Now()
	var pages []model.APIGroup
	var total int64

	query := r.db.Model(&model.APIGroup{})

	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where(
			"slug ILIKE ? OR name ILIKE ?",
			searchPattern, searchPattern,
		)
		logger.GetLogger().Debug("Repository: Applied search filter for API groups",
			zap.String("search", search),
			zap.String("search_pattern", searchPattern),
		)
	}

	if err := query.Count(&total).Error; err != nil {
		logger.GetLogger().Error("Repository: Failed to count total API groups",
			zap.Error(err),
		)
		return nil, 0, err
	}

	if err := query.Limit(limit).Offset(offset).Find(&pages).Error; err != nil {
		logger.GetLogger().Error("Repository: Failed to fetch API groups",
			zap.Int("limit", limit),
			zap.Int("offset", offset),
			zap.String("search", search),
			zap.Duration("query_time", time.Since(start)),
			zap.Error(err),
		)
		return nil, 0, err
	}

	queryDuration := time.Since(start)
	logger.GetLogger().Info("Repository: API groups retrieved successfully",
		zap.Int("limit", limit),
		zap.Int("offset", offset),
		zap.String("search", search),
		zap.Int64("total", total),
		zap.Int("returned_count", len(pages)),
		zap.Duration("query_duration", queryDuration),
	)

	return pages, total, nil
}

// DISABLED: GetAllGroupStep - Group-related functions are disabled
func (r *APIConfigRepository) GetAllGroupStep(limit, offset int, search string, groupID uint) ([]model.APIGroupStep, int64, error) {
	logger.GetLogger().Debug("Repository: Getting all API group steps",
		zap.Int("limit", limit),
		zap.Int("offset", offset),
		zap.String("search", search),
		zap.Uint("group_id", groupID),
	)

	start := time.Now()
	var pages []model.APIGroupStep
	var total int64

	query := r.db.Model(&model.APIGroupStep{})

	// Add filter by group_id
	if groupID > 0 {
		query = query.Where("group_id = ?", groupID)
		logger.GetLogger().Debug("Repository: Applied group filter for API group steps",
			zap.Uint("group_id", groupID),
		)
	}

	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where("alias ILIKE ?", searchPattern)
		logger.GetLogger().Debug("Repository: Applied search filter for API group steps",
			zap.String("search", search),
			zap.String("search_pattern", searchPattern),
		)
	}

	if err := query.Count(&total).Error; err != nil {
		logger.GetLogger().Error("Repository: Failed to count total API group steps",
			zap.Error(err),
		)
		return nil, 0, err
	}

	if err := query.Limit(limit).Offset(offset).Find(&pages).Error; err != nil {
		logger.GetLogger().Error("Repository: Failed to fetch API group steps",
			zap.Int("limit", limit),
			zap.Int("offset", offset),
			zap.String("search", search),
			zap.Uint("group_id", groupID),
			zap.Duration("query_time", time.Since(start)),
			zap.Error(err),
		)
		return nil, 0, err
	}

	queryDuration := time.Since(start)
	logger.GetLogger().Info("Repository: API group steps retrieved successfully",
		zap.Int("limit", limit),
		zap.Int("offset", offset),
		zap.String("search", search),
		zap.Uint("group_id", groupID),
		zap.Int64("total", total),
		zap.Int("returned_count", len(pages)),
		zap.Duration("query_duration", queryDuration),
	)

	return pages, total, nil
}

// DISABLED: GetAllGroupCron - Group-related functions are disabled
func (r *APIConfigRepository) GetAllGroupCron(limit, offset int, search, slug string) ([]model.APIGroupCron, int64, error) {
	logger.GetLogger().Debug("Repository: Getting all API group crons",
		zap.Int("limit", limit),
		zap.Int("offset", offset),
		zap.String("search", search),
		zap.String("slug", slug),
	)

	start := time.Now()
	var pages []model.APIGroupCron
	var total int64

	query := r.db.Model(&model.APIGroupCron{})

	if slug != "" {
		query = query.Where("slug = ?", slug)
		logger.GetLogger().Debug("Repository: Applied slug filter for API group crons",
			zap.String("slug", slug),
		)
	}

	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where("slug ILIKE ?", searchPattern)
		logger.GetLogger().Debug("Repository: Applied search filter for API group crons",
			zap.String("search", search),
			zap.String("search_pattern", searchPattern),
		)
	}

	if err := query.Count(&total).Error; err != nil {
		logger.GetLogger().Error("Repository: Failed to count total API group crons",
			zap.Error(err),
		)
		return nil, 0, err
	}

	if err := query.Limit(limit).Offset(offset).Find(&pages).Error; err != nil {
		logger.GetLogger().Error("Repository: Failed to fetch API group crons",
			zap.Int("limit", limit),
			zap.Int("offset", offset),
			zap.String("search", search),
			zap.String("slug", slug),
			zap.Duration("query_time", time.Since(start)),
			zap.Error(err),
		)
		return nil, 0, err
	}

	queryDuration := time.Since(start)
	logger.GetLogger().Info("Repository: API group crons retrieved successfully",
		zap.Int("limit", limit),
		zap.Int("offset", offset),
		zap.String("search", search),
		zap.String("slug", slug),
		zap.Int64("total", total),
		zap.Int("returned_count", len(pages)),
		zap.Duration("query_duration", queryDuration),
	)

	return pages, total, nil
}

// End All Config
