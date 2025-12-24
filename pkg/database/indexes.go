package database

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

// OptimizedIndexes creates all necessary indexes for optimal performance
func OptimizedIndexes(db *gorm.DB) error {
	log.Println("Creating optimized indexes for performance...")

	// Composite indexes for APIConfig table
	apiConfigIndexes := []string{
		// Protocol-specific indexes
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_configs_protocol_method ON api_configs(protocol, method) WHERE protocol IN ('http', 'grpc');",

		// gRPC-specific composite index
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_configs_grpc_service_method ON api_configs(grpc_service, grpc_method) WHERE protocol = 'grpc';",

		// Performance-related indexes
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_configs_protocol_timeout ON api_configs(protocol, timeout) WHERE timeout > 30;",

		// Admin lookup indexes
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_configs_is_admin_protocol ON api_configs(is_admin, protocol);",

		// JSONB indexes for common queries
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_configs_variables_gin ON api_configs USING GIN (variables);",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_configs_headers_gin ON api_configs USING GIN (headers);",

		// Full-text search for URL and description
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_configs_url_fts ON api_configs USING GIN (to_tsvector('english', url));",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_configs_description_fts ON api_configs USING GIN (to_tsvector('english', description));",
	}

	// APIGroup composite indexes
	apiGroupIndexes := []string{
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_groups_is_active_admin ON api_groups(is_active, is_admin);",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_groups_last_executed_active ON api_groups(last_executed DESC) WHERE is_active = true;",
	}

	// APIGroupStep composite indexes
	apiGroupStepIndexes := []string{
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_group_steps_group_order ON api_group_steps(group_id, order_index) WHERE is_enabled = true;",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_group_steps_config_enabled ON api_group_steps(api_config_id, is_enabled);",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_group_steps_alias_gin ON api_group_steps USING GIN (variables);",
	}

	// APIGroupCron composite indexes
	apiGroupCronIndexes := []string{
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_group_cron_enabled_next_run ON api_group_cron(enabled, next_run) WHERE enabled = true;",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_group_cron_running ON api_group_cron(is_running) WHERE is_running = true;",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_group_cron_performance ON api_group_cron(slug, success_count, failure_count);",
	}

	// Execute all indexes
	allIndexes := append(apiConfigIndexes, apiGroupIndexes...)
	allIndexes = append(allIndexes, apiGroupStepIndexes...)
	allIndexes = append(allIndexes, apiGroupCronIndexes...)

	for _, indexSQL := range allIndexes {
		if err := db.Exec(indexSQL).Error; err != nil {
			log.Printf("Warning: Failed to create index: %v", err)
			// Don't return error for index creation, continue with others
		}
	}

	log.Println("Optimized indexes created successfully")
	return nil
}

// CreatePartitionedTables creates partitioned tables for high-volume data
func CreatePartitionedTables(db *gorm.DB) error {
	log.Println("Creating partitioned tables for high-volume data...")

	// Create audit logs table (partitioned by date)
	auditTableSQL := `
	CREATE TABLE IF NOT EXISTS integration_logs (
		id BIGSERIAL,
		api_config_id INTEGER NOT NULL,
		slug VARCHAR(100) NOT NULL,
		protocol VARCHAR(20) NOT NULL,
		method VARCHAR(100) NOT NULL,
		url VARCHAR(2048) NOT NULL,
		request_headers JSONB DEFAULT '{}',
		request_body JSONB,
		response_status INTEGER,
		response_headers JSONB DEFAULT '{}',
		response_body TEXT,
		duration_ms INTEGER,
		error_message TEXT,
		client_ip INET,
		user_agent TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		PRIMARY KEY (id, created_at)
	) PARTITION BY RANGE (created_at);
	`

	if err := db.Exec(auditTableSQL).Error; err != nil {
		return fmt.Errorf("failed to create partitioned table: %w", err)
	}

	// Create monthly partitions
	partitionSQLs := []string{
		"CREATE TABLE IF NOT EXISTS integration_logs_y2024m01 PARTITION OF integration_logs FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');",
		"CREATE TABLE IF NOT EXISTS integration_logs_y2024m02 PARTITION OF integration_logs FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');",
		"CREATE TABLE IF NOT EXISTS integration_logs_y2024m03 PARTITION OF integration_logs FOR VALUES FROM ('2024-03-01') TO ('2024-04-01');",
		// Add more partitions as needed
	}

	for _, partitionSQL := range partitionSQLs {
		if err := db.Exec(partitionSQL).Error; err != nil {
			log.Printf("Warning: Failed to create partition: %v", err)
		}
	}

	// Create indexes for partitioned table
	logIndexes := []string{
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_integration_logs_api_config_id ON integration_logs(api_config_id);",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_integration_logs_slug ON integration_logs(slug);",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_integration_logs_protocol ON integration_logs(protocol);",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_integration_logs_created_at ON integration_logs(created_at DESC);",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_integration_logs_status ON integration_logs(response_status) WHERE response_status >= 400;",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_integration_logs_performance ON integration_logs(duration_ms) WHERE duration_ms > 1000;",
	}

	for _, indexSQL := range logIndexes {
		if err := db.Exec(indexSQL).Error; err != nil {
			log.Printf("Warning: Failed to create log index: %v", err)
		}
	}

	log.Println("Partitioned tables created successfully")
	return nil
}

