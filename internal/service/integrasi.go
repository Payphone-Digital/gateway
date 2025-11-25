package service

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strings"

	"github.com/surdiana/gateway/internal/dto"
	"github.com/surdiana/gateway/internal/model"
	"github.com/surdiana/gateway/internal/repository"
	"github.com/surdiana/gateway/pkg/logger"
	"go.uber.org/zap"
)

// Helper function to safely get string value from pointer
func getStringValue(ptr *string) string {
	if ptr != nil {
		return *ptr
	}
	return ""
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

type APIConfigService struct {
	repo *repository.APIConfigRepository
}

func NewAPIConfigService(repo *repository.APIConfigRepository) *APIConfigService {
	return &APIConfigService{
		repo: repo,
	}
}

// Start Create
func (s *APIConfigService) CreateConfig(ctx context.Context, req dto.APIConfigRequest) (int, error) {
	// Check context cancellation first
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before creating API config",
			zap.String("slug", req.Slug),
			zap.Error(err),
		)
		return http.StatusRequestTimeout, err
	}

	logger.GetLogger().Info("Service: Creating API config",
		zap.String("slug", req.Slug),
		zap.String("method", req.Method),
		zap.Uint("url_config_id", req.URLConfigID),
		zap.Int("max_retries", req.MaxRetries),
		zap.Int("retry_delay", req.RetryDelay),
		zap.Int("timeout", req.Timeout),
		zap.Bool("is_admin", req.IsAdmin),
		zap.Bool("has_manipulation", req.Manipulation != ""),
		zap.Bool("has_description", req.Description != ""),
		zap.Int("headers_count", len(req.Headers)),
		zap.Int("query_params_count", len(req.QueryParams)),
		zap.Int("variables_count", len(req.Variables)),
		zap.Bool("has_body", req.Body != nil),
	)

	headersJSON, _ := json.Marshal(req.Headers)
	queryParamsJSON, _ := json.Marshal(req.QueryParams)
	bodyJSON, _ := json.Marshal(req.Body)
	variablesJSON, _ := json.Marshal(req.Variables)

	apiConfig := &model.APIConfig{
		Slug:         req.Slug,
		Method:       req.Method,
		URLConfigID:  req.URLConfigID,
		URI:          req.URI,
		Headers:      headersJSON,
		QueryParams:  queryParamsJSON,
		Body:         bodyJSON,
		Variables:    variablesJSON,
		MaxRetries:   req.MaxRetries,
		RetryDelay:   req.RetryDelay,
		Timeout:      req.Timeout,
		Manipulation: req.Manipulation,
		Description:  req.Description,
		IsAdmin:      req.IsAdmin,
	}

	if _, err := s.repo.FindBySlugConfig(ctx, req.Slug); err == nil {
		logger.GetLogger().Warn("Service: API config with slug already exists",
			zap.String("slug", req.Slug),
		)
		return http.StatusConflict, errors.New("API config with this slug already exists")
	}

	// Check context before expensive operation
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before database operation",
			zap.String("slug", req.Slug),
			zap.Error(err),
		)
		return http.StatusRequestTimeout, err
	}

	if err := s.repo.CreateConfig(ctx, apiConfig); err != nil {
		logger.GetLogger().Error("Service: Failed to create API config",
			zap.String("slug", req.Slug),
			zap.Error(err),
		)
		return http.StatusInternalServerError, err
	}

	logger.GetLogger().Info("Service: API config created successfully",
		zap.String("slug", req.Slug),
	)

	return http.StatusCreated, nil
}

// DISABLED: Group creation function
func (s *APIConfigService) CreateGroup(ctx context.Context, req dto.APIGroupRequest) (int, error) {
	/*
	// Check context cancellation first
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before creating API group",
			zap.String("slug", req.Slug),
			zap.Error(err),
		)
		return http.StatusRequestTimeout, err
	}

	logger.GetLogger().Info("Service: Creating API group",
		zap.String("slug", req.Slug),
		zap.String("name", req.Name),
		zap.Bool("is_admin", req.IsAdmin),
	)

	apiGroup := &model.APIGroup{
		Slug:    req.Slug,
		Name:    req.Name,
		IsAdmin: req.IsAdmin,
	}

	// Check context before expensive database query
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before slug check",
			zap.String("slug", req.Slug),
			zap.Error(err),
		)
		return http.StatusRequestTimeout, err
	}

	if _, err := s.repo.FindBySlugGroup(req.Slug); err == nil {
		logger.GetLogger().Warn("Service: API group with slug already exists",
			zap.String("slug", req.Slug),
		)
		return http.StatusConflict, errors.New("API Group with this slug already exists")
	}

	// Check context before database operation
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before database operation",
			zap.String("slug", req.Slug),
			zap.Error(err),
		)
		return http.StatusRequestTimeout, err
	}

	if err := s.repo.CreateGroup(ctx, apiGroup); err != nil {
		logger.GetLogger().Error("Service: Failed to create API group",
			zap.String("slug", req.Slug),
			zap.Error(err),
		)
		return http.StatusInternalServerError, err
	}

	logger.GetLogger().Info("Service: API group created successfully",
		zap.String("slug", req.Slug),
	)

	return http.StatusCreated, nil
	*/
	return http.StatusNotImplemented, errors.New("Group functions are disabled")
}

// DISABLED: Group step creation function
func (s *APIConfigService) CreateGroupStep(ctx context.Context, req dto.APIGroupStepRequest) (int, error) {
	/*
	// Check context cancellation first
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before creating API group step",
			zap.Uint("group_id", req.GroupID),
			zap.String("alias", req.Alias),
			zap.Error(err),
		)
		return http.StatusRequestTimeout, err
	}

	logger.GetLogger().Info("Service: Creating API group step",
		zap.Uint("group_id", req.GroupID),
		zap.Uint("api_config_id", req.APIConfigID),
		zap.String("alias", req.Alias),
		zap.Int("order_index", req.OrderIndex),
		zap.Int("variables_count", len(req.Variables)),
	)

	variablesJSON, _ := json.Marshal(req.Variables)
	groupStep := &model.APIGroupStep{
		GroupID:     req.GroupID,
		APIConfigID: req.APIConfigID,
		OrderIndex:  req.OrderIndex,
		Alias:       req.Alias,
		Variables:   variablesJSON,
	}

	// Check context before database operation
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before database operation",
			zap.Uint("group_id", req.GroupID),
			zap.String("alias", req.Alias),
			zap.Error(err),
		)
		return http.StatusRequestTimeout, err
	}

	if err := s.repo.CreateGroupStep(ctx, groupStep); err != nil {
		logger.GetLogger().Error("Service: Failed to create API group step",
			zap.Uint("group_id", req.GroupID),
			zap.String("alias", req.Alias),
			zap.Error(err),
		)
		return http.StatusInternalServerError, err
	}

	logger.GetLogger().Info("Service: API group step created successfully",
		zap.Uint("group_id", req.GroupID),
		zap.String("alias", req.Alias),
	)

	return http.StatusCreated, nil
	*/
	return http.StatusNotImplemented, errors.New("Group step functions are disabled")
}

