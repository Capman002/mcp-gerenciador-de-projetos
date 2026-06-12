package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// CallAction sends a structured action to /api/mcp/action.
func CallAction(ctx context.Context, cfg *Config, entity, action string, data map[string]any) (string, error) {
	payload := map[string]any{
		"entity": entity,
		"action": action,
		"data":   data,
	}
	return doPost(ctx, cfg, "/api/mcp/action", payload)
}

// CallQuery sends a SQL query to /api/mcp/query.
func CallQuery(ctx context.Context, cfg *Config, query string) (string, error) {
	payload := map[string]any{
		"query": query,
	}
	return doPost(ctx, cfg, "/api/mcp/query", payload)
}

// doPost is the internal HTTP helper.
func doPost(ctx context.Context, cfg *Config, path string, payload map[string]any) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal error: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", cfg.ApiUrl+path, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("request error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.ApiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read error: %w", err)
	}
	return string(respBody), nil
}
