package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisCache_Set(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name        string
		key         string
		value       any
		expiration  time.Duration
		setupMock   func(mock redismock.ClientMock, key string, value any, expiration time.Duration)
		expectedErr error
	}{
		{
			name:       "Success",
			key:        "test-key",
			value:      "test-value",
			expiration: 1 * time.Minute,
			setupMock: func(mock redismock.ClientMock, key string, value any, expiration time.Duration) {
				jsonData, _ := json.Marshal(value)
				mock.ExpectSet(key, jsonData, expiration).SetVal("OK")
			},
			expectedErr: nil,
		},
		{
			name:        "Error on json.Marshal",
			key:         "test-key",
			value:       make(chan int),
			expiration:  1 * time.Minute,
			setupMock:   func(mock redismock.ClientMock, key string, value any, expiration time.Duration) {},
			expectedErr: &json.UnsupportedTypeError{},
		},
		{
			name:       "Error from Redis client",
			key:        "test-key",
			value:      "test-value",
			expiration: 1 * time.Minute,
			setupMock: func(mock redismock.ClientMock, key string, value any, expiration time.Duration) {
				jsonData, _ := json.Marshal(value)
				mock.ExpectSet(key, jsonData, expiration).SetErr(errors.New("redis error"))
			},
			expectedErr: errors.New("redis error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			redisClient, redisMock := redismock.NewClientMock()
			defer redisClient.Close()

			cache := NewRedisCache(redisClient)

			tc.setupMock(redisMock, tc.key, tc.value, tc.expiration)

			err := cache.Set(ctx, tc.key, tc.value, tc.expiration)

			if tc.expectedErr != nil {
				require.Error(t, err)
				if _, ok := tc.expectedErr.(*json.UnsupportedTypeError); ok {
					assert.IsType(t, &json.UnsupportedTypeError{}, err)
				} else {
					assert.EqualError(t, err, tc.expectedErr.Error())
				}
			} else {
				require.NoError(t, err)
			}

			assert.NoError(t, redisMock.ExpectationsWereMet())
		})
	}
}

func TestRedisCache_Get(t *testing.T) {
	ctx := context.Background()
	redisClient, redisMock := redismock.NewClientMock()
	defer redisClient.Close()

	cache := NewRedisCache(redisClient)
	key := "test-key"
	expectedValue := "test-value"

	redisMock.ExpectGet(key).SetVal(expectedValue)

	value, err := cache.Get(ctx, key)

	require.NoError(t, err)
	assert.Equal(t, expectedValue, value)
	assert.NoError(t, redisMock.ExpectationsWereMet())
}

func TestRedisCache_Get_Error(t *testing.T) {
	ctx := context.Background()
	redisClient, redisMock := redismock.NewClientMock()
	defer redisClient.Close()

	cache := NewRedisCache(redisClient)
	key := "test-key"

	redisMock.ExpectGet(key).SetErr(redis.Nil)

	_, err := cache.Get(ctx, key)

	require.Error(t, err)
	assert.Equal(t, redis.Nil, err)
	assert.NoError(t, redisMock.ExpectationsWereMet())
}

func TestRedisCache_Flush(t *testing.T) {
	ctx := context.Background()
	redisClient, redisMock := redismock.NewClientMock()
	defer redisClient.Close()

	cache := NewRedisCache(redisClient)

	redisMock.ExpectFlushDB().SetVal("OK")

	err := cache.Flush(ctx)

	require.NoError(t, err)
	assert.NoError(t, redisMock.ExpectationsWereMet())
}

func TestRedisCache_Flush_Error(t *testing.T) {
	ctx := context.Background()
	redisClient, redisMock := redismock.NewClientMock()
	defer redisClient.Close()

	cache := NewRedisCache(redisClient)

	redisMock.ExpectFlushDB().SetErr(errors.New("flush error"))

	err := cache.Flush(ctx)

	require.Error(t, err)
	assert.EqualError(t, err, "flush error")
	assert.NoError(t, redisMock.ExpectationsWereMet())
}