// DISABLED: Group cron creation function
func (s *APIConfigService) CreateGroupCron(ctx context.Context, req dto.APIGroupCronRequest) (int, error) {
	/*
	// Check context cancellation first
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before creating API group cron",
			zap.String("slug", req.Slug),
			zap.Error(err),
		)
		return http.StatusRequestTimeout, err
	}

	logger.GetLogger().Info("Service: Creating API group cron",
		zap.String("slug", req.Slug),
		zap.String("schedule", req.Schedule),
		zap.Bool("enabled", req.Enabled),
	)

	apiGroupCron := &model.APIGroupCron{
		Slug:     req.Slug,
		Schedule: req.Schedule,
		Enabled:  req.Enabled,
	}

	// Check context before expensive database query
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before slug check",
			zap.String("slug", req.Slug),
			zap.Error(err),
		)
		return http.StatusRequestTimeout, err
	}

	// Check if group exists
	if _, err := s.repo.FindBySlugGroup(req.Slug); err != nil {
		logger.GetLogger().Warn("Service: API group not found for cron",
			zap.String("slug", req.Slug),
		)
		return http.StatusNotFound, errors.New("API Group with this slug not found")
	}

	// Check context before database operation
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before database operation",
			zap.String("slug", req.Slug),
			zap.Error(err),
		)
		return http.StatusRequestTimeout, err
	}

	if err := s.repo.CreateGroupCron(ctx, apiGroupCron); err != nil {
		logger.GetLogger().Error("Service: Failed to create API group cron",
			zap.String("slug", req.Slug),
			zap.Error(err),
		)
		return http.StatusInternalServerError, err
	}

	logger.GetLogger().Info("Service: API group cron created successfully",
		zap.String("slug", req.Slug),
	)

	return http.StatusCreated, nil
	*/
	return http.StatusNotImplemented, errors.New("Group cron functions are disabled")
}

// End Create

// Start Update
func (s *APIConfigService) UpdateConfig(ctx context.Context, id uint, req dto.APIConfigRequest) (int, error) {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before updating API config",
			zap.Uint("config_id", id),
			zap.Error(err),
		)
		return http.StatusRequestTimeout, err
	}

	headersJSON, _ := json.Marshal(req.Headers)
	queryParamsJSON, _ := json.Marshal(req.QueryParams)
	bodyJSON, _ := json.Marshal(req.Body)
	variablesJSON, _ := json.Marshal(req.Variables)

	apiConfig := &model.APIConfig{
		Slug:         req.Slug,
		Method:       req.Method,
		URLConfigID:  req.URLConfigID,
		URI:          req.URI,
		Headers:      headersJSON,
		QueryParams:  queryParamsJSON,
		Body:         bodyJSON,
		Variables:    variablesJSON,
		MaxRetries:   req.MaxRetries,
		RetryDelay:   req.RetryDelay,
		Timeout:      req.Timeout,
		Manipulation: req.Manipulation,
		Description:  req.Description,
		IsAdmin:      req.IsAdmin,
	}
	apiConfig.ID = id

	existing, err := s.repo.GetByIDConfig(id)
	if err != nil || existing == nil {
		return http.StatusNotFound, errors.New("API config not found")
	}

	if existing.Slug != req.Slug {
		if _, err := s.repo.FindBySlugConfig(ctx, req.Slug); err == nil {
			return http.StatusConflict, errors.New("API config with this slug already exists")
		}
	}

	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled during update operation",
			zap.Uint("config_id", id),
			zap.Error(err),
		)
		return http.StatusRequestTimeout, err
	}

	if err := s.repo.UpdateConfig(ctx, apiConfig); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

// DISABLED: Group update function
func (s *APIConfigService) UpdateGroup(id uint, req dto.APIGroupRequest) (int, error) {
	/*
	apiGroup := &model.APIGroup{
		Slug:    req.Slug,
		Name:    req.Name,
		IsAdmin: req.IsAdmin,
	}
	apiGroup.ID = id

	existing, err := s.repo.GetByIDGroup(id)
	if err != nil || existing == nil {
		return http.StatusNotFound, errors.New("API Group not found")
	}

	if existing.Slug != req.Slug {
		if _, err := s.repo.FindBySlugGroup(req.Slug); err == nil {
			return http.StatusConflict, errors.New("API Group with this slug already exists")
		}
	}

	if err := s.repo.UpdateGroup(apiGroup); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
	*/
	return http.StatusNotImplemented, errors.New("Group functions are disabled")
}

// DISABLED: Group step update function
func (s *APIConfigService) UpdateGroupStep(id uint, req dto.APIGroupStepRequest) (int, error) {
	/*
	variablesJSON, _ := json.Marshal(req.Variables)
	groupStep := &model.APIGroupStep{
		GroupID:     req.GroupID,
		APIConfigID: req.APIConfigID,
		OrderIndex:  req.OrderIndex,
		Alias:       req.Alias,
		Variables:   variablesJSON,
	}
	groupStep.ID = id

	existing, err := s.repo.GetByIDGroupStep(context.Background(), id)
	if err != nil || existing == nil {
		return http.StatusNotFound, errors.New("API Group Step not found")
	}

	if err := s.repo.UpdateGroupStep(groupStep); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
	*/
	return http.StatusNotImplemented, errors.New("Group step functions are disabled")
}

// DISABLED: Group cron update function
func (s *APIConfigService) UpdateGroupCron(id uint, req dto.APIGroupCronRequest) (int, error) {
	/*
	apiGroupCron := &model.APIGroupCron{
		Slug:     req.Slug,
		Schedule: req.Schedule,
		Enabled:  req.Enabled,
	}
	apiGroupCron.ID = id
	existing, err := s.repo.GetByIDGroupCron(context.Background(), id)
	if err != nil || existing == nil {
		return http.StatusNotFound, errors.New("API Group Cron not found")
	}

	if err := s.repo.UpdateGroupCron(apiGroupCron); err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
	*/
	return http.StatusNotImplemented, errors.New("Group cron functions are disabled")
}

//End Update

// Start Delete
func (s *APIConfigService) DeleteConfig(ctx context.Context, id uint) (int, error) {
	// Check context cancellation first
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before deleting API config",
			zap.Uint("config_id", id),
			zap.Error(err),
		)
		return http.StatusRequestTimeout, err
	}

	logger.GetLogger().Info("Service: Deleting API config",
		zap.Uint("config_id", id),
	)

	// Check context before expensive database query
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before config check",
			zap.Uint("config_id", id),
			zap.Error(err),
		)
		return http.StatusRequestTimeout, err
	}

	// Check if config exists
	if _, err := s.repo.GetByIDConfig(id); err != nil {
		logger.GetLogger().Warn("Service: API config not found for deletion",
			zap.Uint("config_id", id),
		)
		return http.StatusNotFound, errors.New("API config not found")
	}

	// Check context before database operation
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before database operation",
			zap.Uint("config_id", id),
			zap.Error(err),
		)
		return http.StatusRequestTimeout, err
	}

	if err := s.repo.DeleteConfig(ctx, id); err != nil {
		logger.GetLogger().Error("Service: Failed to delete API config",
			zap.Uint("config_id", id),
			zap.Error(err),
		)
		return http.StatusInternalServerError, err
	}

	logger.GetLogger().Info("Service: API config deleted successfully",
		zap.Uint("config_id", id),
	)

	return http.StatusNoContent, nil
}

