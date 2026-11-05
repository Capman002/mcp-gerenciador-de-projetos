package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

type requestCredentialKey struct{}

var credentialPattern = regexp.MustCompile(`^epx_cap_[A-Za-z0-9_-]{43}$`)
var adminCredentialPattern = regexp.MustCompile(`^epx_adm_[A-Za-z0-9_-]{43}$`)
var apiHTTP = &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}

func requestCredential(ctx context.Context, cfg *Config) string {
	if token, ok := ctx.Value(requestCredentialKey{}).(string); ok {
		return token
	}
	return cfg.ApiKey
}

// Separate servers prevent sales from discovering or calling administrative handlers.
func newHTTPHandler(cfg *Config, adminToken string) http.Handler {
	admin := newMCPServer(cfg, false)
	sales := newMCPServer(cfg, true)
	adminHandler := server.NewStreamableHTTPServer(admin, server.WithStateLess(true), server.WithEndpointPath("/mcp/administracao"))
	salesHandler := server.NewStreamableHTTPServer(sales, server.WithStateLess(true), server.WithEndpointPath("/mcp/captacao"))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, no-store")
		next := r.Clone(r.Context())
		next.Header = r.Header.Clone()
		next.Header.Del("Authorization")
		next.URL.RawPath = ""
		handler := adminHandler
		next.URL.Path = "/mcp/administracao"
		if r.URL.Path != "/mcp/"+adminToken {
			prefix, scope, pattern := "/mcp/sales/", "captacao", credentialPattern
			if strings.HasPrefix(r.URL.Path, "/mcp/admin/") {
				prefix, scope, pattern = "/mcp/admin/", "admin", adminCredentialPattern
			}
			if !strings.HasPrefix(r.URL.Path, prefix) {
				http.NotFound(w, r)
				return
			}
			token := strings.TrimPrefix(r.URL.Path, prefix)
			if !pattern.MatchString(token) {
				http.Error(w, "Não autorizado.", http.StatusUnauthorized)
				return
			}
			status := authorizePrincipal(r.Context(), cfg, token, scope)
			if status != http.StatusOK {
				http.Error(w, "Não autorizado.", status)
				return
			}
			next = next.WithContext(context.WithValue(next.Context(), requestCredentialKey{}, token))
			if scope == "captacao" {
				handler = salesHandler
				next.URL.Path = "/mcp/captacao"
				next.Body = http.MaxBytesReader(w, r.Body, 64*1024)
			}
		}
		handler.ServeHTTP(w, next)
	})
}

func authorizePrincipal(ctx context.Context, cfg *Config, token, scope string) int {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.ApiUrl+"/api/mcp/access", nil)
	if err != nil {
		return http.StatusServiceUnavailable
	}
	req.Header.Set("Authorization", "Bearer "+token)
	response, err := apiHTTP.Do(req)
	if err != nil {
		return http.StatusServiceUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusForbidden {
			return http.StatusForbidden
		}
		if response.StatusCode == http.StatusUnauthorized {
			return http.StatusUnauthorized
		}
		return http.StatusServiceUnavailable
	}
	var access struct {
		ID    int    `json:"id"`
		Scope string `json:"scope"`
	}
	if json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&access) != nil || access.ID <= 0 || access.Scope != scope {
		return http.StatusForbidden
	}
	return http.StatusOK
}

func newMCPServer(cfg *Config, salesOnly bool) *server.MCPServer {
	name := "gerenciador-projetos"
	if salesOnly {
		name = "epixel-captacao"
	}
	s := server.NewMCPServer(name, "1.1.0", server.WithInputSchemaValidation(), server.WithOutputSchemaValidation())
	if !salesOnly {
		RegisterAllTools(s, cfg)
	}
	RegisterCaptacaoTools(s, cfg, salesOnly)
	return s
}
