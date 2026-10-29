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

	// Modo remoto (celular, claude.ai, ChatGPT): esses clientes não mandam header
	// de autenticação, então o token secreto vai no próprio caminho da URL.
	if addr := os.Getenv("MCP_HTTP_ADDR"); addr != "" {
		token := os.Getenv("MCP_HTTP_TOKEN")
		if len(token) < 32 {
			fmt.Fprintln(os.Stderr, "MCP_HTTP_TOKEN precisa ter ao menos 32 caracteres.")
			os.Exit(1)
		}
		h := server.NewStreamableHTTPServer(s, server.WithStateLess(true), server.WithEndpointPath("/mcp/"+token))
		if err := h.Start(addr); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao iniciar servidor MCP HTTP: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao iniciar servidor MCP: %v\n", err)
		os.Exit(1)
	}
}
