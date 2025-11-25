// pkg/database/postgresql.go
package database

import (
	"sync"
	"time"

	configs "github.com/surdiana/gateway/config"
	"github.com/surdiana/gateway/internal/model"
	"github.com/surdiana/gateway/pkg/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
	"go.uber.org/zap"
)

var (
	db   *gorm.DB
	once sync.Once
)

// InitDatabase initializes the database connection
func InitDatabase(config *configs.Config) *gorm.DB {
	var err error
	once.Do(func() {
		startTime := time.Now()

		// Configure GORM logger based on environment
		var dbLogger gormLogger.Interface
		switch config.App.Environment {
		case "production":
			dbLogger = gormLogger.Default.LogMode(gormLogger.Silent)
		case "staging":
			dbLogger = gormLogger.Default.LogMode(gormLogger.Warn)
		default:
			dbLogger = gormLogger.Default.LogMode(gormLogger.Info)
		}

		// Konfigurasi koneksi database dengan optimasi performa
		gormConfig := &gorm.Config{
			Logger: dbLogger,
			// Performance optimizations
			PrepareStmt: true, // Caching prepared statements
			DisableForeignKeyConstraintWhenMigrating: true, // Temporarily disable FK constraints during migration
			DisableNestedTransaction: false, // Enable nested transactions for consistency
			NowFunc: func() time.Time {
				return time.Now().UTC() // Konsistensi timezone
			},
		}

		// Open koneksi dengan custom driver untuk optimasi
		db, err = gorm.Open(postgres.New(postgres.Config{
			DSN: config.DatabaseConnectionString(),
			// Performance optimizations
			PreferSimpleProtocol: false, // Use binary protocol
			WithoutReturning:     false, // Use RETURNING for better performance
		}), gormConfig)
		if err != nil {
			logger.GetLogger().Fatal("Failed to connect to database",
				zap.Error(err),
				zap.String("host", config.Database.Host),
				zap.Int("port", config.Database.Port),
				zap.String("database", config.Database.Name),
			)
		}

		// Konfigurasi connection pool
		sqlDB, err := db.DB()
		if err != nil {
			logger.GetLogger().Fatal("Failed to get DB instance",
				zap.Error(err),
			)
		}

		// Set ukuran pool koneksi yang dioptimasi untuk high traffic
		sqlDB.SetMaxIdleConns(25)                  // Koneksi idle minimum (increased for caching)
		sqlDB.SetMaxOpenConns(200)                 // Batas maksimum koneksi (increased for concurrency)
		sqlDB.SetConnMaxLifetime(2 * time.Hour)    // Waktu hidup koneksi (increased for stability)
		sqlDB.SetConnMaxIdleTime(15 * time.Minute) // Waktu idle maksimum (decreased for faster cleanup)

		// Test database connection
		if err := sqlDB.Ping(); err != nil {
			logger.GetLogger().Fatal("Failed to ping database",
				zap.Error(err),
			)
		}

		// Auto migrate models dengan optimasi
		migrateStart := time.Now()
		err = db.AutoMigrate(
			&model.URLConfig{},
			&model.APIConfig{},
			&model.APIGroup{},
			&model.APIGroupStep{},
			&model.APIGroupCron{},
		)

		if err != nil {
			logger.GetLogger().Fatal("Failed to auto-migrate database",
				zap.Error(err),
			)
		}

		// Apply database performance optimizations
		if err := DatabaseOptimizations(db); err != nil {
			logger.GetLogger().Error("Failed to apply database optimizations",
				zap.Error(err),
			)
			// Don't fail, just log the error
		}

		// Create materialized views for performance monitoring
		// TODO: Fix materialized views to reference correct column structure
		// if err := CreateMaterializedViews(db); err != nil {
		// 	logger.GetLogger().Error("Failed to create materialized views",
		// 		zap.Error(err),
		// 	)
		// 	// Don't fail, just log the error
		// }

		connectionTime := time.Since(startTime)
		migrateTime := time.Since(migrateStart)

		logger.GetLogger().Info("Database connected successfully with optimizations",
			zap.String("host", config.Database.Host),
			zap.Int("port", config.Database.Port),
			zap.String("database", config.Database.Name),
			zap.Duration("connection_time_ms", connectionTime),
			zap.Duration("migration_time_ms", migrateTime),
			zap.Int("max_open_conns", 200),
			zap.Int("max_idle_conns", 25),
			zap.String("connection_lifetime", "2 hours"),
			zap.String("idle_timeout", "15 minutes"),
		)
	})

	return db
}

// GetDB returns the database connection
func GetDB() *gorm.DB {
	if db == nil {
		logger.GetLogger().Fatal("Database not initialized. Call InitDatabase first.")
	}
	return db
}

// CloseDB closes the database connection
func CloseDB() error {
	if db != nil {
		sqlDB, err := db.DB()
		if err != nil {
			logger.GetLogger().Error("Failed to get database instance for closing",
				zap.Error(err),
			)
			return err
		}

		if err := sqlDB.Close(); err != nil {
			logger.GetLogger().Error("Failed to close database connection",
				zap.Error(err),
			)
			return err
		}

		logger.GetLogger().Info("Database connection closed successfully")
		return nil
	}
	return nil
}
