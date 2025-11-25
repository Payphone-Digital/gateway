package database

import (
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

// QueryOptimization represents optimization strategies for different query patterns
type QueryOptimization struct {
	Description string
	SQL         string
	Benefit     string
}

// GetRecommendedOptimizations returns recommended optimizations for common queries
func GetRecommendedOptimizations() []QueryOptimization {
	return []QueryOptimization{
		{
			Description: "Optimize slug lookups with B-tree index",
			SQL:         "EXPLAIN (ANALYZE, BUFFERS) SELECT * FROM api_configs WHERE slug = 'test-api';",
			Benefit:     "Reduces query time from O(n) to O(log n)",
		},
		{
			Description: "Optimize protocol filtering",
			SQL:         "EXPLAIN (ANALYZE, BUFFERS) SELECT * FROM api_configs WHERE protocol = 'grpc' AND is_admin = false;",
			Benefit:     "Uses composite index for efficient filtering",
		},
		{
			Description: "Optimize gRPC service/method lookups",
			SQL:         "EXPLAIN (ANALYZE, BUFFERS) SELECT * FROM api_configs WHERE grpc_service = 'user.UserService' AND grpc_method = 'GetUser';",
			Benefit:     "Composite index enables fast gRPC lookups",
		},
		{
			Description: "Optimize JSONB variable queries",
			SQL:         "EXPLAIN (ANALYZE, BUFFERS) SELECT * FROM api_configs WHERE variables ? 'user_id';",
			Benefit:     "GIN index enables fast JSONB key existence checks",
		},
		{
			Description: "Optimize group step sequential execution",
			SQL:         "EXPLAIN (ANALYZE, BUFFERS) SELECT * FROM api_group_steps WHERE group_id = 1 ORDER BY order_index;",
			Benefit:     "Composite index ensures ordered retrieval without sorting",
		},
		{
			Description: "Optimize cron job scheduling",
			SQL:         "EXPLAIN (ANALYZE, BUFFERS) SELECT * FROM api_group_cron WHERE enabled = true AND next_run <= NOW();",
			Benefit:     "Composite index enables efficient scheduling queries",
		},
	}
}

// BenchmarkQueries runs performance benchmarks on critical queries
func BenchmarkQueries(db *gorm.DB) error {
	log.Println("Running database query benchmarks...")

	benchmarks := []struct {
		name     string
		query    string
		expected time.Duration
	}{
		{
			name:     "API Config Slug Lookup",
			query:    "SELECT * FROM api_configs WHERE slug = $1;",
			expected: 10 * time.Millisecond,
		},
		{
			name:     "gRPC Service Method Lookup",
			query:    "SELECT * FROM api_configs WHERE grpc_service = $1 AND grpc_method = $2;",
			expected: 15 * time.Millisecond,
		},
		{
			name:     "Group Steps Sequential Query",
			query:    "SELECT * FROM api_group_steps WHERE group_id = $1 ORDER BY order_index;",
			expected: 20 * time.Millisecond,
		},
		{
			name:     "JSONB Variable Query",
			query:    "SELECT * FROM api_configs WHERE variables ? $1;",
			expected: 25 * time.Millisecond,
		},
		{
			name:     "Cron Job Scheduling Query",
			query:    "SELECT * FROM api_group_cron WHERE enabled = true AND next_run <= NOW();",
			expected: 30 * time.Millisecond,
		},
	}

	for _, benchmark := range benchmarks {
		start := time.Now()

		// Execute query with EXPLAIN ANALYZE to get actual execution time
		var result []map[string]interface{}
		err := db.Raw("EXPLAIN (ANALYZE, BUFFERS) "+benchmark.query, "test").Scan(&result).Error
		if err != nil {
			log.Printf("Warning: Benchmark query failed for %s: %v", benchmark.name, err)
			continue
		}

		duration := time.Since(start)

		if duration > benchmark.expected {
			log.Printf("⚠️  Slow query detected: %s took %v (expected: < %v)",
				benchmark.name, duration, benchmark.expected)
		} else {
			log.Printf("✅ Query performance OK: %s took %v", benchmark.name, duration)
		}
	}

	return nil
}

// OptimizeSlowQueries identifies and optimizes slow queries
func OptimizeSlowQueries(db *gorm.DB) error {
	log.Println("Analyzing slow queries...")

	// Get query statistics from pg_stat_statements
	var slowQueries []struct {
		Query     string
		Calls     int64
		TotalTime float64
		MeanTime  float64
		Rows      int64
	}

	err := db.Raw(`
		SELECT
			query,
			calls,
			total_time,
			mean_time,
			rows
		FROM pg_stat_statements
		WHERE mean_time > 100 -- queries taking more than 100ms on average
		ORDER BY mean_time DESC
		LIMIT 10;
	`).Scan(&slowQueries).Error

	if err != nil {
		log.Printf("Warning: Could not analyze slow queries: %v", err)
		return nil // Don't fail, just log
	}

	if len(slowQueries) == 0 {
		log.Println("✅ No slow queries detected")
		return nil
	}

	log.Printf("Found %d slow queries:", len(slowQueries))
	for i, sq := range slowQueries {
		log.Printf("%d. Query: %.100s...", i+1, sq.Query)
		log.Printf("   Calls: %d, Avg Time: %.2fms, Total Time: %.2fms",
			sq.Calls, sq.MeanTime, sq.TotalTime)

		// Suggest optimizations based on query patterns
		if contains(sq.Query, "api_configs") && contains(sq.Query, "WHERE") {
			log.Printf("   💡 Suggestion: Consider adding index for filter conditions")
		}
		if contains(sq.Query, "ORDER BY") && !contains(sq.Query, "INDEX") {
			log.Printf("   💡 Suggestion: Consider adding index for ORDER BY columns")
		}
		if contains(sq.Query, "jsonb") && contains(sq.Query, "?") {
			log.Printf("   💡 Suggestion: Consider GIN index for JSONB queries")
		}
	}

	return nil
}

// AnalyzeTableStatistics provides detailed table analysis
func AnalyzeTableStatistics(db *gorm.DB) error {
	log.Println("Analyzing table statistics...")

	tables := []string{"api_configs", "api_groups", "api_group_steps", "api_group_cron"}

	for _, table := range tables {
		var stats struct {
			TableSize    string
			IndexSize    string
			TotalSize    string
			RowEstimate  int64
			LastAnalyzed *time.Time
		}

		err := db.Raw(`
			SELECT
				pg_size_pretty(pg_total_relation_size(?)) as table_size,
				pg_size_pretty(pg_indexes_size(?)) as index_size,
				pg_size_pretty(pg_total_relation_size(?)) as total_size,
				s.n_tup_ins + s.n_tup_upd + s.n_tup_del as row_estimate,
				s.last_vacuum as last_analyzed
			FROM pg_stat_user_tables s
			WHERE s.relname = ?;
		`, table, table, table, table).Scan(&stats).Error

		if err != nil {
			log.Printf("Warning: Could not analyze table %s: %v", table, err)
			continue
		}

		log.Printf("📊 Table: %s", table)
		log.Printf("   Size: %s, Indexes: %s, Total: %s",
			stats.TableSize, stats.IndexSize, stats.TotalSize)
		log.Printf("   Estimated Rows: %d", stats.RowEstimate)
		if stats.LastAnalyzed != nil {
			log.Printf("   Last Analyzed: %v", stats.LastAnalyzed)
		}

		// Check if table needs vacuum
		if stats.LastAnalyzed == nil || time.Since(*stats.LastAnalyzed) > 24*time.Hour {
			log.Printf("   💡 Suggestion: Table needs VACUUM ANALYZE")
		}
	}

	return nil
}

// CheckIndexUsage analyzes index efficiency
func CheckIndexUsage(db *gorm.DB) error {
	log.Println("Analyzing index usage...")

	var indexUsage []struct {
		TableName  string
		IndexName  string
		IndexScan  int64
		SeqScan    int64
		UsageRatio float64
	}

	err := db.Raw(`
		SELECT
			schemaname || '.' || tablename as table_name,
			indexname,
			idx_scan as index_scan,
			seq_scan,
			CASE
				WHEN (seq_scan + idx_scan) > 0
				THEN idx_scan::float / (seq_scan + idx_scan)::float
				ELSE 0
			END as usage_ratio
		FROM pg_stat_user_indexes
		WHERE schemaname = current_schema()
		ORDER BY usage_ratio ASC;
	`).Scan(&indexUsage).Error

	if err != nil {
		log.Printf("Warning: Could not analyze index usage: %v", err)
		return nil
	}

	log.Printf("Found %d indexes:", len(indexUsage))
	unusedIndexes := 0

	for _, iu := range indexUsage {
		if iu.UsageRatio < 0.1 && iu.IndexScan < 100 {
			log.Printf("⚠️  Possibly unused index: %s on %s (usage: %.2f%%)",
				iu.IndexName, iu.TableName, iu.UsageRatio*100)
			unusedIndexes++
		} else {
			log.Printf("✅ Index in use: %s on %s (usage: %.2f%%)",
				iu.IndexName, iu.TableName, iu.UsageRatio*100)
		}
	}

	if unusedIndexes > 0 {
		log.Printf("💡 Found %d potentially unused indexes that could be dropped", unusedIndexes)
	}

	return nil
}

// GeneratePerformanceReport creates a comprehensive performance report
func GeneratePerformanceReport(db *gorm.DB) error {
	log.Println("Generating comprehensive performance report...")

	// Table statistics
	if err := AnalyzeTableStatistics(db); err != nil {
		return fmt.Errorf("failed to analyze table statistics: %w", err)
	}

	// Index usage
	if err := CheckIndexUsage(db); err != nil {
		return fmt.Errorf("failed to check index usage: %w", err)
	}

	// Slow queries
	if err := OptimizeSlowQueries(db); err != nil {
		return fmt.Errorf("failed to optimize slow queries: %w", err)
	}

	// Benchmark queries
	if err := BenchmarkQueries(db); err != nil {
		return fmt.Errorf("failed to benchmark queries: %w", err)
	}

	// Database connection stats
	var connStats struct {
		ActiveConnections int
		IdleConnections   int
		TotalConnections  int
		MaxConnections    int
	}

	err := db.Raw(`
		SELECT
			count(*) as active_connections,
			COUNT(*) FILTER (WHERE state = 'idle') as idle_connections,
			(SELECT setting::int FROM pg_settings WHERE name = 'max_connections') as max_connections
		FROM pg_stat_activity
		WHERE datname = current_database();
	`).Scan(&connStats).Error

	if err == nil {
		connStats.TotalConnections = connStats.ActiveConnections
		log.Printf("🔗 Connection Stats: Active: %d, Idle: %d, Max: %d",
			connStats.ActiveConnections, connStats.IdleConnections, connStats.MaxConnections)

		usagePercent := float64(connStats.ActiveConnections) / float64(connStats.MaxConnections) * 100
		if usagePercent > 80 {
			log.Printf("⚠️  High connection usage: %.1f%%", usagePercent)
		}
	}

	log.Println("✅ Performance report generated successfully")
	return nil
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
			 s[len(s)-len(substr):] == substr ||
			 findSubstring(s, substr))))
}

// Helper function to find substring
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}