// DISABLED: Group deletion function
func (s *APIConfigService) DeleteGroup(ctx context.Context, id uint) (int, error) {
	/*
	// Check context cancellation first
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before deleting API group",
			zap.Uint("group_id", id),
			zap.Error(err),
		)
		return http.StatusRequestTimeout, err
	}

	logger.GetLogger().Info("Service: Deleting API group",
		zap.Uint("group_id", id),
	)

	// Check context before expensive database query
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before group check",
			zap.Uint("group_id", id),
			zap.Error(err),
		)
		return http.StatusRequestTimeout, err
	}

	// Check if group exists
	if _, err := s.repo.GetByIDGroup(id); err != nil {
		logger.GetLogger().Warn("Service: API group not found for deletion",
			zap.Uint("group_id", id),
		)
		return http.StatusNotFound, errors.New("API Group not found")
	}

	// Check context before database operation
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before database operation",
			zap.Uint("group_id", id),
			zap.Error(err),
		)
		return http.StatusRequestTimeout, err
	}

	if err := s.repo.DeleteGroup(ctx, id); err != nil {
		logger.GetLogger().Error("Service: Failed to delete API group",
			zap.Uint("group_id", id),
			zap.Error(err),
		)
		return http.StatusInternalServerError, err
	}

	logger.GetLogger().Info("Service: API group deleted successfully",
		zap.Uint("group_id", id),
	)

	return http.StatusNoContent, nil
	*/
	return http.StatusNotImplemented, errors.New("Group functions are disabled")
}

// DISABLED: Group step deletion function
func (s *APIConfigService) DeleteGroupStep(ctx context.Context, id uint) (int, error) {
	/*
	// Check context cancellation first
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before deleting API group step",
			zap.Uint("step_id", id),
			zap.Error(err),
		)
		return http.StatusRequestTimeout, err
	}

	logger.GetLogger().Info("Service: Deleting API group step",
		zap.Uint("step_id", id),
	)

	// Check context before expensive database query
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before step check",
			zap.Uint("step_id", id),
			zap.Error(err),
		)
		return http.StatusRequestTimeout, err
	}

	// Check if step exists
	if _, err := s.repo.GetByIDGroupStep(context.Background(), id); err != nil {
		logger.GetLogger().Warn("Service: API group step not found for deletion",
			zap.Uint("step_id", id),
		)
		return http.StatusNotFound, errors.New("API Group Step not found")
	}

	// Check context before database operation
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before database operation",
			zap.Uint("step_id", id),
			zap.Error(err),
		)
		return http.StatusRequestTimeout, err
	}

	if err := s.repo.DeleteGroupStep(ctx, id); err != nil {
		logger.GetLogger().Error("Service: Failed to delete API group step",
			zap.Uint("step_id", id),
			zap.Error(err),
		)
		return http.StatusInternalServerError, err
	}

	logger.GetLogger().Info("Service: API group step deleted successfully",
		zap.Uint("step_id", id),
	)

	return http.StatusNoContent, nil
	*/
	return http.StatusNotImplemented, errors.New("Group step functions are disabled")
}

// DISABLED: Group cron deletion function
func (s *APIConfigService) DeleteGroupCron(ctx context.Context, id uint) (int, error) {
	/*
	// Check context cancellation first
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before deleting API group cron",
			zap.Uint("cron_id", id),
			zap.Error(err),
		)
		return http.StatusRequestTimeout, err
	}

	logger.GetLogger().Info("Service: Deleting API group cron",
		zap.Uint("cron_id", id),
	)

	// Check context before expensive database query
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before cron check",
			zap.Uint("cron_id", id),
			zap.Error(err),
		)
		return http.StatusRequestTimeout, err
	}

	// Check if cron exists
	if _, err := s.repo.GetByIDGroupCron(context.Background(), id); err != nil {
		logger.GetLogger().Warn("Service: API group cron not found for deletion",
			zap.Uint("cron_id", id),
		)
		return http.StatusNotFound, errors.New("API Group Cron not found")
	}

	// Check context before database operation
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before database operation",
			zap.Uint("cron_id", id),
			zap.Error(err),
		)
		return http.StatusRequestTimeout, err
	}

	if err := s.repo.DeleteGroupCron(ctx, id); err != nil {
		logger.GetLogger().Error("Service: Failed to delete API group cron",
			zap.Uint("cron_id", id),
			zap.Error(err),
		)
		return http.StatusInternalServerError, err
	}

	logger.GetLogger().Info("Service: API group cron deleted successfully",
		zap.Uint("cron_id", id),
	)

	return http.StatusNoContent, nil
	*/
	return http.StatusNotImplemented, errors.New("Group cron functions are disabled")
}

// End Delete

// Start Get By ID
func (s *APIConfigService) GetByIDConfig(ctx context.Context, id uint) (*dto.APIConfigResponse, int, error) {
	// Check context cancellation first
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before getting API config",
			zap.Uint("config_id", id),
			zap.Error(err),
		)
		return nil, http.StatusRequestTimeout, err
	}

	logger.GetLogger().Info("Service: Getting API config",
		zap.Uint("config_id", id),
	)

	// Check context before expensive database query
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before database operation",
			zap.Uint("config_id", id),
			zap.Error(err),
		)
		return nil, http.StatusRequestTimeout, err
	}

	res, err := s.repo.GetByIDConfig(id)
	if err != nil {
		logger.GetLogger().Warn("Service: API config not found",
			zap.Uint("config_id", id),
		)
		return nil, http.StatusNotFound, errors.New("API config not found")
	}

	var headers map[string]string
	var queryParams map[string]string
	var variables map[string]dto.Variable
	var rawBody interface{}

	_ = json.Unmarshal(res.Headers, &headers)
	_ = json.Unmarshal(res.QueryParams, &queryParams)
	_ = json.Unmarshal(res.Variables, &variables)

	if err := json.Unmarshal(res.Body, &rawBody); err != nil {
		// error handling jika perlu
	}

	if bodyStr, ok := rawBody.(string); ok {
		var bodyObj map[string]interface{}
		if err := json.Unmarshal([]byte(bodyStr), &bodyObj); err == nil {
			rawBody = bodyObj
		}
	}

	body, _ := rawBody.(map[string]interface{})

	// Combine base URL with URI to get complete URL
	completeURL := res.URLConfig.URL
	if res.URI != "" {
		// Remove trailing slash from base URL and leading slash from URI to avoid double slashes
		baseURL := strings.TrimSuffix(res.URLConfig.URL, "/")
		uri := strings.TrimPrefix(res.URI, "/")
		completeURL = baseURL + "/" + uri
	}

	resp := &dto.APIConfigResponse{
		ID:           res.ID,
		Slug:         res.Slug,
		Protocol:     res.URLConfig.Protocol, // Get protocol from URLConfig
		Method:       res.Method,
		URLConfigID:  res.URLConfigID,
		URI:          res.URI,
		URL:          completeURL,
		URLConfig: dto.URLConfigResponse{
			ID:        res.URLConfig.ID,
			Nama:      res.URLConfig.Nama,
			Protocol:  res.URLConfig.Protocol,
			URL:       res.URLConfig.URL,
			Deskripsi: res.URLConfig.Deskripsi,
			IsActive:  res.URLConfig.IsActive,
			GRPCService: getStringValue(res.URLConfig.GRPCService),
					ProtoFile:   getStringValue(res.URLConfig.ProtoFile),
			TLSEnabled:  res.URLConfig.TLSEnabled,
		},
		Headers:      headers,
		QueryParams:  queryParams,
		Body:         body,
		Variables:    variables,
		MaxRetries:   res.MaxRetries,
		RetryDelay:   res.RetryDelay,
		Timeout:      res.Timeout,
		Manipulation: res.Manipulation,
		Description:  res.Description,
		IsAdmin:      res.IsAdmin,
	}

	logger.GetLogger().Info("Service: API config retrieved successfully",
		zap.Uint("config_id", id),
		zap.String("slug", res.Slug),
		zap.String("url", res.URLConfig.URL),
	)

	return resp, http.StatusOK, nil
}

