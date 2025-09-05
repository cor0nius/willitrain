package main

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAPIConfig(t *testing.T) {
	testCases := []struct {
		name      string
		setup     func(t *testing.T)
		expectErr bool
	}{
		{
			name: "Success - No Optional Vars",
			setup: func(t *testing.T) {
				t.Setenv("DB_URL", "postgres://user:password@localhost:5432/testdb")
				t.Setenv("REDIS_URL", "redis://localhost:6379")
				t.Setenv("GMP_KEY", "test_gmp_key")
				t.Setenv("GMP_GEOCODE_URL", "http://localhost/geocode")
				t.Setenv("GMP_WEATHER_URL", "http://localhost/weather")
				t.Setenv("OWM_WEATHER_URL", "http://localhost/weather")
				t.Setenv("OMETEO_WEATHER_URL", "http://localhost/weather")
				t.Setenv("OWM_KEY", "test_owm_key")
			},
			expectErr: false,
		},
		{
			name: "Success - Dev Mode True",
			setup: func(t *testing.T) {
				t.Setenv("DEV_MODE", "true")
				t.Setenv("DB_URL", "postgres://user:password@localhost:5432/testdb")
				t.Setenv("REDIS_URL", "redis://localhost:6379")
				t.Setenv("GMP_KEY", "test_gmp_key")
				t.Setenv("GMP_GEOCODE_URL", "http://localhost/geocode")
				t.Setenv("GMP_WEATHER_URL", "http://localhost/weather")
				t.Setenv("OWM_WEATHER_URL", "http://localhost/weather")
				t.Setenv("OMETEO_WEATHER_URL", "http://localhost/weather")
				t.Setenv("OWM_KEY", "test_owm_key")
			},
			expectErr: false,
		},
		{
			name: "Success - Dev Mode Invalid",
			setup: func(t *testing.T) {
				t.Setenv("DEV_MODE", "not_a_bool")
				t.Setenv("DB_URL", "postgres://user:password@localhost:5432/testdb")
				t.Setenv("REDIS_URL", "redis://localhost:6379")
				t.Setenv("GMP_KEY", "test_gmp_key")
				t.Setenv("GMP_GEOCODE_URL", "http://localhost/geocode")
				t.Setenv("GMP_WEATHER_URL", "http://localhost/weather")
				t.Setenv("OWM_WEATHER_URL", "http://localhost/weather")
				t.Setenv("OMETEO_WEATHER_URL", "http://localhost/weather")
				t.Setenv("OWM_KEY", "test_owm_key")
			},
			expectErr: false,
		},
		{
			name: "Success - All Optional Vars",
			setup: func(t *testing.T) {
				t.Setenv("DEV_MODE", "false")
				t.Setenv("DB_URL", "postgres://user:password@localhost:5432/testdb")
				t.Setenv("REDIS_URL", "redis://localhost:6379")
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
			expectErr: false,
		},
		{
			name: "Success - Optional Vars Invalid/Empty",
			setup: func(t *testing.T) {
				t.Setenv("DB_URL", "postgres://user:password@localhost:5432/testdb")
				t.Setenv("REDIS_URL", "redis://localhost:6379")
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
			expectErr: false,
		},
		{
			name: "Failure - Missing DB_URL",
			setup: func(t *testing.T) {
				t.Setenv("DB_URL", "")
			},
			expectErr: true,
		},
		{
			name: "Failure - Missing REDIS_URL",
			setup: func(t *testing.T) {
				t.Setenv("DB_URL", "postgres://user:password@localhost:5432/testdb")
				t.Setenv("REDIS_URL", "")
			},
			expectErr: true,
		},
		{
			name: "Failure - Missing GMP_KEY",
			setup: func(t *testing.T) {
				t.Setenv("DB_URL", "postgres://user:password@localhost:5432/testdb")
				t.Setenv("REDIS_URL", "redis://localhost:6379")
				t.Setenv("GMP_KEY", "")
			},
			expectErr: true,
		},
		{
			name: "Failure - Missing GMP_GEOCODE_URL",
			setup: func(t *testing.T) {
				t.Setenv("DB_URL", "postgres://user:password@localhost:5432/testdb")
				t.Setenv("REDIS_URL", "redis://localhost:6379")
				t.Setenv("GMP_KEY", "test_gmp_key")
				t.Setenv("GMP_GEOCODE_URL", "")
			},
			expectErr: true,
		},
		{
			name: "Failure - Missing GMP_WEATHER_URL",
			setup: func(t *testing.T) {
				t.Setenv("DB_URL", "postgres://user:password@localhost:5432/testdb")
				t.Setenv("REDIS_URL", "redis://localhost:6379")
				t.Setenv("GMP_KEY", "test_gmp_key")
				t.Setenv("GMP_GEOCODE_URL", "http://localhost/geocode")
				t.Setenv("GMP_WEATHER_URL", "")
			},
			expectErr: true,
		},
		{
			name: "Failure - Missing OWM_WEATHER_URL",
			setup: func(t *testing.T) {
				t.Setenv("DB_URL", "postgres://user:password@localhost:5432/testdb")
				t.Setenv("REDIS_URL", "redis://localhost:6379")
				t.Setenv("GMP_KEY", "test_gmp_key")
				t.Setenv("GMP_GEOCODE_URL", "http://localhost/geocode")
				t.Setenv("GMP_WEATHER_URL", "http://localhost/weather")
				t.Setenv("OWM_WEATHER_URL", "")
			},
			expectErr: true,
		},
		{
			name: "Failure - Missing OMETEO_WEATHER_URL",
			setup: func(t *testing.T) {
				t.Setenv("DB_URL", "postgres://user:password@localhost:5432/testdb")
				t.Setenv("REDIS_URL", "redis://localhost:6379")
				t.Setenv("GMP_KEY", "test_gmp_key")
				t.Setenv("GMP_GEOCODE_URL", "http://localhost/geocode")
				t.Setenv("GMP_WEATHER_URL", "http://localhost/weather")
				t.Setenv("OWM_WEATHER_URL", "http://localhost/weather")
				t.Setenv("OMETEO_WEATHER_URL", "")
			},
			expectErr: true,
		},
		{
			name: "Failure - Missing OWM_KEY",
			setup: func(t *testing.T) {
				t.Setenv("DB_URL", "postgres://user:password@localhost:5432/testdb")
				t.Setenv("REDIS_URL", "redis://localhost:6379")
				t.Setenv("GMP_KEY", "test_gmp_key")
				t.Setenv("GMP_GEOCODE_URL", "http://localhost/geocode")
				t.Setenv("GMP_WEATHER_URL", "http://localhost/weather")
				t.Setenv("OWM_WEATHER_URL", "http://localhost/weather")
				t.Setenv("OMETEO_WEATHER_URL", "http://localhost/weather")
				t.Setenv("OWM_KEY", "")
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup(t)
			}
			cfg, err := NewAPIConfig(io.Discard)
			if tc.expectErr {
				assert.Error(t, err, "expected an error but got none")
			} else {
				assert.NoError(t, err, "did not expect an error but got one")
				assert.NotNil(t, cfg, "expected cfg to be non-nil")
			}
		})
	}
}
