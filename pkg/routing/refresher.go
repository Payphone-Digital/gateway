package routing

import (
	"context"
	"errors"
	"fmt"

	"github.com/Payphone-Digital/gateway/internal/service"
	"go.uber.org/zap"
)

// Refresher handles dynamic route updates
type Refresher struct {
	registry          *RouteRegistry
	apiConfigService  *service.APIConfigService
	logger            *zap.Logger
}

// NewRefresher creates a new route refresher
func NewRefresher(
	registry *RouteRegistry,
	apiConfigService *service.APIConfigService,
	logger *zap.Logger,
) *Refresher {
	return &Refresher{
		registry:         registry,
		apiConfigService: apiConfigService,
		logger:           logger,
	}
}

// Refresh reloads all routes from the database
func (r *Refresher) Refresh(ctx context.Context) error {
	r.logger.Info("Starting route registry refresh")

	// Load all active configs from database
	configs, err := r.apiConfigService.LoadAllActiveConfigs(ctx)
	if err != nil {
		r.logger.Error("Failed to load configs from database", zap.Error(err))
		return fmt.Errorf("failed to load configs: %w", err)
	}

	if len(configs) == 0 {
		r.logger.Warn("No active configs found in database")
		return errors.New("no active configs found")
	}

	r.logger.Info("Loaded configs from database",
		zap.Int("count", len(configs)),
	)

	// Clear existing registry
	r.registry.Clear()

	// Add all configs
	successCount := 0
	errorCount := 0

	for _, config := range configs {
		if err := r.registry.AddRoute(config); err != nil {
			r.logger.Error("Failed to add route to registry",
				zap.String("slug", config.Path),
				zap.Error(err),
			)
			errorCount++
			continue
		}
		successCount++
	}

	r.logger.Info("Route registry refresh completed",
		zap.Int("total", len(configs)),
		zap.Int("success", successCount),
		zap.Int("errors", errorCount),
	)

	if errorCount > 0 {
		return fmt.Errorf("refresh completed with %d errors", errorCount)
	}

	return nil
}

// RefreshSingle updates a single route in the registry
func (r *Refresher) RefreshSingle(ctx context.Context, path, method string) error {
	r.logger.Info("Refreshing single route", zap.String("path", path), zap.String("method", method))

	// Get config from database using path+method to find correct config
	config, _, err := r.apiConfigService.GetByPathAndMethodConfig(ctx, path, method)
	if err != nil {
		r.logger.Error("Failed to get config from database",
			zap.String("path", path),
			zap.String("method", method),
			zap.Error(err),
		)
		return fmt.Errorf("failed to get config: %w", err)
	}

	// Remove old version if exists
	_ = r.registry.RemoveRoute(path, method) // Ignore error if doesn't exist

	// Add new version
	if err := r.registry.AddRoute(config); err != nil {
		r.logger.Error("Failed to add route to registry",
			zap.String("path", path),
			zap.String("method", method),
			zap.Error(err),
		)
		return fmt.Errorf("failed to add route: %w", err)
	}

	r.logger.Info("Route refreshed successfully", zap.String("path", path), zap.String("method", method))
	return nil
}

// InvalidateRoute removes a route from the registry
func (r *Refresher) InvalidateRoute(path, method string) error {
	r.logger.Info("Invalidating route", zap.String("path", path), zap.String("method", method))

	if err := r.registry.RemoveRoute(path, method); err != nil {
		r.logger.Error("Failed to remove route",
			zap.String("path", path),
			zap.String("method", method),
			zap.Error(err),
		)
		return err
	}

	r.logger.Info("Route invalidated successfully", zap.String("path", path), zap.String("method", method))
	return nil
}