// DISABLED: Group retrieval by ID function
func (s *APIConfigService) GetByIDGroup(ctx context.Context, id uint) (*dto.APIGroupResponse, int, error) {
	/*
	// Check context cancellation first
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before getting API group",
			zap.Uint("group_id", id),
			zap.Error(err),
		)
		return nil, http.StatusRequestTimeout, err
	}

	logger.GetLogger().Info("Service: Getting API group",
		zap.Uint("group_id", id),
	)

	// Check context before expensive database query
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before database operation",
			zap.Uint("group_id", id),
			zap.Error(err),
		)
		return nil, http.StatusRequestTimeout, err
	}

	res, err := s.repo.GetByIDGroup(id)
	if err != nil {
		logger.GetLogger().Warn("Service: API group not found",
			zap.Uint("group_id", id),
		)
		return nil, http.StatusNotFound, errors.New("API Group not found")
	}

	resp := &dto.APIGroupResponse{
		ID:      res.ID,
		Slug:    res.Slug,
		Name:    res.Name,
		IsAdmin: res.IsAdmin,
	}

	logger.GetLogger().Info("Service: API group retrieved successfully",
		zap.Uint("group_id", id),
		zap.String("slug", res.Slug),
	)

	return resp, http.StatusOK, nil
	*/
	return nil, http.StatusNotImplemented, errors.New("Group functions are disabled")
}

// DISABLED: Group step retrieval by ID function
func (s *APIConfigService) GetByIDGroupStep(ctx context.Context, id uint) (*dto.APIGroupStepResponse, int, error) {
	/*
	// Check context cancellation first
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before getting API group step",
			zap.Uint("step_id", id),
			zap.Error(err),
		)
		return nil, http.StatusRequestTimeout, err
	}

	logger.GetLogger().Info("Service: Getting API group step",
		zap.Uint("step_id", id),
	)

	// Check context before expensive database query
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before database operation",
			zap.Uint("step_id", id),
			zap.Error(err),
		)
		return nil, http.StatusRequestTimeout, err
	}

	res, err := s.repo.GetByIDGroupStep(ctx, id)
	if err != nil {
		logger.GetLogger().Warn("Service: API group step not found",
			zap.Uint("step_id", id),
		)
		return nil, http.StatusNotFound, errors.New("API Group Step not found")
	}

	var variables map[string]interface{}
	_ = json.Unmarshal(res.Variables, &variables)

	resp := &dto.APIGroupStepResponse{
		ID:          res.ID,
		GroupID:     res.GroupID,
		APIConfigID: res.APIConfigID,
		OrderIndex:  res.OrderIndex,
		Alias:       res.Alias,
		Variables:   variables,
	}

	logger.GetLogger().Info("Service: API group step retrieved successfully",
		zap.Uint("step_id", id),
		zap.String("alias", res.Alias),
	)

	return resp, http.StatusOK, nil
	*/
	return nil, http.StatusNotImplemented, errors.New("Group step functions are disabled")
}

// DISABLED: Group cron retrieval by ID function
func (s *APIConfigService) GetByIDGroupCron(ctx context.Context, id uint) (*dto.APIGroupCronResponse, int, error) {
	/*
	// Check context cancellation first
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before getting API group cron",
			zap.Uint("cron_id", id),
			zap.Error(err),
		)
		return nil, http.StatusRequestTimeout, err
	}

	logger.GetLogger().Info("Service: Getting API group cron",
		zap.Uint("cron_id", id),
	)

	// Check context before expensive database query
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before database operation",
			zap.Uint("cron_id", id),
			zap.Error(err),
		)
		return nil, http.StatusRequestTimeout, err
	}

	res, err := s.repo.GetByIDGroupCron(ctx, id)
	if err != nil {
		logger.GetLogger().Warn("Service: API group cron not found",
			zap.Uint("cron_id", id),
		)
		return nil, http.StatusNotFound, errors.New("API Group Cron not found")
	}

	resp := &dto.APIGroupCronResponse{
		ID:       res.ID,
		Slug:     res.Slug,
		Schedule: res.Schedule,
		Enabled:  res.Enabled,
	}

	logger.GetLogger().Info("Service: API group cron retrieved successfully",
		zap.Uint("cron_id", id),
		zap.String("slug", res.Slug),
	)

	return resp, http.StatusOK, nil
	*/
	return nil, http.StatusNotImplemented, errors.New("Group cron functions are disabled")
}

// End Get By ID

