package service

// Extension to integrasi.go for route registry support

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Payphone-Digital/gateway/internal/dto"
	"github.com/Payphone-Digital/gateway/pkg/logger"
	"go.uber.org/zap"
)

// LoadAllActiveConfigs loads all API configs for route registry initialization
func (s *APIConfigService) LoadAllActiveConfigs(ctx context.Context) ([]*dto.APIConfigResponse, error) {
	logger.GetLogger().Info("Service: Loading all active configs for route registry")

	// Check context cancellation
	if err := ctx.Err(); err != nil {
		logger.GetLogger().Warn("Service: Context cancelled before loading configs",
			zap.Error(err),
		)
		return nil, err
	}

	// Get all configs from repository
	models, err := s.repo.LoadAllActiveConfigs(ctx)
	if err != nil {
		logger.GetLogger().Error("Service: Failed to load all active configs",
			zap.Error(err),
		)
		return nil, err
	}

	// Convert to DTO
	configs := make([]*dto.APIConfigResponse, 0, len(models))
	for _, res := range models {
		var headers map[string]string
		var queryParams map[string]string
		var variables map[string]dto.Variable
		var rawBody interface{}

		_ = json.Unmarshal(res.Headers, &headers)
		_ = json.Unmarshal(res.QueryParams, &queryParams)
		_ = json.Unmarshal(res.Variables, &variables)

		if err := json.Unmarshal(res.Body, &rawBody); err != nil {
			// error handling if needed
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
			baseURL := strings.TrimSuffix(res.URLConfig.URL, "/")
			uri := strings.TrimPrefix(res.URI, "/")
			completeURL = baseURL + "/" + uri
		}

		config := &dto.APIConfigResponse{
			ID:          res.ID,
			Path:        res.Path,
			Protocol:    res.URLConfig.Protocol,
			Method:      res.Method,
			URLConfigID: res.URLConfigID,
			URI:         res.URI,
			URL:         completeURL,
			URLConfig: dto.URLConfigResponse{
				ID:          res.URLConfig.ID,
				Nama:        res.URLConfig.Nama,
				Protocol:    res.URLConfig.Protocol,
				URL:         res.URLConfig.URL,
				Deskripsi:   res.URLConfig.Deskripsi,
				IsActive:    res.URLConfig.IsActive,
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

		configs = append(configs, config)
	}

	logger.GetLogger().Info("Service: All active configs loaded successfully",
		zap.Int("count", len(configs)),
	)

	return configs, nil
}
