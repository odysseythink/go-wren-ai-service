package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/odysseythink/go-wren-ai-service/internal/config"
)

// forceDeploy sends a GraphQL mutation to wren-ui to trigger a forced deployment.
// It mirrors the behavior of Python's src/force_deploy.py:
//   - retries with exponential backoff on network errors (max 60s, max 3 attempts)
//   - 60-second request timeout per attempt
func forceDeploy(cfg *config.Config) error {
	endpoint := cfg.WrenUIEndpoint
	if endpoint == "" {
		endpoint = "http://wren-ui:3000"
	}

	payload := map[string]any{
		"query":     "mutation Deploy($force: Boolean) { deploy(force: $force) }",
		"variables": map[string]bool{"force": true},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal graphql payload: %w", err)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	url := endpoint + "/api/graphql"

	var lastErr error
	// Exponential backoff: 1s, 2s, 4s (capped by maxTime)
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			delay := time.Duration(1<<attempt) * time.Second
			if delay > 10*time.Second {
				delay = 10 * time.Second
			}
			time.Sleep(delay)
		}

		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		var result map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			lastErr = err
			continue
		}
		resp.Body.Close()

		fmt.Printf("Forcing deployment: %v\n", result)
		return nil
	}

	return fmt.Errorf("force deploy failed after 3 attempts: %w", lastErr)
}

// waitForServer polls the local health endpoint until it responds or timeout.
func waitForServer(addr string, timeout time.Duration) error {
	client := &http.Client{Timeout: 2 * time.Second}
	url := "http://" + addr + "/health"
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("server at %s did not become ready within %v", url, timeout)
}
