package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSalesHTTPScopesSchemasCredentialAndRevocation(t *testing.T) {
	token := "epx_cap_" + strings.Repeat("a", 43)
	state := "active"
	accessCalls, toolCalls := 0, 0
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("Authorization") != "Bearer "+token {
			w.WriteHeader(401)
			io.WriteString(w, `{"error":"Unauthorized"}`)
			return
		}
		if r.URL.Path == "/api/mcp/access" {
			accessCalls++
			switch state {
			case "revoked":
				w.WriteHeader(401)
				io.WriteString(w, `{}`)
			case "role-changed":
				w.WriteHeader(403)
				io.WriteString(w, `{}`)
			case "admin":
				io.WriteString(w, `{"id":1,"scope":"admin"}`)
			default:
				io.WriteString(w, `{"id":2,"name":"Seller","scope":"captacao"}`)
			}
			return
		}
		if r.URL.Path != "/api/mcp/captacao" {
			t.Errorf("seller used administrative API: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		toolCalls++
		var payload struct {
			Tool string         `json:"tool"`
			Args map[string]any `json:"args"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		if payload.Tool != "captacao_listar" {
			t.Errorf("unexpected action: %s", payload.Tool)
		}
		io.WriteString(w, `{"success":true,"data":{"items":[],"nextCursor":null}}`)
	}))
	defer backend.Close()
	cfg := &Config{ApiUrl: backend.URL, ApiKey: "administrator-key"}
	app := httptest.NewServer(newHTTPHandler(cfg, strings.Repeat("z", 40)))
	defer app.Close()
	rpc := func(path, method string, params any) (int, map[string]any) {
		t.Helper()
		body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
		req, _ := http.NewRequest(http.MethodPost, app.URL+path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set("Authorization", "Bearer administrator-key")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		raw, _ := io.ReadAll(res.Body)
		var got map[string]any
		if res.StatusCode == 200 {
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatalf("invalid RPC result: %s", raw)
			}
		}
		return res.StatusCode, got
	}
	path := "/mcp/sales/" + token
	code, list := rpc(path, "tools/list", map[string]any{})
	if code != 200 {
		t.Fatalf("catalog HTTP %d", code)
	}
	result := list["result"].(map[string]any)
	tools := result["tools"].([]any)
	if len(tools) != 13 {
		t.Fatalf("seller tool count %d", len(tools))
	}
	for _, item := range tools {
		tool := item.(map[string]any)
		if !strings.HasPrefix(tool["name"].(string), "captacao_") {
			t.Fatalf("admin tool exposed: %s", tool["name"])
		}
		if tool["inputSchema"] == nil || tool["outputSchema"] == nil {
			t.Fatal("missing typed contract")
		}
	}
	for _, name := range []string{"consulta_sql", "gerenciar_projetos", "gerenciar_clientes", "gerenciar_parcelas"} {
		_, got := rpc(path, "tools/call", map[string]any{"name": name, "arguments": map[string]any{}})
		if got["error"] == nil {
			t.Fatalf("admin tool callable: %s", name)
		}
	}
	if toolCalls != 0 {
		t.Fatal("admin calls reached backend")
	}
	for _, args := range []map[string]any{{"limit": 51}, {"limit": 1.5}, {"limit": "20"}, {"query": "SELECT * FROM projects"}, {"status": "invalid"}} {
		_, got := rpc(path, "tools/call", map[string]any{"name": "captacao_listar", "arguments": args})
		if got["error"] == nil {
			r, _ := got["result"].(map[string]any)
			if r["isError"] != true {
				t.Fatalf("invalid arguments accepted: %v", args)
			}
		}
	}
	if toolCalls != 0 {
		t.Fatal("invalid arguments reached API")
	}
	_, called := rpc(path, "tools/call", map[string]any{"name": "captacao_listar", "arguments": map[string]any{"limit": 20}})
	r, ok := called["result"].(map[string]any)
	if !ok || r["isError"] == true || r["structuredContent"] == nil {
		t.Fatalf("valid request failed: %v", called)
	}
	if toolCalls != 1 {
		t.Fatalf("valid call count %d", toolCalls)
	}
	if len(r["content"].([]any)) != 0 {
		t.Fatal("structured data duplicated in text")
	}
	for _, s := range []string{"revoked", "role-changed", "admin"} {
		state = s
		code, _ := rpc(path, "tools/list", map[string]any{})
		if code != 401 && code != 403 {
			t.Fatalf("%s authorization returned %d", s, code)
		}
		code, _ = rpc(path, "tools/call", map[string]any{"name": "captacao_listar", "arguments": map[string]any{}})
		if code != 401 && code != 403 {
			t.Fatalf("%s call returned %d", s, code)
		}
	}
	if toolCalls != 1 {
		t.Fatal("revoked credential reached data API")
	}
	previous := accessCalls
	for _, p := range []string{"/mcp/sales/short", "/mcp/sales/" + token + "/extra", "/mcp/captacao", "/mcp/not-admin-token"} {
		code, _ := rpc(p, "tools/list", map[string]any{})
		if code != 401 && code != 404 {
			t.Fatalf("invalid endpoint accepted: %s", p)
		}
	}
	if accessCalls != previous {
		t.Fatal("malformed credential reached authorization service")
	}
	code, admin := rpc("/mcp/"+strings.Repeat("z", 40), "tools/list", map[string]any{})
	if code != 200 || len(admin["result"].(map[string]any)["tools"].([]any)) != 27 {
		t.Fatal("administrative catalog compatibility broken")
	}
}

func TestCaptacaoReplyBoundAndAuthFailure(t *testing.T) {
	for _, scenario := range []string{"large", "redirect", "forbidden", "invalid", "wrong-output"} {
		t.Run(scenario, func(t *testing.T) {
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch scenario {
				case "large":
					io.WriteString(w, strings.Repeat("x", 256*1024+1))
				case "redirect":
					http.Redirect(w, r, "/other", 302)
				case "forbidden":
					w.WriteHeader(403)
					io.WriteString(w, `{"error":"secret internal detail"}`)
				case "invalid":
					io.WriteString(w, "invalid")
				default:
					io.WriteString(w, `{"success":true,"data":null}`)
				}
			}))
			defer backend.Close()
			_, err := callCaptacao(t.Context(), &Config{ApiUrl: backend.URL}, "credential", "captacao_listar", map[string]any{})
			if err == nil || strings.Contains(err.Error(), "secret internal detail") {
				t.Fatal("unsafe backend response accepted")
			}
		})
	}
}

func TestSalesToolCannotFallbackToAdminCredential(t *testing.T) {
	calls := 0
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(500) }))
	defer backend.Close()
	server := newMCPServer(&Config{ApiUrl: backend.URL, ApiKey: "administrator-key"}, true)
	response := server.HandleMessage(t.Context(), json.RawMessage(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"captacao_listar","arguments":{}}}`))
	raw, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 0 || !bytes.Contains(raw, []byte(`"isError":true`)) {
		t.Fatalf("missing sales principal fell back to administrator access: %s", raw)
	}
}

func TestCaptacaoMalformedOutputIsRejectedBySDK(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"success":true,"data":{"items":"not-an-array","nextCursor":null}}`)
	}))
	defer backend.Close()
	server := newMCPServer(&Config{ApiUrl: backend.URL, ApiKey: "administrator-key"}, true)
	ctx := context.WithValue(t.Context(), requestCredentialKey{}, "epx_cap_"+strings.Repeat("b", 43))
	response := server.HandleMessage(ctx, json.RawMessage(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"captacao_listar","arguments":{}}}`))
	raw, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte(`"isError":true`)) {
		t.Fatalf("wrong output escaped schema validation: %s", raw)
	}
}