// Start Get By Slug
// DISABLED: Config retrieval by slug function
func (s *APIConfigService) GetBySlugConfig(ctx context.Context, slug string) (*dto.APIConfigResponse, int, error) {
	/*
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before getting API config by slug",
			zap.String("slug", slug),
			zap.Error(err),
		)
		return nil, http.StatusRequestTimeout, err
	}

	res, err := s.repo.FindBySlugConfig(ctx, slug)
	if err != nil {
		return nil, http.StatusNotFound, errors.New("API config not found")
	}

	var headers map[string]string
	var queryParams map[string]string
	var variables map[string]dto.Variable
	var rawBody interface{}

	_ = json.Unmarshal(res.Headers, &headers)
	_ = json.Unmarshal(res.QueryParams, &queryParams)
	_ = json.Unmarshal(res.Variables, &variables)

	if err := json.Unmarshal(res.Body, &rawBody); err != nil {
		// error handling jika perlu
	}

	if bodyStr, ok := rawBody.(string); ok {
		var bodyObj map[string]interface{}
		if err := json.Unmarshal([]byte(bodyStr), &bodyObj); err == nil {
			rawBody = bodyObj
		}
	}

	body, _ := rawBody.(map[string]interface{})

	// Combine base URL with URI to get complete URL
	completeURL := res.URLConfig.URL
	if res.URI != "" {
		// Remove trailing slash from base URL and leading slash from URI to avoid double slashes
		baseURL := strings.TrimSuffix(res.URLConfig.URL, "/")
		uri := strings.TrimPrefix(res.URI, "/")
		completeURL = baseURL + "/" + uri
	}

	resp := &dto.APIConfigResponse{
		ID:           res.ID,
		Slug:         res.Slug,
		Protocol:     res.URLConfig.Protocol, // Get protocol from URLConfig
		Method:       res.Method,
		URLConfigID:  res.URLConfigID,
		URI:          res.URI,
		URL:          completeURL,
		URLConfig: dto.URLConfigResponse{
			ID:        res.URLConfig.ID,
			Nama:      res.URLConfig.Nama,
			Protocol:  res.URLConfig.Protocol,
			URL:       res.URLConfig.URL,
			Deskripsi: res.URLConfig.Deskripsi,
			IsActive:  res.URLConfig.IsActive,
			GRPCService: getStringValue(res.URLConfig.GRPCService),
					ProtoFile:   getStringValue(res.URLConfig.ProtoFile),
			TLSEnabled:  res.URLConfig.TLSEnabled,
		},
		Headers:      headers,
		QueryParams:  queryParams,
		Body:         body,
		Variables:    variables,
		MaxRetries:   res.MaxRetries,
		RetryDelay:   res.RetryDelay,
		Timeout:      res.Timeout,
		Manipulation: res.Manipulation,
		Description:  res.Description,
		IsAdmin:      res.IsAdmin,
	}

	return resp, http.StatusOK, nil
	*/
	return nil, http.StatusNotImplemented, errors.New("Config retrieval by slug is disabled")
}

// DISABLED: Config retrieval by URI function
func (s *APIConfigService) GetByURIConfig(ctx context.Context, uri, method string) (*dto.APIConfigResponse, error) {
	/*
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before getting API config by URI",
			zap.String("uri", uri),
			zap.String("method", method),
			zap.Error(err),
		)
		return nil, ctx.Err()
	}

	logger.GetLogger().Info("Service: Looking up API config by URI",
		zap.String("uri", uri),
		zap.String("method", method),
	)

	res, err := s.repo.FindByURIConfig(ctx, uri, method)
	if err != nil {
		logger.GetLogger().Debug("No API config found for URI",
			zap.String("uri", uri),
			zap.String("method", method),
			zap.Error(err),
		)
		return nil, err
	}

	var headers map[string]string
	var queryParams map[string]string
	var variables map[string]dto.Variable
	var rawBody interface{}

	_ = json.Unmarshal(res.Headers, &headers)
	_ = json.Unmarshal(res.QueryParams, &queryParams)
	_ = json.Unmarshal(res.Variables, &variables)

	if err := json.Unmarshal(res.Body, &rawBody); err != nil {
		// error handling jika perlu
	}

	if bodyStr, ok := rawBody.(string); ok {
		var bodyObj map[string]interface{}
		if err := json.Unmarshal([]byte(bodyStr), &bodyObj); err == nil {
			rawBody = bodyObj
		}
	}

	body, _ := rawBody.(map[string]interface{})

	// Combine base URL with URI to get complete URL
	completeURL := res.URLConfig.URL
	if res.URI != "" {
		// Remove trailing slash from base URL and leading slash from URI to avoid double slashes
		baseURL := strings.TrimSuffix(res.URLConfig.URL, "/")
		uri := strings.TrimPrefix(res.URI, "/")
		completeURL = baseURL + "/" + uri
	}

	resp := &dto.APIConfigResponse{
		ID:           res.ID,
		Slug:         res.Slug,
		Protocol:     res.URLConfig.Protocol, // Get protocol from URLConfig
		Method:       res.Method,
		URLConfigID:  res.URLConfigID,
		URI:          res.URI,
		URL:          completeURL,
		URLConfig: dto.URLConfigResponse{
			ID:        res.URLConfig.ID,
			Nama:      res.URLConfig.Nama,
			Protocol:  res.URLConfig.Protocol,
			URL:       res.URLConfig.URL,
			Deskripsi: res.URLConfig.Deskripsi,
			IsActive:  res.URLConfig.IsActive,
			GRPCService: getStringValue(res.URLConfig.GRPCService),
					ProtoFile:   getStringValue(res.URLConfig.ProtoFile),
			TLSEnabled:  res.URLConfig.TLSEnabled,
		},
		Headers:      headers,
		QueryParams:  queryParams,
		Body:         body,
		Variables:    variables,
		MaxRetries:   res.MaxRetries,
		RetryDelay:   res.RetryDelay,
		Timeout:      res.Timeout,
		Manipulation: res.Manipulation,
		Description:  res.Description,
		IsAdmin:      res.IsAdmin,
	}

	logger.GetLogger().Info("Service: API config found for URI",
		zap.String("slug", res.Slug),
		zap.String("uri", uri),
		zap.String("method", method),
		zap.String("protocol", resp.Protocol),
	)

	return resp, nil
	*/
	return nil, errors.New("Config retrieval by URI is disabled")
}

// DISABLED: Config retrieval by URI pattern function
func (s *APIConfigService) GetByURIConfigWithPattern(ctx context.Context, requestURI, method string) (*dto.APIConfigResponse, map[string]string, error) {
	/*
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before getting API config by URI pattern",
			zap.String("request_uri", requestURI),
			zap.String("method", method),
			zap.Error(err),
		)
		return nil, nil, ctx.Err()
	}

	logger.GetLogger().Info("Service: Looking up API config by URI pattern",
		zap.String("request_uri", requestURI),
		zap.String("method", method),
	)

	res, params, err := s.repo.FindByURIConfigWithPattern(ctx, requestURI, method)
	if err != nil {
		logger.GetLogger().Debug("No API config found for URI pattern",
			zap.String("request_uri", requestURI),
			zap.String("method", method),
			zap.Error(err),
		)
		return nil, nil, err
	}

	var headers map[string]string
	var queryParams map[string]string
	var variables map[string]dto.Variable
	var rawBody interface{}

	_ = json.Unmarshal(res.Headers, &headers)
	_ = json.Unmarshal(res.QueryParams, &queryParams)
	_ = json.Unmarshal(res.Variables, &variables)

	if err := json.Unmarshal(res.Body, &rawBody); err != nil {
		// error handling jika perlu
	}

	if bodyStr, ok := rawBody.(string); ok {
		var bodyObj map[string]interface{}
		if err := json.Unmarshal([]byte(bodyStr), &bodyObj); err == nil {
			rawBody = bodyObj
		}
	}

	body, _ := rawBody.(map[string]interface{})

	// Combine base URL with URI to get complete URL
	completeURL := res.URLConfig.URL
	if res.URI != "" {
		// Remove trailing slash from base URL and leading slash from URI to avoid double slashes
		baseURL := strings.TrimSuffix(res.URLConfig.URL, "/")
		uri := strings.TrimPrefix(res.URI, "/")
		completeURL = baseURL + "/" + uri
	}

	resp := &dto.APIConfigResponse{
		ID:           res.ID,
		Slug:         res.Slug,
		Protocol:     res.URLConfig.Protocol, // Get protocol from URLConfig
		Method:       res.Method,
		URLConfigID:  res.URLConfigID,
		URI:          res.URI,
		URL:          completeURL,
		URLConfig: dto.URLConfigResponse{
			ID:        res.URLConfig.ID,
			Nama:      res.URLConfig.Nama,
			Protocol:  res.URLConfig.Protocol,
			URL:       res.URLConfig.URL,
			Deskripsi: res.URLConfig.Deskripsi,
			IsActive:  res.URLConfig.IsActive,
			GRPCService: getStringValue(res.URLConfig.GRPCService),
					ProtoFile:   getStringValue(res.URLConfig.ProtoFile),
			TLSEnabled:  res.URLConfig.TLSEnabled,
		},
		Headers:      headers,
		QueryParams:  queryParams,
		Body:         body,
		Variables:    variables,
		MaxRetries:   res.MaxRetries,
		RetryDelay:   res.RetryDelay,
		Timeout:      res.Timeout,
		Manipulation: res.Manipulation,
		Description:  res.Description,
		IsAdmin:      res.IsAdmin,
	}

	logger.GetLogger().Info("Service: API config found for URI pattern",
		zap.String("slug", res.Slug),
		zap.String("config_uri", res.URI),
		zap.String("request_uri", requestURI),
		zap.String("method", method),
		zap.String("protocol", resp.Protocol),
		zap.Int("matched_params", len(params)),
	)

	return resp, params, nil
	*/
	return nil, nil, errors.New("Config retrieval by URI pattern is disabled")
}

