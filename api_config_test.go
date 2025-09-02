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
			name: "Success - No Optional Vars",
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
			name: "Success - Dev Mode True",
			setup: func(t *testing.T, dbMock sqlmock.Sqlmock, redisMock redismock.ClientMock) {
				t.Setenv("DEV_MODE", "true")
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
			name: "Success - Dev Mode Invalid",
			setup: func(t *testing.T, dbMock sqlmock.Sqlmock, redisMock redismock.ClientMock) {
				t.Setenv("DEV_MODE", "not_a_bool")
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
			name: "Success - All Optional Vars",
			setup: func(t *testing.T, dbMock sqlmock.Sqlmock, redisMock redismock.ClientMock) {
				t.Setenv("DEV_MODE", "false")
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
				t.Setenv("CURRENT_INTERVAL_MIN", "15")
				t.Setenv("HOURLY_INTERVAL_MIN", "120")
				t.Setenv("DAILY_INTERVAL_MIN", "1440")
				t.Setenv("PORT", "9090")
			},
			expectExit: false,
		},
		{
			name: "Success - Optional Vars Invalid/Empty",
			setup: func(t *testing.T, dbMock sqlmock.Sqlmock, redisMock redismock.ClientMock) {
				t.Setenv("DEV_MODE", "false")
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
				t.Setenv("CURRENT_INTERVAL_MIN", "not_an_int")
				t.Setenv("HOURLY_INTERVAL_MIN", "also_not_an_int")
				t.Setenv("DAILY_INTERVAL_MIN", "")
				t.Setenv("PORT", "")
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
			name: "Failure - sql.Open Error",
			setup: func(t *testing.T, dbMock sqlmock.Sqlmock, redisMock redismock.ClientMock) {
				t.Setenv("DB_URL", "invalid_db_url")
				// Override sqlOpen to simulate an error
				sqlOpen = func(driverName, dataSourceName string) (*sql.DB, error) {
					return nil, sql.ErrConnDone
				}
			},
			expectExit: true,
		},
		{
			name: "Failure - DB Ping Error",
			setup: func(t *testing.T, dbMock sqlmock.Sqlmock, redisMock redismock.ClientMock) {
				t.Setenv("DB_URL", "postgres://user:password@localhost:5432/testdb")
				dbMock.ExpectPing().WillReturnError(sql.ErrConnDone)
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
		{
			name: "Failure - redis.ParseURL Error",
			setup: func(t *testing.T, dbMock sqlmock.Sqlmock, redisMock redismock.ClientMock) {
				t.Setenv("DB_URL", "postgres://user:password@localhost:5432/testdb")
				dbMock.ExpectPing()
				t.Setenv("REDIS_URL", "invalid_redis_url")
			},
			expectExit: true,
		},
		{
			name: "Failure - Redis Ping Error",
			setup: func(t *testing.T, dbMock sqlmock.Sqlmock, redisMock redismock.ClientMock) {
				t.Setenv("DB_URL", "postgres://user:password@localhost:5432/testdb")
				dbMock.ExpectPing()
				t.Setenv("REDIS_URL", "redis://localhost:6379")
				redisMock.ExpectPing().SetErr(redis.Nil)
			},
			expectExit: true,
		},
		{
			name: "Failure - Missing GMP_KEY",
			setup: func(t *testing.T, dbMock sqlmock.Sqlmock, redisMock redismock.ClientMock) {
				t.Setenv("DB_URL", "postgres://user:password@localhost:5432/testdb")
				dbMock.ExpectPing()
				t.Setenv("REDIS_URL", "redis://localhost:6379")
				redisMock.ExpectPing().SetVal("PONG")
				t.Setenv("GMP_KEY", "")
			},
			expectExit: true,
		},
		{
			name: "Failure - Missing GMP_GEOCODE_URL",
			setup: func(t *testing.T, dbMock sqlmock.Sqlmock, redisMock redismock.ClientMock) {
				t.Setenv("DB_URL", "postgres://user:password@localhost:5432/testdb")
				dbMock.ExpectPing()
				t.Setenv("REDIS_URL", "redis://localhost:6379")
				redisMock.ExpectPing().SetVal("PONG")
				t.Setenv("GMP_KEY", "test_gmp_key")
				t.Setenv("GMP_GEOCODE_URL", "")
			},
			expectExit: true,
		},
		{
			name: "Failure - Missing GMP_WEATHER_URL",
			setup: func(t *testing.T, dbMock sqlmock.Sqlmock, redisMock redismock.ClientMock) {
				t.Setenv("DB_URL", "postgres://user:password@localhost:5432/testdb")
				dbMock.ExpectPing()
				t.Setenv("REDIS_URL", "redis://localhost:6379")
				redisMock.ExpectPing().SetVal("PONG")
				t.Setenv("GMP_KEY", "test_gmp_key")
				t.Setenv("GMP_GEOCODE_URL", "http://localhost/geocode")
				t.Setenv("GMP_WEATHER_URL", "")
			},
			expectExit: true,
		},
		{
			name: "Failure - Missing OWM_WEATHER_URL",
			setup: func(t *testing.T, dbMock sqlmock.Sqlmock, redisMock redismock.ClientMock) {
				t.Setenv("DB_URL", "postgres://user:password@localhost:5432/testdb")
				dbMock.ExpectPing()
				t.Setenv("REDIS_URL", "redis://localhost:6379")
				redisMock.ExpectPing().SetVal("PONG")
				t.Setenv("GMP_KEY", "test_gmp_key")
				t.Setenv("GMP_GEOCODE_URL", "http://localhost/geocode")
				t.Setenv("GMP_WEATHER_URL", "http://localhost/weather")
				t.Setenv("OWM_WEATHER_URL", "")
			},
			expectExit: true,
		},
		{
			name: "Failure - Missing OMETEO_WEATHER_URL",
			setup: func(t *testing.T, dbMock sqlmock.Sqlmock, redisMock redismock.ClientMock) {
				t.Setenv("DB_URL", "postgres://user:password@localhost:5432/testdb")
				dbMock.ExpectPing()
				t.Setenv("REDIS_URL", "redis://localhost:6379")
				redisMock.ExpectPing().SetVal("PONG")
				t.Setenv("GMP_KEY", "test_gmp_key")
				t.Setenv("GMP_GEOCODE_URL", "http://localhost/geocode")
				t.Setenv("GMP_WEATHER_URL", "http://localhost/weather")
				t.Setenv("OWM_WEATHER_URL", "http://localhost/weather")
				t.Setenv("OMETEO_WEATHER_URL", "")
			},
			expectExit: true,
		},
		{
			name: "Failure - Missing OWM_KEY",
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
				t.Setenv("OWM_KEY", "")
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
					assert.True(t, exitCalled, "expected os.Exit to be called")
				} else {
					assert.False(t, exitCalled, "expected os.Exit not to be called")
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
