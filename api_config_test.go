package main

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	testCases := []struct {
		name       string
		setup      func(t *testing.T, dbMock sqlmock.Sqlmock, redisMock redismock.ClientMock)
		expectExit bool
	}{
		{
			name: "Success",
			setup: func(t *testing.T, dbMock sqlmock.Sqlmock, redisMock redismock.ClientMock) {
				t.Setenv("DB_URL", "postgres://user:password@localhost:5432/testdb")
				dbMock.ExpectPing()
				t.Setenv("REDIS_URL", "redis://localhost:6379")
				redisMock.ExpectPing().SetVal("PONG")
				t.Setenv("GMP_KEY", "test_gmp_key")
				t.Setenv("GMP_GEOCODE_URL", "http://localhost/geocode")
				t.Setenv("GMP_WEATHER_URL", "http://localhost/weather")
				t.Setenv("OWM_WEATHER_URL", "http://localhost/weather")
				t.Setenv("OMETEO_WEATHER_URL", "http://localhost/weather")
				t.Setenv("OWM_KEY", "test_owm_key")
			},
			expectExit: false,
		},
		{
			name: "Failure - Missing DB_URL",
			setup: func(t *testing.T, dbMock sqlmock.Sqlmock, redisMock redismock.ClientMock) {
				t.Setenv("DB_URL", "")
			},
			expectExit: true,
		},
		{
			name: "Failure - Missing REDIS_URL",
			setup: func(t *testing.T, dbMock sqlmock.Sqlmock, redisMock redismock.ClientMock) {
				t.Setenv("DB_URL", "postgres://user:password@localhost:5432/testdb")
				dbMock.ExpectPing()
				t.Setenv("REDIS_URL", "")
			},
			expectExit: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// --- Setup ---
			originalOsExit := osExit
			defer func() {
				osExit = originalOsExit
			}()

			var exitCalled bool
			osExit = func(code int) {
				exitCalled = true
				if tc.expectExit {
					assert.True(t, exitCalled, "Expected os.Exit to be called")
				} else {
					assert.False(t, exitCalled, "Expected os.Exit not to be called")
				}
				t.SkipNow()
			}

			db, dbMock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer db.Close()

			redisClient, redisMock := redismock.NewClientMock()

			// Override functions to return mocks
			sqlOpen = func(driverName, dataSourceName string) (*sql.DB, error) { return db, nil }
			redisNewClient = func(opt *redis.Options) *redis.Client { return redisClient }

			// Run test-specific setup
			if tc.setup != nil {
				tc.setup(t, dbMock, redisMock)
			}

			// --- Execute ---
			config()

			// Verify that all mock expectations were met
			if err := dbMock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
			if err := redisMock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