// DISABLED: Group retrieval by slug function
func (s *APIConfigService) GetBySlugGroup(slug string) (*dto.APIGroupResponse, int, error) {
	/*
	res, err := s.repo.FindBySlugGroup(slug)
	if err != nil {
		return nil, http.StatusNotFound, errors.New("API Group not found")
	}

	resp := &dto.APIGroupResponse{
		ID:      res.ID,
		Slug:    res.Slug,
		Name:    res.Name,
		IsAdmin: res.IsAdmin,
	}

	return resp, http.StatusOK, nil
	*/
	return nil, http.StatusNotImplemented, errors.New("Group functions are disabled")
}

// End Get By Slug

// Start Get All
func (s *APIConfigService) GetAllConfig(params any) ([]dto.APIConfigResponse, int64, int, int, error) {
	// Extract all data from map[string]any
	paginatedData, ok := params.(map[string]any)
	if !ok {
		return nil, 0, 0, http.StatusBadRequest, errors.New("invalid pagination parameters")
	}

	// Extract core pagination parameters
	limit := int(paginatedData["limit"].(int64))
	offset := int(paginatedData["offset"].(int64))
	search, _ := paginatedData["search"].(string)

	// Extract URL config IDs from filter parameters
	var urlConfigIDs []uint
	if urlConfigID, exists := paginatedData["url_config_id"]; exists {
		if typedID, ok := urlConfigID.(uint); ok && typedID > 0 {
			urlConfigIDs = append(urlConfigIDs, typedID)
		}
	}

	// Log dynamic parameters for debugging
	logger.GetLogger().Info("Service: Processing request with dynamic parameters",
		zap.Int("limit", limit),
		zap.Int("offset", offset),
		zap.String("search", search),
		zap.Any("url_config_ids", urlConfigIDs),
		zap.Any("all_params", paginatedData),
	)

	pages, total, err := s.repo.GetAllConfig(limit, offset, search, urlConfigIDs...)
	if err != nil {
		return nil, 0, 0, http.StatusInternalServerError, err
	}
	pageTotal := int(math.Ceil(float64(total) / float64(limit)))
	var res []dto.APIConfigResponse
	for _, data := range pages {
		var headers map[string]string
		var queryParams map[string]string
		var body map[string]interface{}
		var variables map[string]dto.Variable

		_ = json.Unmarshal(data.Headers, &headers)
		_ = json.Unmarshal(data.QueryParams, &queryParams)
		_ = json.Unmarshal(data.Body, &body)
		_ = json.Unmarshal(data.Variables, &variables)

		// Combine base URL with URI to get complete URL
		completeURL := data.URLConfig.URL
		if data.URI != "" {
			// Remove trailing slash from base URL and leading slash from URI to avoid double slashes
			baseURL := strings.TrimSuffix(data.URLConfig.URL, "/")
			uri := strings.TrimPrefix(data.URI, "/")
			completeURL = baseURL + "/" + uri
		}

		res = append(res, dto.APIConfigResponse{
			ID:           data.ID,
			Slug:         data.Slug,
			Protocol:     data.URLConfig.Protocol, // Get protocol from URLConfig
			Method:       data.Method,
			URLConfigID:  data.URLConfigID,
			URI:          data.URI,
			URL:          completeURL,
			URLConfig: dto.URLConfigResponse{
				ID:        data.URLConfig.ID,
				Nama:      data.URLConfig.Nama,
				Protocol:  data.URLConfig.Protocol,
				URL:       data.URLConfig.URL,
				Deskripsi: data.URLConfig.Deskripsi,
				IsActive:  data.URLConfig.IsActive,
				GRPCService: getStringValue(data.URLConfig.GRPCService),
								ProtoFile:   getStringValue(data.URLConfig.ProtoFile),
				TLSEnabled:  data.URLConfig.TLSEnabled,
			},
			Headers:      headers,
			QueryParams:  queryParams,
			Body:         body,
			Variables:    variables,
			MaxRetries:   data.MaxRetries,
			RetryDelay:   data.RetryDelay,
			Timeout:      data.Timeout,
			Manipulation: data.Manipulation,
			Description:  data.Description,
			IsAdmin:      data.IsAdmin,
		})
	}
	return res, total, pageTotal, http.StatusOK, nil
}

// DISABLED: Group retrieval all function
func (s *APIConfigService) GetAllGroup(limit, offset int, search string) ([]dto.APIGroupResponse, int64, int, int, error) {
	/*
	pages, total, err := s.repo.GetAllGroup(limit, offset, search)
	if err != nil {
		return nil, 0, 0, http.StatusInternalServerError, err
	}
	pageTotal := int(math.Ceil(float64(total) / float64(limit)))
	var res []dto.APIGroupResponse
	for _, data := range pages {
		res = append(res, dto.APIGroupResponse{
			ID:      data.ID,
			Slug:    data.Slug,
			Name:    data.Name,
			IsAdmin: data.IsAdmin,
		})
	}
	return res, total, pageTotal, http.StatusOK, nil
	*/
	return nil, 0, 0, http.StatusNotImplemented, errors.New("Group functions are disabled")
}

