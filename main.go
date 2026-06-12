package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	s := server.NewMCPServer("mcp-gerenciador-de-projetos", "1.0.0")

	// admin_action tool
	adminActionTool := mcp.NewTool("admin_action",
		mcp.WithDescription("Perform an admin action"),
		mcp.WithString("entity", mcp.Required(), mcp.Description("The entity to perform the action on")),
		mcp.WithString("action", mcp.Required(), mcp.Description("The action to perform")),
		mcp.WithString("data", mcp.Required(), mcp.Description("JSON payload for the action")),
	)

	s.AddTool(adminActionTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		entity, ok := request.Params.Arguments["entity"].(string)
		if !ok {
			return mcp.NewToolResultError("entity argument is missing or not a string"), nil
		}
		action, ok := request.Params.Arguments["action"].(string)
		if !ok {
			return mcp.NewToolResultError("action argument is missing or not a string"), nil
		}
		dataStr, ok := request.Params.Arguments["data"].(string)
		if !ok {
			return mcp.NewToolResultError("data argument is missing or not a string"), nil
		}

		apiUrl := os.Getenv("SVELTEKIT_API_URL")
		apiKey := os.Getenv("MCP_API_KEY")

		type ActionPayload struct {
			Entity string          `json:"entity"`
			Action string          `json:"action"`
			Data   json.RawMessage `json:"data"`
		}

		payload := ActionPayload{
			Entity: entity,
			Action: action,
			Data:   json.RawMessage(dataStr),
		}

		body, err := json.Marshal(payload)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal payload: %v", err)), nil
		}

		req, err := http.NewRequestWithContext(ctx, "POST", apiUrl+"/api/mcp/action", bytes.NewBuffer(body))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create request: %v", err)), nil
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Request failed: %v", err)), nil
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to read response body: %v", err)), nil
		}

		return mcp.NewToolResultText(string(respBody)), nil
	})

	// query_db tool
	queryDbTool := mcp.NewTool("query_db",
		mcp.WithDescription("Query the database"),
		mcp.WithString("query", mcp.Required(), mcp.Description("The query string")),
	)

	s.AddTool(queryDbTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, ok := request.Params.Arguments["query"].(string)
		if !ok {
			return mcp.NewToolResultError("query argument is missing or not a string"), nil
		}

		apiUrl := os.Getenv("SVELTEKIT_API_URL")
		apiKey := os.Getenv("MCP_API_KEY")

		type QueryPayload struct {
			Query string `json:"query"`
		}

		payload := QueryPayload{
			Query: query,
		}

		body, err := json.Marshal(payload)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal payload: %v", err)), nil
		}

		req, err := http.NewRequestWithContext(ctx, "POST", apiUrl+"/api/mcp/query", bytes.NewBuffer(body))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create request: %v", err)), nil
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Request failed: %v", err)), nil
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to read response body: %v", err)), nil
		}

		return mcp.NewToolResultText(string(respBody)), nil
	})

	// Start the server
	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
	}
}
