package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/redis/go-redis/v9"
)

func TestMainExecution(t *testing.T) {
	// Setup Docker pool
	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatalf("Could not construct pool: %s", err)
	}
	if err := pool.Client.Ping(); err != nil {
		t.Fatalf("Could not connect to Docker: %s", err)
	}

	// Clean up previous networks
	networks, err := pool.Client.ListNetworks()
	if err != nil {
		t.Fatalf("Could not list networks: %s", err)
	}
	for _, network := range networks {
		if network.Name == "willitrain-test-network" {
			if err := pool.Client.RemoveNetwork(network.ID); err != nil {
				t.Fatalf("Could not remove existing network: %s", err)
			}
		}
	}

	// Create a network for the containers
	network, err := pool.Client.CreateNetwork(docker.CreateNetworkOptions{
		Name: "willitrain-test-network",
	})
	if err != nil {
		t.Fatalf("Could not create network: %s", err)
	}
	defer func() {
		if err := pool.Client.RemoveNetwork(network.ID); err != nil {
			t.Fatalf("Could not remove network: %s", err)
		}
	}()

	// Setup Postgres
	postgres, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "13",
		Env: []string{
			"POSTGRES_USER=test",
			"POSTGRES_PASSWORD=test",
			"POSTGRES_DB=test",
		},
		NetworkID: network.ID,
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		t.Fatalf("Could not start postgres: %s", err)
	}
	defer func() {
		if err := pool.Purge(postgres); err != nil {
			t.Fatalf("Could not purge postgres: %s", err)
		}
	}()

	// Setup Redis
	redisContainer, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "redis",
		Tag:        "6",
		NetworkID:  network.ID,
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		t.Fatalf("Could not start redis: %s", err)
	}
	defer func() {
		if err := pool.Purge(redisContainer); err != nil {
			t.Fatalf("Could not purge redis: %s", err)
		}
	}()

	// Determine Docker host IP
	dockerHost := os.Getenv("DOCKER_HOST")
	var dockerHostIP string
	if strings.HasPrefix(dockerHost, "unix://") {
		dockerHostIP = "localhost"
	} else {
		u, err := url.Parse(dockerHost)
		if err != nil {
			t.Fatalf("Could not parse DOCKER_HOST: %s", err)
		}
		dockerHostIP = u.Hostname()
	}

	dbURL := fmt.Sprintf("postgres://test:test@%s/test?sslmode=disable", net.JoinHostPort(dockerHostIP, postgres.GetPort("5432/tcp")))
	redisURL := fmt.Sprintf("redis://%s", net.JoinHostPort(dockerHostIP, redisContainer.GetPort("6379/tcp")))
	port := "8081"

	// Wait for DB and Redis to be ready
	if err := pool.Retry(func() error {
		db, err := sql.Open("postgres", dbURL)
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		t.Fatalf("Could not connect to database: %s", err)
	}

	if err := pool.Retry(func() error {
		opts, err := redis.ParseURL(redisURL)
		if err != nil {
			return err
		}
		client := redis.NewClient(opts)
		return client.Ping(context.Background()).Err()
	}); err != nil {
		t.Fatalf("Could not connect to redis: %s", err)
	}

	// --- Test Cases ---

	t.Run("Success", func(t *testing.T) {
		t.Setenv("DB_URL", dbURL)
		t.Setenv("REDIS_URL", redisURL)
		t.Setenv("PORT", port)
		t.Setenv("GMP_KEY", "test")
		t.Setenv("GMP_GEOCODE_URL", "test")
		t.Setenv("GMP_WEATHER_URL", "test")
		t.Setenv("OWM_WEATHER_URL", "test")
		t.Setenv("OMETEO_WEATHER_URL", "test")
		t.Setenv("OWM_KEY", "test")
		t.Setenv("DEV_MODE", "true")

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			if err := run(ctx); err != nil && err != http.ErrServerClosed {
				log.Printf("run() returned an error: %v", err)
			}
		}()

		var resp *http.Response
		err = pool.Retry(func() error {
			var httpErr error
			resp, httpErr = http.Get("http://localhost:" + port + "/api/config")
			if httpErr != nil {
				return httpErr
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Could not connect to the server: %s", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status OK, got %s", resp.Status)
		}
	})

	t.Run("Failure - NewAPIConfig fails", func(t *testing.T) {
		// Unset a required env var
		t.Setenv("DB_URL", "")
		err := run(context.Background())
		if err == nil {
			t.Fatal("expected an error but got nil")
		}
		if !strings.Contains(err.Error(), "missing required environment variable: DB_URL") {
			t.Errorf("expected error to contain 'missing required environment variable: DB_URL', got %q", err.Error())
		}
	})

	t.Run("Failure - DB connection fails", func(t *testing.T) {
		t.Setenv("DB_URL", "postgres://bad:user@localhost/test")
		t.Setenv("REDIS_URL", redisURL) // Set other vars to valid values
		t.Setenv("GMP_KEY", "test")
		t.Setenv("GMP_GEOCODE_URL", "test")
		t.Setenv("GMP_WEATHER_URL", "test")
		t.Setenv("OWM_WEATHER_URL", "test")
		t.Setenv("OMETEO_WEATHER_URL", "test")
		t.Setenv("OWM_KEY", "test")

		err := run(context.Background())
		if err == nil {
			t.Fatal("expected an error but got nil")
		}
		if !strings.Contains(err.Error(), "couldn't connect to database") {
			t.Errorf("expected error to contain 'couldn't connect to database', got %q", err.Error())
		}
	})

	t.Run("Failure - Cache connection fails", func(t *testing.T) {
		t.Setenv("DB_URL", dbURL)
		t.Setenv("REDIS_URL", "redis://bad-host:6379")
		t.Setenv("GMP_KEY", "test")
		t.Setenv("GMP_GEOCODE_URL", "test")
		t.Setenv("GMP_WEATHER_URL", "test")
		t.Setenv("OWM_WEATHER_URL", "test")
		t.Setenv("OMETEO_WEATHER_URL", "test")
		t.Setenv("OWM_KEY", "test")

		err := run(context.Background())
		if err == nil {
			t.Fatal("expected an error but got nil")
		}
		if !strings.Contains(err.Error(), "couldn't connect to cache") {
			t.Errorf("expected error to contain 'couldn't connect to cache', got %q", err.Error())
		}
	})
}