// DISABLED: Group step retrieval all function
func (s *APIConfigService) GetAllGroupStep(limit, offset int, search string, groupID uint) ([]dto.APIGroupStepResponse, int64, int, int, error) {
	/*
	pages, total, err := s.repo.GetAllGroupStep(limit, offset, search, groupID)
	if err != nil {
		return nil, 0, 0, http.StatusInternalServerError, err
	}
	pageTotal := int(math.Ceil(float64(total) / float64(limit)))
	var res []dto.APIGroupStepResponse
	for _, data := range pages {

		var variables map[string]interface{}
		_ = json.Unmarshal(data.Variables, &variables)

		res = append(res, dto.APIGroupStepResponse{
			ID:          data.ID,
			GroupID:     data.GroupID,
			APIConfigID: data.APIConfigID,
			OrderIndex:  data.OrderIndex,
			Alias:       data.Alias,
			Variables:   variables,
		})
	}
	return res, total, pageTotal, http.StatusOK, nil
	*/
	return nil, 0, 0, http.StatusNotImplemented, errors.New("Group step functions are disabled")
}
// DISABLED: Group cron retrieval all function
func (s *APIConfigService) GetAllGroupCron(limit, offset int, search, slug string) ([]dto.APIGroupCronResponse, int64, int, int, error) {
	/*
	pages, total, err := s.repo.GetAllGroupCron(limit, offset, search, slug)
	if err != nil {
		return nil, 0, 0, http.StatusInternalServerError, err
	}
	pageTotal := int(math.Ceil(float64(total) / float64(limit)))
	var res []dto.APIGroupCronResponse
	for _, data := range pages {
		res = append(res, dto.APIGroupCronResponse{
			ID:       data.ID,
			Slug:     data.Slug,
			Schedule: data.Schedule,
			Enabled:  data.Enabled,
		})
	}
	return res, total, pageTotal, http.StatusOK, nil
	*/
	return nil, 0, 0, http.StatusNotImplemented, errors.New("Group cron functions are disabled")
}

// End Get All

// DISABLED: Execute group by slug function
func (s *APIConfigService) ExecuteBySlug(slug string, input map[string]interface{}) (map[string]interface{}, int, error) {
	/*
	logger.GetLogger().Info("Service: Executing API group by slug",
		zap.String("slug", slug),
		zap.Any("input", input),
	)

	group, err := s.repo.FindBySlugGroup(slug)
	if err != nil {
		logger.GetLogger().Error("Service: API group not found",
			zap.String("slug", slug),
			zap.Error(err),
		)
		return nil, http.StatusNotFound, errors.New("API group not found")
	}

	logger.GetLogger().Info("Service: API group found, executing steps",
		zap.String("slug", slug),
		zap.String("group_name", group.Name),
		zap.Int("steps_count", len(group.Steps)),
	)

	contextData := map[string]interface{}{
		"input": input,
		"steps": map[string]interface{}{},
	}

	for stepIndex, step := range group.Steps {
		logger.GetLogger().Info("Service: Executing group step",
			zap.String("slug", slug),
			zap.Int("step_index", stepIndex),
			zap.String("step_alias", step.Alias),
			zap.Uint("api_config_id", step.APIConfigID),
			zap.Int("step_order", step.OrderIndex),
		)

		config, err := s.repo.GetByIDConfig(step.APIConfigID)
		if err != nil {
			logger.GetLogger().Error("Service: Failed to get API config for step",
				zap.String("slug", slug),
				zap.Int("step_index", stepIndex),
				zap.Uint("api_config_id", step.APIConfigID),
				zap.Error(err),
			)
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to get API config: %v", err)
		}

		// Combine base URL with URI to get complete URL for this step
		completeURL := config.URLConfig.URL
		if config.URI != "" {
			// Remove trailing slash from base URL and leading slash from URI to avoid double slashes
			baseURL := strings.TrimSuffix(config.URLConfig.URL, "/")
			uri := strings.TrimPrefix(config.URI, "/")
			completeURL = baseURL + "/" + uri
		}

		// Render URI if it contains template variables
		renderedURL := completeURL
		if strings.Contains(completeURL, "{{") || strings.Contains(completeURL, "${") {
			rendered, err := integrasi.RenderTemplate(completeURL, contextData)
			if err != nil {
				logger.GetLogger().Error("Service: Failed to render URL template",
					zap.String("slug", slug),
					zap.Int("step_index", stepIndex),
					zap.String("url", completeURL),
					zap.Error(err),
				)
				return nil, http.StatusInternalServerError, fmt.Errorf("failed to render URL template: %v", err)
			}
			renderedURL = rendered
		}

		renderedVars := map[string]interface{}{}
		if len(step.Variables) > 0 {
			var rawVars map[string]interface{}
			if err := json.Unmarshal(step.Variables, &rawVars); err != nil {
				logger.GetLogger().Error("Service: Failed to parse step variables",
					zap.String("slug", slug),
					zap.Int("step_index", stepIndex),
					zap.Error(err),
				)
				return nil, http.StatusInternalServerError, fmt.Errorf("failed to parse variables: %v", err)
			}

			for key, val := range rawVars {
				rendered, err := integrasi.RenderTemplate(fmt.Sprint(val), contextData)
				if err != nil {
					logger.GetLogger().Error("Service: Failed to render template variable",
						zap.String("slug", slug),
						zap.Int("step_index", stepIndex),
						zap.String("variable_key", key),
						zap.Error(err),
					)
					return nil, http.StatusInternalServerError, fmt.Errorf("failed to render template: %v", err)
				}
				renderedVars[key] = rendered
			}
		}

		reqBody := bytes.NewBuffer(nil)
		if body, ok := renderedVars["body"]; ok {
			reqBody = bytes.NewBuffer([]byte(body.(string)))
		}

		req, err := http.NewRequest(config.Method, renderedURL, reqBody)
		if err != nil {
			logger.GetLogger().Error("Service: Failed to create HTTP request",
				zap.String("slug", slug),
				zap.Int("step_index", stepIndex),
				zap.String("method", config.Method),
				zap.String("url", renderedURL),
				zap.Error(err),
			)
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to create request: %v", err)
		}

		if headers, ok := renderedVars["headers"].(map[string]interface{}); ok {
			for k, v := range headers {
				req.Header.Set(k, fmt.Sprint(v))
			}
		}

		logger.GetLogger().Debug("Service: Making HTTP request for step",
			zap.String("slug", slug),
			zap.Int("step_index", stepIndex),
			zap.String("method", config.Method),
			zap.String("url", renderedURL),
		)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			logger.GetLogger().Error("Service: HTTP request failed for step",
				zap.String("slug", slug),
				zap.Int("step_index", stepIndex),
				zap.Error(err),
			)
			return nil, http.StatusInternalServerError, fmt.Errorf("http request failed: %v", err)
		}
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.GetLogger().Error("Service: Failed to read response for step",
				zap.String("slug", slug),
				zap.Int("step_index", stepIndex),
				zap.Error(err),
			)
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to read response: %v", err)
		}

		var parsedResp interface{}
		if err := json.Unmarshal(bodyBytes, &parsedResp); err != nil {
			logger.GetLogger().Error("Service: Failed to parse response JSON for step",
				zap.String("slug", slug),
				zap.Int("step_index", stepIndex),
				zap.Int("response_status", resp.StatusCode),
				zap.Error(err),
			)
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to parse response: %v", err)
		}

		logger.GetLogger().Info("Service: Step executed successfully",
			zap.String("slug", slug),
			zap.Int("step_index", stepIndex),
			zap.String("step_alias", step.Alias),
			zap.Int("response_status", resp.StatusCode),
			zap.Int("response_size", len(bodyBytes)),
		)

		contextData["steps"].(map[string]interface{})[step.Alias] = map[string]interface{}{
			"result": parsedResp,
			"status": resp.StatusCode,
		}
	}

	logger.GetLogger().Info("Service: API group execution completed",
		zap.String("slug", slug),
		zap.String("group_name", group.Name),
		zap.Int("total_steps", len(group.Steps)),
	)

	return contextData, http.StatusOK, nil
	*/
	return nil, http.StatusNotImplemented, errors.New("Group execution functions are disabled")
}

