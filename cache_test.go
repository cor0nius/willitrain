package main

import (
	"context"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
)

func TestCache(t *testing.T) {
	testCases := []struct {
		name  string
		setup func(redisMock redismock.ClientMock)
		check func(t *testing.T, err error)
	}{
		{
			name: "Success",
			setup: func(redisMock redismock.ClientMock) {
				redisMock.ExpectSet("key", []byte(`"value"`), 0).SetVal("OK")
				redisMock.ExpectGet("key").SetVal(`"value"`)
				redisMock.ExpectFlushDB().SetVal("OK")
			},
			check: func(t *testing.T, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
			},
		},
		{
			name: "Failure - Set Marshal Error",
			setup: func(redisMock redismock.ClientMock) {
				// No setup needed as the error occurs before any Redis interaction
			},
			check: func(t *testing.T, err error) {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
			},
		},
		{
			name: "Failure - Set Error",
			setup: func(redisMock redismock.ClientMock) {
				redisMock.ExpectSet("key", []byte(`"value"`), 0).SetErr(redis.Nil)
			},
			check: func(t *testing.T, err error) {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
			},
		},
		{
			name: "Failure - Get Error",
			setup: func(redisMock redismock.ClientMock) {
				redisMock.ExpectSet("key", []byte(`"value"`), 0).SetVal("OK")
				redisMock.ExpectGet("key").SetErr(redis.Nil)
			},
			check: func(t *testing.T, err error) {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
			},
		},
		{
			name: "Failure - Flush Error",
			setup: func(redisMock redismock.ClientMock) {
				redisMock.ExpectSet("key", []byte(`"value"`), 0).SetVal("OK")
				redisMock.ExpectGet("key").SetVal(`"value"`)
				redisMock.ExpectFlushDB().SetErr(redis.Nil)
			},
			check: func(t *testing.T, err error) {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			redisClient, redisMock := redismock.NewClientMock()
			defer redisClient.Close()

			if tc.setup != nil {
				tc.setup(redisMock)
			}

			cache := NewRedisCache(redisClient)

			err := cache.Set(context.Background(), "key", "value", 0)
			if err != nil {
				tc.check(t, err)
			}

			_, err = cache.Get(context.Background(), "key")
			if err != nil {
				tc.check(t, err)
			}

			err = cache.Flush(context.Background())
			if tc.check != nil {
				tc.check(t, err)
			}

			if err := redisMock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