func TestIndividualAdminUsesOwnCredentialAcrossToolsAndHelpers(t *testing.T) {
	token := "epx_adm_" + strings.Repeat("c", 43)
	scope := "admin"
	toolCalls, proposalCalls := 0, 0
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			t.Error("individual admin credential replaced by master")
			w.WriteHeader(401)
			return
		}
		if r.URL.Path == "/api/mcp/access" {
			json.NewEncoder(w).Encode(map[string]any{"id": 3, "scope": scope})
			return
		}
		switch r.URL.Path {
		case "/api/mcp/query":
			toolCalls++
			io.WriteString(w, `{"success":true,"results":[]}`)
		case "/api/mcp/captacao":
			toolCalls++
			io.WriteString(w, `{"success":true,"data":{"items":[],"nextCursor":null}}`)
		case "/propostas/test":
			proposalCalls++
			io.WriteString(w, "<html><main>Individual admin fixture</main></html>")
		default:
			t.Errorf("unexpected API %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer backend.Close()
	cfg := &Config{ApiUrl: backend.URL, ApiKey: "administrator-master-key"}
	app := httptest.NewServer(newHTTPHandler(cfg, strings.Repeat("z", 40)))
	defer app.Close()
	call := func(path, name string, args map[string]any) int {
		t.Helper()
		body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": name, "arguments": args}})
		req, _ := http.NewRequest(http.MethodPost, app.URL+path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set("Authorization", "Bearer administrator-master-key")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		raw, _ := io.ReadAll(res.Body)
		if res.StatusCode == 200 && (!bytes.Contains(raw, []byte(`"structuredContent"`)) || bytes.Contains(raw, []byte(`"isError":true`))) {
			t.Fatalf("typed admin call failed: %s", raw)
		}
		return res.StatusCode
	}
	path := "/mcp/admin/" + token
	for _, tool := range []string{"gerenciar_projetos", "captacao_listar"} {
		args := map[string]any{}
		if tool == "gerenciar_projetos" {
			args["acao"] = "listar"
		}
		if call(path, tool, args) != 200 {
			t.Fatal("individual admin request rejected")
		}
	}
	if toolCalls != 2 {
		t.Fatal("individual tools did not reach API")
	}
	text, err := GetProposalText(context.WithValue(t.Context(), requestCredentialKey{}, token), cfg, "test")
	if err != nil || !strings.Contains(text, "Individual admin fixture") || proposalCalls != 1 {
		t.Fatal("proposal helper lost individual admin principal")
	}
	scope = "captacao"
	if call(path, "captacao_listar", map[string]any{}) != 403 || toolCalls != 2 {
		t.Fatal("downgraded admin principal retained admin connection")
	}
	if call("/mcp/sales/"+token, "captacao_listar", map[string]any{}) != 401 {
		t.Fatal("admin token accepted on seller endpoint")
	}
}

func TestExplicitEmptyPrincipalNeverFallsBackToMaster(t *testing.T) {
	masterUsed := false
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Authorization"), "administrator-master-key") {
			masterUsed = true
		}
		w.WriteHeader(401)
		io.WriteString(w, `{"error":"Unauthorized"}`)
	}))
	defer backend.Close()
	cfg := &Config{ApiUrl: backend.URL, ApiKey: "administrator-master-key"}
	ctx := context.WithValue(t.Context(), requestCredentialKey{}, "")
	CallQuery(ctx, cfg, "SELECT 1")
	GetProposalText(ctx, cfg, "fixture")
	if masterUsed {
		t.Fatal("explicit empty credential silently fell back to master")
	}
}
