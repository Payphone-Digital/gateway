package logger

import (
	"time"

	"go.uber.org/zap/zapcore"
)

// Contoh penggunaan logger dynamic yang fleksibel

// Contoh 1: Logging request handler
func ExampleHandlerLogging() {
	// Logging request masuk
	Info("Processing user request").
		Method("GET").
		Path("/api/users/123").
		ClientIP("192.168.1.1").
		UserAgent("Mozilla/5.0...").
		UserID(123).
		Module("handler").
		Function("GetUser").
		Log()

	// Logging dengan fields tambahan
	Info("Database query executed").
		String("query", "SELECT * FROM users WHERE id = $1").
		Duration(150 * time.Millisecond).
		Int("rows_affected", 1).
		Module("repository").
		Function("FindByID").
		Log()

	// Logging error dengan context lengkap
	Error("Failed to process request").
		String("error_code", "VALIDATION_ERROR").
		Fields(map[string]interface{}{
			"invalid_fields": []string{"email", "phone"},
			"request_size":   1024,
		}).
		ClientIP("192.168.1.1").
		Module("validator").
		Function("ValidateUser").
		Log()
}

// Contoh 2: Logging service layer dengan performance tracking
func ExampleServiceLogging() {
	start := time.Now()

	// Service logic...

	// Logging dengan duration yang dihitung
	Info("Service operation completed").
		Function("ProcessPayment").
		UserID(456).
		String("payment_id", "pay_123456").
		Duration(time.Since(start)).
		Bool("success", true).
		Float64("amount", 99.99).
		Module("service").
		Log()
}

// Contoh 3: Logging dengan conditional fields
func ExampleConditionalLogging() {
	isDebug := true
	isError := false

	builder := Info("Processing batch operation").
		Module("processor").
		Function("BatchProcess")

	if isDebug {
		builder = builder.String("debug_info", "detailed_trace").
			Fields(map[string]interface{}{
				"memory_usage": "45MB",
				"goroutines":   12,
			})
	}

	if isError {
		builder = builder.Level(zapcore.ErrorLevel).
			String("error_type", "TIMEOUT").
			Duration(30 * time.Second)
	}

	builder.Log()
}

// Contoh 4: Logging untuk background jobs/cron
func ExampleBackgroundJobLogging() {
	Info("Background job started").
		String("job_type", "email_sender").
		String("job_id", "job_789").
		Module("scheduler").
		Function("ExecuteJob").
		Fields(map[string]interface{}{
			"total_emails":     500,
			"batch_size":       50,
			"scheduled_at":     time.Now().Add(-1 * time.Hour),
			"retry_count":      0,
			"priority":         "high",
		}).
		Log()

	// Logging progress
	for i := 0; i < 5; i++ {
		Info("Processing batch").
			String("job_id", "job_789").
			Int("batch_number", i+1).
			Int("total_batches", 5).
			Int("processed_emails", (i+1)*50).
			Module("scheduler").
			Function("ExecuteJob").
			Log()
		time.Sleep(100 * time.Millisecond) // Simulate work
	}

	Info("Background job completed").
		String("job_id", "job_789").
		Int("total_processed", 500).
		Int("total_failed", 2).
		Duration(5 * time.Second).
		Module("scheduler").
		Function("ExecuteJob").
		Log()
}

// Contoh 5: Logging untuk integrasi API
func ExampleIntegrationLogging() {
	// Request ke external API
	Info("Making external API request").
		String("api_name", "payment_gateway").
		Method("POST").
		String("url", "https://api.payment.com/v1/charges").
		String("request_id", "req_abc123").
		Module("integration").
		Function("ChargePayment").
		Fields(map[string]interface{}{
			"amount":       99.99,
			"currency":     "USD",
			"payment_method": "credit_card",
		}).
		Log()

	// Response dari external API
	Info("External API response received").
		String("api_name", "payment_gateway").
		String("request_id", "req_abc123").
		StatusCode(200).
		Duration(250 * time.Millisecond).
		Module("integration").
		Function("ChargePayment").
		Fields(map[string]interface{}{
			"transaction_id": "txn_xyz789",
			"status":         "succeeded",
			"fee":           2.99,
		}).
		Log()
}

// Contoh 6: Logging untuk security events
func ExampleSecurityLogging() {
	// Failed login attempt
	Warn("Failed login attempt").
		ClientIP("192.168.1.100").
		UserAgent("Mozilla/5.0...").
		String("username", "admin").
		String("reason", "invalid_password").
		Module("security").
		Function("Authenticate").
		Fields(map[string]interface{}{
			"attempt_count": 3,
			"locked_until":  time.Now().Add(15 * time.Minute),
			"ip_risk_score": 0.7,
		}).
		Log()

	// Suspicious activity
	Error("Suspicious activity detected").
		ClientIP("10.0.0.1").
		UserAgent("curl/7.68.0").
		String("pattern", "sql_injection_attempt").
		String("endpoint", "/api/users").
		Module("security").
		Function("SecurityMonitor").
		Fields(map[string]interface{}{
			"blocked":        true,
			"threat_level":   "high",
			"additional_info": "Multiple rapid requests with suspicious payloads",
		}).
		Log()
}