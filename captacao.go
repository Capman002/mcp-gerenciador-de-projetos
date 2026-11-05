package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Shared with the backend; schemas validate both request and structured response.
//go:embed captacao-tools.json
var captacaoCatalog []byte

type captacaoTool struct {
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	InputSchema  json.RawMessage `json:"inputSchema"`
	OutputSchema json.RawMessage `json:"outputSchema"`
}

func RegisterCaptacaoTools(s *server.MCPServer, cfg *Config, salesOnly bool) {
	var definitions []captacaoTool
	if err := json.Unmarshal(captacaoCatalog, &definitions); err != nil {
		panic("invalid embedded Captação catalog")
	}
	for _, definition := range definitions {
		name := definition.Name
		tool := mcp.Tool{Name: name, Description: definition.Description, RawInputSchema: definition.InputSchema, RawOutputSchema: definition.OutputSchema}
		s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			credential := requestCredential(ctx, cfg)
			if salesOnly {
				credential, _ = ctx.Value(requestCredentialKey{}).(string)
				if !credentialPattern.MatchString(credential) {
					return mcp.NewToolResultError("Não autorizado."), nil
				}
			}
			if credential == "" {
				return mcp.NewToolResultError("Não autorizado."), nil
			}
			data, err := callCaptacao(ctx, cfg, credential, name, request.GetArguments())
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			// One structured copy avoids doubling JSON in text and structuredContent.
			return &mcp.CallToolResult{Content: []mcp.Content{}, StructuredContent: data}, nil
		})
	}
}

func callCaptacao(ctx context.Context, cfg *Config, credential, tool string, args map[string]any) (any, error) {
	if args == nil {
		args = map[string]any{}
	}
	body, err := json.Marshal(struct {
		Tool string         `json:"tool"`
		Args map[string]any `json:"args"`
	}{tool, args})
	if err != nil {
		return nil, fmt.Errorf("Argumentos inválidos.")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.ApiUrl+"/api/mcp/captacao", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("Não foi possível consultar Captação.")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+credential)
	response, err := apiHTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Não foi possível consultar Captação.")
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 256*1024+1))
	if err != nil || len(raw) > 256*1024 {
		return nil, fmt.Errorf("Resposta excedeu o limite; use uma página menor.")
	}
	var result struct {
		Success bool   `json:"success"`
		Data    any    `json:"data"`
		Error   string `json:"error"`
	}
	if json.Unmarshal(raw, &result) != nil {
		return nil, fmt.Errorf("Resposta inválida de Captação.")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 || !result.Success {
		if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("Não autorizado.")
		}
		if len(result.Error) > 0 && len(result.Error) <= 1000 {
			return nil, fmt.Errorf("%s", result.Error)
		}
		return nil, fmt.Errorf("Não foi possível executar esta ação de Captação.")
	}
	if result.Data == nil {
		return nil, fmt.Errorf("Resposta inválida de Captação.")
	}
	return result.Data, nil
}
