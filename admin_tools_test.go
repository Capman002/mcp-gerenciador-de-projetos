package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func adminRPCFixture(t *testing.T, backend http.HandlerFunc) (func(string, map[string]any) map[string]any, *Config) {
	t.Helper()
	api := httptest.NewServer(backend)
	t.Cleanup(api.Close)
	cfg := &Config{ApiUrl: api.URL, ApiKey: "admin-fixture"}
	app := httptest.NewServer(newHTTPHandler(cfg, strings.Repeat("z", 40)))
	t.Cleanup(app.Close)
	return func(name string, args map[string]any) map[string]any {
		t.Helper()
		method := "tools/call"
		params := any(map[string]any{"name": name, "arguments": args})
		if name == "tools/list" {
			method = name
			params = map[string]any{}
		}
		body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
		req, _ := http.NewRequest("POST", app.URL+"/mcp/"+strings.Repeat("z", 40), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		response, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var value map[string]any
		if err = json.NewDecoder(response.Body).Decode(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}, cfg
}
func adminStructured(t *testing.T, response map[string]any) map[string]any {
	t.Helper()
	result, ok := response["result"].(map[string]any)
	if !ok || result["isError"] == true {
		t.Fatalf("invalid tool response: %+v", response)
	}
	content, ok := result["content"].([]any)
	if ok && len(content) > 0 {
		t.Fatal("structured JSON duplicated in content")
	}
	data, ok := result["structuredContent"].(map[string]any)
	if !ok {
		t.Fatalf("missing typed response: %+v", response)
	}
	return data
}
func adminIsError(response map[string]any) bool {
	result, _ := response["result"].(map[string]any)
	return response["error"] != nil || result["isError"] == true
}

func TestAdminSchemasBoundedListsAndSecretProjection(t *testing.T) {
	calls := 0
	query := ""
	var params []any
	rpc, _ := adminRPCFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		var body struct {
			Query  string `json:"query"`
			Params []any  `json:"params"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		query, params = body.Query, body.Params
		rows := []map[string]any{}
		for i := 1; i <= 21; i++ {
			rows = append(rows, map[string]any{"id": i, "name": "Project", "client_id": 1, "github_token": "SECRET", "google_refresh_token": "SECRET"})
		}
		json.NewEncoder(w).Encode(map[string]any{"success": true, "results": rows})
	})
	list := rpc("tools/list", nil)["result"].(map[string]any)["tools"].([]any)
	if len(list) != 27 {
		t.Fatalf("tools %d", len(list))
	}
	for _, value := range list {
		tool := value.(map[string]any)
		if tool["inputSchema"] == nil || tool["outputSchema"] == nil {
			t.Fatal("missing schema")
		}
	}
	result := adminStructured(t, rpc("gerenciar_projetos", map[string]any{"acao": "listar", "client_id": float64(7)}))
	rows := result["results"].([]any)
	if len(rows) != 20 || result["next_offset"] != float64(20) {
		t.Fatal("pagination missing")
	}
	raw, _ := json.Marshal(result)
	if bytes.Contains(raw, []byte("SECRET")) {
		t.Fatal("credentials leaked")
	}
	if strings.Contains(query, "p.*") || strings.Contains(query, "github_token") || !strings.Contains(query, "p.client_id=?") || !strings.Contains(query, "LIMIT ? OFFSET ?") {
		t.Fatalf("unsafe query %s", query)
	}
	if len(params) != 3 || params[0] != float64(7) || params[1] != float64(21) || params[2] != float64(0) {
		t.Fatalf("wrong binds %+v", params)
	}
	before := calls
	for _, args := range []map[string]any{{"acao": "buscar", "id": "1 OR 1=1"}, {"acao": "listar", "limit": 51}, {"acao": "listar", "offset": -1}, {"acao": "listar", "client_id": 1.5}, {"acao": "listar", "unknown": true}} {
		if !adminIsError(rpc("gerenciar_projetos", args)) {
			t.Fatal("invalid input accepted")
		}
	}
	if calls != before {
		t.Fatal("invalid input reached SQL")
	}
}
func TestAdminMutationRetainsZeroAndFalseAndBindsStatus(t *testing.T) {
	var payload map[string]any
	calls := 0
	rpc, _ := adminRPCFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		json.NewDecoder(r.Body).Decode(&payload)
		io.WriteString(w, `{"success":true,"id":12}`)
	})
	adminStructured(t, rpc("gerenciar_entregas", map[string]any{"acao": "editar", "id": float64(1), "percentual": float64(0), "order_index": float64(0), "client_approved": false}))
	data := payload["data"].(map[string]any)
	if data["percentual"] != float64(0) || data["order_index"] != float64(0) || data["client_approved"] != false {
		t.Fatal("zero/false update lost")
	}
	before := calls
	if !adminIsError(rpc("gerenciar_entregas", map[string]any{"acao": "listar", "project_id": float64(1), "status": "x' OR 1=1 --"})) || calls != before {
		t.Fatal("status injection reached API")
	}
	adminStructured(t, rpc("gerenciar_entregas", map[string]any{"acao": "listar", "project_id": float64(3), "status": "Em Pauta"}))
	if !strings.Contains(payload["query"].(string), "status=?") {
		t.Fatal("status not bound")
	}
	params := payload["params"].([]any)
	if params[1] != "Em Estruturação" {
		t.Fatalf("status alias missing: %+v", params)
	}
}
func TestAdminSQLAndProposalPagesAreBoundedAndUnicodeSafe(t *testing.T) {
	var query string
	text := strings.Repeat("🙂", 9000)
	rpc, _ := adminRPCFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/propostas/") {
			io.WriteString(w, "<main>"+text+"</main>")
			return
		}
		var payload map[string]any
		json.NewDecoder(r.Body).Decode(&payload)
		query, _ = payload["query"].(string)
		rows := []map[string]any{}
		for i := 0; i < 11; i++ {
			rows = append(rows, map[string]any{"v": i})
		}
		json.NewEncoder(w).Encode(map[string]any{"success": true, "results": rows})
	})
	result := adminStructured(t, rpc("consulta_sql", map[string]any{"query": "SELECT v FROM example;", "limit": float64(10)}))
	if len(result["rows"].([]any)) != 10 || result["next_offset"] != float64(10) || !strings.HasSuffix(query, "LIMIT 11 OFFSET 0") {
		t.Fatalf("SQL unbounded %s %+v", query, result)
	}
	first := adminStructured(t, rpc("ver_proposta", map[string]any{"pagina": "example"}))
	if len([]rune(first["text"].(string))) != 8000 || first["next_character"] != float64(8000) {
		t.Fatal("proposal first chunk invalid")
	}
	second := adminStructured(t, rpc("ver_proposta", map[string]any{"pagina": "example", "character": first["next_character"]}))
	if first["text"].(string)+second["text"].(string) != text {
		t.Fatal("proposal continuation dropped Unicode")
	}
	if !adminIsError(rpc("ver_proposta", map[string]any{"pagina": "../admin"})) {
		t.Fatal("proposal traversal accepted")
	}
}
func TestAdminOutputTypesAreEnforced(t *testing.T) {
	rpc, _ := adminRPCFixture(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"success":true,"results":[{"id":"not-an-integer"}]}`)
	})
	if !adminIsError(rpc("gerenciar_clientes", map[string]any{"acao": "listar"})) {
		t.Fatal("invalid output ignored schema")
	}
}
func FuzzAdminIDs(f *testing.F) {
	for _, seed := range []string{"1 OR 1=1", "-1", "1.5", "42", ";DROP TABLE clients;"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, id string) {
		schema := adminInputSchema(adminEntities[0])
		if err := adminValidate(map[string]any{"acao": "buscar", "id": id}, schema); err == nil {
			t.Fatal(fmt.Sprintf("string ID accepted: %q", id))
		}
	})
}