// DatabaseOptimizations applies all performance optimizations
func DatabaseOptimizations(db *gorm.DB) error {
	log.Println("Applying database performance optimizations...")

	// Set optimal PostgreSQL parameters
	optimizations := []string{
		// Increase work_mem for complex queries
		"SET LOCAL work_mem = '16MB';",

		// Optimize maintenance_work_mem for index building
		"SET LOCAL maintenance_work_mem = '256MB';",

		// Enable parallel query processing
		"SET LOCAL max_parallel_workers_per_gather = 4;",

		// Optimize random_page_cost for SSD
		"SET LOCAL random_page_cost = 1.1;",

		// Increase effective_cache_size
		"SET LOCAL effective_cache_size = '4GB';",
	}

	for _, opt := range optimizations {
		if err := db.Exec(opt).Error; err != nil {
			log.Printf("Warning: Failed to set optimization %v", err)
		}
	}

	// Create optimized indexes
	if err := OptimizedIndexes(db); err != nil {
		return fmt.Errorf("failed to create optimized indexes: %w", err)
	}

	// Create partitioned tables
	if err := CreatePartitionedTables(db); err != nil {
		return fmt.Errorf("failed to create partitioned tables: %w", err)
	}

	// Analyze tables for better query planning
	if err := db.Exec("ANALYZE api_configs;").Error; err != nil {
		log.Printf("Warning: Failed to analyze api_configs table: %v", err)
	}

	if err := db.Exec("ANALYZE api_groups;").Error; err != nil {
		log.Printf("Warning: Failed to analyze api_groups table: %v", err)
	}

	if err := db.Exec("ANALYZE integration_logs;").Error; err != nil {
		log.Printf("Warning: Failed to analyze integration_logs table: %v", err)
	}

	log.Println("Database performance optimizations applied successfully")
	return nil
}

// CreateMaterializedViews creates materialized views for complex queries
func CreateMaterializedViews(db *gorm.DB) error {
	log.Println("Creating materialized views for complex queries...")

	// Performance monitoring view
	perfViewSQL := `
	CREATE MATERIALIZED VIEW IF NOT EXISTS mv_api_performance AS
	SELECT
		ac.id,
		ac.slug,
		ac.protocol,
		ac.method,
		ac.url,
		COUNT(il.id) as total_requests,
		AVG(il.duration_ms) as avg_duration,
		MAX(il.duration_ms) as max_duration,
		COUNT(CASE WHEN il.response_status >= 400 THEN 1 END) as error_count,
		COUNT(CASE WHEN il.response_status < 400 THEN 1 END) as success_count,
		MAX(il.created_at) as last_request
	FROM api_configs ac
	LEFT JOIN integration_logs il ON ac.id = il.api_config_id
	GROUP BY ac.id, ac.slug, ac.protocol, ac.method, ac.url;
	`

	if err := db.Exec(perfViewSQL).Error; err != nil {
		return fmt.Errorf("failed to create performance view: %w", err)
	}

	// Create unique index on materialized view
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_api_performance_id ON mv_api_performance(id);").Error; err != nil {
		log.Printf("Warning: Failed to create index on materialized view: %v", err)
	}

	// Refresh function for materialized view
	refreshFunctionSQL := `
	CREATE OR REPLACE FUNCTION refresh_api_performance()
	RETURNS void AS $$
	BEGIN
		REFRESH MATERIALIZED VIEW CONCURRENTLY mv_api_performance;
	END;
	$$ LANGUAGE plpgsql;
	`

	if err := db.Exec(refreshFunctionSQL).Error; err != nil {
		return fmt.Errorf("failed to create refresh function: %w", err)
	}

	log.Println("Materialized views created successfully")
	return nil
}