// URLConfig Service Functions
func (s *APIConfigService) CreateURLConfig(ctx context.Context, req dto.URLConfigRequest) (int, error) {
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before creating URL config",
			zap.String("nama", req.Nama),
			zap.Error(err),
		)
		return http.StatusRequestTimeout, err
	}

	logger.GetLogger().Info("Service: Creating URL config",
		zap.String("nama", req.Nama),
		zap.String("protocol", req.Protocol),
		zap.String("url", req.URL),
		zap.Bool("is_active", req.IsActive),
	)

	urlConfig := &model.URLConfig{
		Nama:       req.Nama,
		Protocol:   req.Protocol,
		URL:        req.URL,
		Deskripsi:  req.Deskripsi,
		IsActive:   req.IsActive,
		GRPCService: stringPtr(req.GRPCService),
				ProtoFile:   stringPtr(req.ProtoFile),
		TLSEnabled:  req.TLSEnabled,
	}

	if err := s.repo.CreateURLConfig(ctx, urlConfig); err != nil {
		logger.GetLogger().Error("Service: Failed to create URL config",
			zap.String("nama", req.Nama),
			zap.Error(err),
		)
		return http.StatusInternalServerError, err
	}

	logger.GetLogger().Info("Service: URL config created successfully",
		zap.String("nama", req.Nama),
	)

	return http.StatusCreated, nil
}

func (s *APIConfigService) GetByIDURLConfig(ctx context.Context, id uint) (*dto.URLConfigResponse, int, error) {
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before getting URL config",
			zap.Uint("url_config_id", id),
			zap.Error(err),
		)
		return nil, http.StatusRequestTimeout, err
	}

	res, err := s.repo.GetByIDURLConfig(id)
	if err != nil {
		logger.GetLogger().Warn("Service: URL config not found",
			zap.Uint("url_config_id", id),
		)
		return nil, http.StatusNotFound, errors.New("URL config not found")
	}

	resp := &dto.URLConfigResponse{
		ID:        res.ID,
		Nama:      res.Nama,
		Protocol:  res.Protocol,
		URL:       res.URL,
		Deskripsi: res.Deskripsi,
		IsActive:  res.IsActive,
		GRPCService: getStringValue(res.GRPCService),
				ProtoFile:   getStringValue(res.ProtoFile),
		TLSEnabled:  res.TLSEnabled,
	}

	logger.GetLogger().Info("Service: URL config retrieved successfully",
		zap.Uint("url_config_id", id),
		zap.String("nama", res.Nama),
	)

	return resp, http.StatusOK, nil
}

func (s *APIConfigService) GetAllURLConfig(params any) ([]dto.URLConfigResponse, int64, int, int, error) {
	// Type assertion to extract parameters from all-in-one struct
	paginatedData, ok := params.(struct {
		Page              int
		Limit             int
		Offset            int
		Search            string
		DynamicParams     map[string]string
		DynamicTypedParams map[string]any
	})
	if !ok {
		return nil, 0, 0, http.StatusBadRequest, errors.New("invalid pagination parameters")
	}

	// Get filter values
	var protocolFilter *string
	var isActiveFilter *bool

	if protocol, exists := paginatedData.DynamicParams["protocol"]; exists {
		protocolFilter = &protocol
	}

	if isActive, exists := paginatedData.DynamicTypedParams["is_active"]; exists {
		if boolVal, ok := isActive.(bool); ok {
			isActiveFilter = &boolVal
		}
	}

	urlConfigs, total, err := s.repo.GetAllURLConfig(paginatedData.Limit, paginatedData.Offset, paginatedData.Search)
	if err != nil {
		return nil, 0, 0, http.StatusInternalServerError, err
	}

	pageTotal := int(math.Ceil(float64(total) / float64(paginatedData.Limit)))
	var res []dto.URLConfigResponse

	for _, data := range urlConfigs {
		// Apply filters
		if protocolFilter != nil && data.Protocol != *protocolFilter {
			continue
		}
		if isActiveFilter != nil && data.IsActive != *isActiveFilter {
			continue
		}

		res = append(res, dto.URLConfigResponse{
			ID:        data.ID,
			Nama:      data.Nama,
			Protocol:  data.Protocol,
			URL:       data.URL,
			Deskripsi: data.Deskripsi,
			IsActive:  data.IsActive,
			GRPCService: getStringValue(data.GRPCService),
				ProtoFile:   getStringValue(data.ProtoFile),
			TLSEnabled:  data.TLSEnabled,
		})
	}

	return res, total, pageTotal, http.StatusOK, nil
}

func (s *APIConfigService) UpdateURLConfig(ctx context.Context, id uint, req dto.URLConfigRequest) (int, error) {
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before updating URL config",
			zap.Uint("url_config_id", id),
			zap.Error(err),
		)
		return http.StatusRequestTimeout, err
	}

	urlConfig := &model.URLConfig{
		Nama:       req.Nama,
		Protocol:   req.Protocol,
		URL:        req.URL,
		Deskripsi:  req.Deskripsi,
		IsActive:   req.IsActive,
		GRPCService: stringPtr(req.GRPCService),
				ProtoFile:   stringPtr(req.ProtoFile),
		TLSEnabled:  req.TLSEnabled,
	}
	urlConfig.ID = id

	if err := s.repo.UpdateURLConfig(ctx, urlConfig); err != nil {
		logger.GetLogger().Error("Service: Failed to update URL config",
			zap.Uint("url_config_id", id),
			zap.Error(err),
		)
		return http.StatusInternalServerError, err
	}

	logger.GetLogger().Info("Service: URL config updated successfully",
		zap.Uint("url_config_id", id),
		zap.String("nama", req.Nama),
	)

	return http.StatusOK, nil
}

func (s *APIConfigService) DeleteURLConfig(ctx context.Context, id uint) (int, error) {
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before deleting URL config",
			zap.Uint("url_config_id", id),
			zap.Error(err),
		)
		return http.StatusRequestTimeout, err
	}

	if err := s.repo.DeleteURLConfig(ctx, id); err != nil {
		logger.GetLogger().Error("Service: Failed to delete URL config",
			zap.Uint("url_config_id", id),
			zap.Error(err),
		)
		return http.StatusInternalServerError, err
	}

	logger.GetLogger().Info("Service: URL config deleted successfully",
		zap.Uint("url_config_id", id),
	)

	return http.StatusNoContent, nil
}
