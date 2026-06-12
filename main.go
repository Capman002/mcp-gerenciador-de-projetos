package main

import (
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	apiUrl := os.Getenv("SVELTEKIT_API_URL")
	apiKey := os.Getenv("MCP_API_KEY")

	if apiUrl == "" {
		fmt.Fprintln(os.Stderr, "SVELTEKIT_API_URL não definida. Configure no mcp_config.json do seu host.")
		os.Exit(1)
	}

	cfg := &Config{ApiUrl: apiUrl, ApiKey: apiKey}

	s := server.NewMCPServer("gerenciador-projetos", "1.0.0")
	RegisterAllTools(s, cfg)

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao iniciar servidor MCP: %v\n", err)
		os.Exit(1)
	}
}
