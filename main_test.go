package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/redis/go-redis/v9"
)

var (
	dbURL    string
	redisURL string
	dockerURL string
)

func TestMain(m *testing.M) {
	if os.Getenv("GO_TEST_MAIN") == "1" {
		main()
		return
	}

	dockerURL = os.Getenv("DOCKER_HOST")
	if dockerURL == "" {
		dockerURL = "unix:///var/run/docker.sock"
	}
	os.Setenv("DOCKER_HOST", dockerURL)
	
	u, err := url.Parse(dockerURL)
	if err != nil {
		log.Fatalf("Could not parse DOCKER_HOST: %s", err)
	}
	host := u.Hostname()
	if host == "" {
		host = "localhost"
	}

	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not construct pool: %s", err)
	}

	err = pool.Client.Ping()
	if err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}

	test_network, err := pool.CreateNetwork("test_network")
	if err != nil {
		log.Fatalf("Could not create Docker network: %s", err)
	}

	pgResource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "13",
		Env: []string{
			"POSTGRES_PASSWORD=secret",
			"POSTGRES_USER=user",
			"POSTGRES_DB=testdb",
			"listen_addresses='*'",
		},
		NetworkID: test_network.Network.ID,
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start PostgreSQL container: %s", err)
	}
	dbURL = fmt.Sprintf("postgres://user:secret@%s:%s/testdb?sslmode=disable", host, pgResource.GetPort("5432/tcp"))

	redisResource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "redis",
		Tag:        "6",
		NetworkID:  test_network.Network.ID,
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start Redis container: %s", err)
	}
	redisURL = fmt.Sprintf("redis://%s:%s", host, redisResource.GetPort("6379/tcp"))

	os.Setenv("DB_URL", dbURL)
	os.Setenv("REDIS_URL", redisURL)
	
	pool.MaxWait = 30 * time.Second
	if err = pool.Retry(func() error {
		db, err := sql.Open("postgres", dbURL)
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		if err := pool.Purge(pgResource); err != nil {
			log.Fatalf("Could not purge PostgreSQL container: %s", err)
		}
		if err := pool.Purge(redisResource); err != nil {
			log.Fatalf("Could not purge Redis container: %s", err)
		}
		if err := pool.RemoveNetwork(test_network); err != nil {
			log.Fatalf("Could not remove Docker network: %s", err)
		}
		log.Fatalf("Could not connect to PostgreSQL container: %s", err)
	}

	if err = pool.Retry(func() error {
		opts, err := redis.ParseURL(redisURL)
		if err != nil {
			return err
		}
		client := redis.NewClient(opts)
		return client.Ping(context.Background()).Err()
	}); err != nil {
		if err := pool.Purge(pgResource); err != nil {
			log.Fatalf("Could not purge PostgreSQL container: %s", err)
		}
		if err := pool.Purge(redisResource); err != nil {
			log.Fatalf("Could not purge Redis container: %s", err)
		}
		if err := pool.RemoveNetwork(test_network); err != nil {
			log.Fatalf("Could not remove Docker network: %s", err)
		}
		log.Fatalf("Could not connect to Redis container: %s", err)
	}

	code := m.Run()

	if err := pool.Purge(pgResource); err != nil {
		log.Fatalf("Could not purge PostgreSQL container: %s", err)
	}
	if err := pool.Purge(redisResource); err != nil {
		log.Fatalf("Could not purge Redis container: %s", err)
	}
	if err := pool.RemoveNetwork(test_network); err != nil {
			log.Fatalf("Could not remove Docker network: %s", err)
	}

	os.Exit(code)
}

func TestMainExecution(t *testing.T) {
	baseEnv := map[string]string{
		"GMP_KEY":            "dummy",
		"GMP_GEOCODE_URL":    "dummy",
		"GMP_WEATHER_URL":    "dummy",
		"OWM_WEATHER_URL":    "dummy",
		"OMETEO_WEATHER_URL": "dummy",
		"OWM_KEY":            "dummy",
		"DOCKER_HOST":	  	  "127.0.0.1:2375",	
	}

	testCases := []struct {
		name          string
		env           map[string]string
		wantExitCode  int
		wantInLog     []string
		checkDuration time.Duration
	}{
		{
			name: "Success",
			env: map[string]string{
				"DB_URL":    dbURL,
				"REDIS_URL": redisURL,
				"PORT":      "8081",
			},
			wantExitCode: -1,
			wantInLog: []string{
				"configuration loaded",
				"starting scheduler",
				"starting server",
			},
			checkDuration: 200 * time.Millisecond,
		},
		{
			name: "Failure - NewAPIConfig fails",
			env: map[string]string{
				"DB_URL":    "",
				"REDIS_URL": "",
			},
			wantExitCode: 1,
			wantInLog:    []string{"failed to load configuration"},
		},
		{
			name: "Failure - DB connection fails",
			env: map[string]string{
				"DB_URL":    "postgres://user:secret@localhost:9999/testdb?sslmode=disable",
				"REDIS_URL": redisURL,
			},
			wantExitCode: 1,
			wantInLog:    []string{"couldn't connect to database"},
		},
		{
			name: "Failure - Cache connection fails",
			env: map[string]string{
				"DB_URL":    dbURL,
				"REDIS_URL": "redis://localhost:9999",
			},
			wantExitCode: 1,
			wantInLog:    []string{"couldn't connect to cache"},
		},
		{
			name: "Failure - Server startup fails (port in use)",
			env: map[string]string{
				"DB_URL":    dbURL,
				"REDIS_URL": redisURL,
				"PORT":      "8082",
			},
			wantExitCode: 1,
			wantInLog:    []string{"server startup failed"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.env["PORT"] == "8082" {
				listener, err := net.Listen("tcp", ":8082")
				if err != nil {
					t.Logf("could not listen on port 8082: %v", err)
				} else {
					t.Cleanup(func() { listener.Close() })
				}
			}

			cmd := exec.Command(os.Args[0], "-test.run=^TestMain$")
			cmd.Env = []string{"GO_TEST_MAIN=1"}
			for k, v := range baseEnv {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
			}
			for k, v := range tc.env {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
			}

			var out bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &out

			err := cmd.Start()
			if err != nil {
				t.Fatalf("failed to start subprocess: %v", err)
			}

			if tc.checkDuration > 0 {
				time.Sleep(tc.checkDuration)
				if err := cmd.Process.Kill(); err != nil {
					t.Fatalf("failed to kill process: %v", err)
				}
			} else {
				err = cmd.Wait()
			}

			logs := out.String()

			for _, expectedLog := range tc.wantInLog {
				if !strings.Contains(logs, expectedLog) {
					t.Errorf("expected log to contain %q, but it didn't. Logs:\n%s", expectedLog, logs)
				}
			}

			if tc.wantExitCode != -1 {
				if err == nil {
					t.Fatalf("process exited with code 0, but expected non-zero exit code. Logs:\n%s", logs)
				}
				exitErr, ok := err.(*exec.ExitError)
				if !ok {
					t.Fatalf("expected command to fail with an ExitError, but got %T: %v", err, err)
				}
				if exitErr.ExitCode() != tc.wantExitCode {
					t.Errorf("expected exit code %d, got %d. Logs:\n%s", tc.wantExitCode, exitErr.ExitCode(), logs)
				}
			}
		})
	}
}
