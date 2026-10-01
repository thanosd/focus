package adapters_test

import "github.com/thanosd/focus/backend/internal/config"

func testConfig() *config.Config {
	return &config.Config{
		Environment:        "development",
		SessionExpireHours: 1,
		AuthAllowedEmails:  []string{"thanos@example.com"},
	}
}
