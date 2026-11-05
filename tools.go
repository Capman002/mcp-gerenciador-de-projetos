package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Each entity has a fixed projection and editable fields; SQL identifiers never come from arguments.
type adminEntity struct {
	name, entity, table, parent, fields, editable, order string
	actions                                              []string
}

var adminEntities = []adminEntity{
	{"gerenciar_clientes", "client", "clients", "", "id,name,email,is_admin,avatar_url,role", "name,email,avatar_url,is_admin", "id DESC", []string{"listar", "buscar", "criar", "editar", "excluir"}},
	{"gerenciar_projetos", "project", "projects", "client_id", "id,client_id,name,scope_summary,progress_percentage,next_payment_date,payment_status,invoice_link,github_repo,client_name", "client_id,name,scope_summary,progress_percentage,next_payment_date,payment_status,invoice_link,github_repo,github_token", "id DESC", []string{"listar", "buscar", "criar", "editar", "excluir"}},
	{"gerenciar_entregas", "task", "tasks", "project_id", "id,project_id,title,status,description,media_url,due_date,order_index,client_approved,implemented_at,percentual", "project_id,title,status,description,media_url,due_date,order_index,client_approved,implemented_at,percentual", "order_index,id", []string{"listar", "buscar", "criar", "editar", "excluir"}},
	{"gerenciar_parcelas", "payment_installment", "payment_installments", "project_id", "id,project_id,label,amount,status,due_date,order_index", "project_id,label,amount,status,due_date,order_index", "order_index,id", []string{"listar", "criar", "editar", "excluir"}},
	{"gerenciar_avisos", "notice", "quick_notices", "project_id", "id,project_id,notice,url,link_label", "project_id,notice,url,link_label", "id DESC", []string{"listar", "criar", "editar", "excluir"}},
	{"gerenciar_documentos", "document", "documents", "project_id", "id,project_id,title,url,type", "project_id,title,url,type", "id DESC", []string{"listar", "criar", "editar", "excluir"}},
	{"gerenciar_marcos", "milestone", "milestones", "project_id", "id,project_id,title,due_date,type", "project_id,title,due_date,type", "id DESC", []string{"listar", "criar", "editar", "excluir"}},
	{"gerenciar_diario", "changelog", "changelog", "project_id", "id,project_id,description,date,media_links", "project_id,description,date,media_links", "date DESC,id DESC", []string{"listar", "criar", "editar", "excluir"}},
	{"gerenciar_membros", "project_member", "project_members", "project_id", "id,project_id,client_id,role,added_at,name,email", "project_id,client_id,role", "id DESC", []string{"listar", "adicionar", "remover"}},
	{"gerenciar_eventos", "manual_event", "manual_events", "project_id", "id,project_id,title,date,author", "project_id,title,date,author", "date DESC,id DESC", []string{"listar", "criar", "editar", "excluir"}},
	{"gerenciar_anexos", "task_attachment", "task_attachments", "task_id", "id,task_id,filename,file_url,file_type,uploaded_by_name,uploaded_at", "task_id,filename,file_url,file_type,uploaded_by_name", "uploaded_at DESC,id DESC", []string{"listar", "criar", "excluir"}},
	{"anotacoes", "note", "notes", "project_id", "id,project_id,content,created_at,updated_at", "project_id,content", "updated_at DESC,id DESC", []string{"listar", "criar", "editar", "excluir"}},
}

type adminMutationInfo struct {
	Changes         int64 `json:"changes"`
	LastInsertRowid int64 `json:"lastInsertRowid"`
}
type adminAPIResponse struct {
	Success bool               `json:"success"`
	Results []map[string]any   `json:"results"`
	ID      *int64             `json:"id"`
	Info    *adminMutationInfo `json:"info"`
	Pages   []string           `json:"pages"`
	Error   string             `json:"error"`
}
type adminOutput struct {
	Success       bool               `json:"success"`
	Results       *[]map[string]any  `json:"results,omitempty"`
	ID            *int64             `json:"id,omitempty"`
	Info          *adminMutationInfo `json:"info,omitempty"`
	NextOffset    *int               `json:"next_offset,omitempty"`
	Pages         []string           `json:"pages,omitempty"`
	Text          string             `json:"text,omitempty"`
	NextCharacter *int               `json:"next_character,omitempty"`
	Columns       []string           `json:"columns,omitempty"`
	Rows          [][]any            `json:"rows,omitempty"`
}

func mapTaskStatus(status string) string {
	if status == "Bastidores" {
		return "Backlog"
	}
	if status == "Em Pauta" {
		return "Em Estruturação"
	}
	return status
}
func getArgs(request mcp.CallToolRequest) map[string]any {
	args, _ := request.Params.Arguments.(map[string]any)
	return args
}
func getStr(args map[string]any, key string) string    { value, _ := args[key].(string); return value }
func getFloat(args map[string]any, key string) float64 { value, _ := args[key].(float64); return value }

func adminFieldSchema(field string, output bool) map[string]any {
	typ := "string"
	props := map[string]any{}
	if field == "id" || strings.HasSuffix(field, "_id") || field == "order_index" {
		typ = "integer"
		if !output {
			props["minimum"] = 0
			if field != "order_index" {
				props["minimum"] = 1
			}
			props["maximum"] = 9007199254740991
		}
	}
	if field == "amount" || field == "progress_percentage" || field == "percentual" {
		typ = "number"
		if !output {
			props["minimum"] = 0
			if field != "amount" {
				props["maximum"] = 100
			}
		}
	}
	if field == "is_admin" || field == "client_approved" {
		typ = "boolean"
		if output {
			typ = "integer"
		}
	}
	if typ == "string" && !output {
		props["maxLength"] = 16000
	}
	if output {
		props["type"] = []string{typ, "null"}
	} else {
		props["type"] = typ
	}
	return props
}
func adminInputSchema(e adminEntity) json.RawMessage {
	p := map[string]any{"acao": map[string]any{"type": "string", "enum": e.actions}, "id": adminFieldSchema("id", false), "limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 50, "default": 20}, "offset": map[string]any{"type": "integer", "minimum": 0, "maximum": 1000000, "default": 0}}
	for _, f := range strings.Split(e.editable, ",") {
		p[f] = adminFieldSchema(f, false)
	}
	if e.entity == "note" {
		delete(p, "content")
		p["texto"] = adminFieldSchema("texto", false)
		p["busca"] = adminFieldSchema("busca", false)
	}
	if e.entity == "task" {
		p["status"] = map[string]any{"type": "string", "enum": []string{"Bastidores", "Em Pauta", "Em Produção", "No Ar", "Backlog", "Em Estruturação"}}
	}
	if e.entity == "payment_installment" {
		p["status"] = map[string]any{"type": "string", "enum": []string{"Pendente", "Pago", "Atrasado"}}
	}
	if e.entity == "project_member" {
		p["role"] = map[string]any{"type": "string", "enum": []string{"owner", "viewer", "hidden"}}
	}
	b, _ := json.Marshal(map[string]any{"type": "object", "properties": p, "required": []string{"acao"}, "additionalProperties": false})
	return b
}
func adminOutputSchema(fields string) json.RawMessage {
	props := map[string]any{"success": map[string]any{"type": "boolean"}, "id": map[string]any{"type": "integer"}, "next_offset": map[string]any{"type": "integer"}, "next_character": map[string]any{"type": "integer"}, "text": map[string]any{"type": "string"}, "pages": map[string]any{"type": "array", "items": map[string]any{"type": "string"}}, "info": map[string]any{"type": "object", "properties": map[string]any{"changes": map[string]any{"type": "integer"}, "lastInsertRowid": map[string]any{"type": "integer"}}, "required": []string{"changes", "lastInsertRowid"}, "additionalProperties": false}}
	row := map[string]any{}
	for _, f := range strings.Split(fields, ",") {
		if f != "" {
			row[f] = adminFieldSchema(f, true)
		}
	}
	props["results"] = map[string]any{"type": "array", "maxItems": 50, "items": map[string]any{"type": "object", "properties": row, "additionalProperties": false}}
	scalar := map[string]any{"type": []string{"string", "number", "boolean", "null"}}
	props["columns"] = map[string]any{"type": "array", "items": map[string]any{"type": "string"}}
	props["rows"] = map[string]any{"type": "array", "maxItems": 50, "items": map[string]any{"type": "array", "items": scalar}}
	b, _ := json.Marshal(map[string]any{"type": "object", "properties": props, "required": []string{"success"}, "additionalProperties": false})
	return b
}

// Validation is also applied here, so direct handler calls cannot bypass the declared contract.
func adminValidate(args map[string]any, schema json.RawMessage) error {
	var contract struct {
		Properties map[string]map[string]any `json:"properties"`
		Required   []string                  `json:"required"`
	}
	if err := json.Unmarshal(schema, &contract); err != nil {
		return err
	}
	for _, key := range contract.Required {
		if _, ok := args[key]; !ok {
			return fmt.Errorf("%s é obrigatório", key)
		}
	}
	for key, value := range args {
		rule, ok := contract.Properties[key]
		if !ok {
			return fmt.Errorf("campo inválido: %s", key)
		}
		switch rule["type"] {
		case "integer", "number":
			n, ok := value.(float64)
			if !ok || math.IsNaN(n) || math.IsInf(n, 0) || (rule["type"] == "integer" && math.Trunc(n) != n) {
				return fmt.Errorf("%s deve ser %s", key, rule["type"])
			}
			if min, ok := rule["minimum"].(float64); ok && n < min {
				return fmt.Errorf("%s abaixo do mínimo", key)
			}
			if max, ok := rule["maximum"].(float64); ok && n > max {
				return fmt.Errorf("%s acima do máximo", key)
			}
		case "boolean":
			if _, ok := value.(bool); !ok {
				return fmt.Errorf("%s deve ser boolean", key)
			}
		case "string":
			s, ok := value.(string)
			if !ok {
				return fmt.Errorf("%s deve ser string", key)
			}
			if max, ok := rule["maxLength"].(float64); ok && utf8.RuneCountInString(s) > int(max) {
				return fmt.Errorf("%s muito longo", key)
			}
		}
		if enums, ok := rule["enum"].([]any); ok {
			valid := false
			for _, allowed := range enums {
				if allowed == value {
					valid = true
				}
			}
			if !valid {
				return fmt.Errorf("%s inválido", key)
			}
		}
	}
	return nil
}
func adminPage(args map[string]any) (int, int) {
	limit, offset := 20, 0
	if n, ok := args["limit"].(float64); ok {
		limit = int(n)
	}
	if n, ok := args["offset"].(float64); ok {
		offset = int(n)
	}
	return limit, offset
}
func adminRequireID(args map[string]any, key string) error {
	n, ok := args[key].(float64)
	if !ok || n < 1 {
		return fmt.Errorf("%s é obrigatório", key)
	}
	return nil
}
func adminDecode(raw string) (adminAPIResponse, error) {
	var response adminAPIResponse
	err := json.Unmarshal([]byte(raw), &response)
	if err != nil {
		return response, fmt.Errorf("resposta inválida do portal")
	}
	if response.Error != "" {
		return response, fmt.Errorf("%s", response.Error)
	}
	return response, nil
}
func adminResult(value adminOutput) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{Content: []mcp.Content{}, StructuredContent: value}, nil
}
func adminError(err error) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultError(err.Error()), nil
}

func RegisterAllTools(s *server.MCPServer, cfg *Config) {
	for _, entity := range adminEntities {
		e := entity
		if !slicesContains(e.actions, "buscar") {
			e.actions = append(append([]string{}, e.actions...), "buscar")
		}
		schema := adminInputSchema(e)
		tool := adminNewTool(e.name, mcp.WithDescription("Administração de "+e.table+". Listas paginadas (20 por padrão, máximo50); use buscar para ficha."), mcp.WithRawInputSchema(schema), mcp.WithRawOutputSchema(adminOutputSchema(e.fields)))
		s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := getArgs(request)
			if err := adminValidate(args, schema); err != nil {
				return adminError(err)
			}
			return adminEntityCall(ctx, cfg, e, args)
		})
	}
	registerAdminSQL(s, cfg)
	registerAdminProposal(s, cfg)
}
func adminEntityCall(ctx context.Context, cfg *Config, e adminEntity, args map[string]any) (*mcp.CallToolResult, error) {
	action := getStr(args, "acao")
	limit, offset := adminPage(args)
	if action == "listar" || action == "buscar" {
		projection := e.fields
		table := e.table
		if e.entity == "project" {
			projection = "p.id,p.client_id,p.name,p.scope_summary,p.progress_percentage,p.next_payment_date,p.payment_status,p.invoice_link,p.github_repo,c.name AS client_name"
			table = "projects p JOIN clients c ON c.id=p.client_id"
		}
		if e.entity == "project_member" {
			projection = "pm.id,pm.project_id,pm.client_id,pm.role,pm.added_at,c.name,c.email"
			table = "project_members pm JOIN clients c ON c.id=pm.client_id"
		}
		if action == "listar" {
			for _, field := range []string{"description", "content", "scope_summary", "notice"} {
				col := field
				if e.entity == "project" {
					col = "p." + field
				}
				projection = strings.ReplaceAll(projection, col, "substr("+col+",1,240) AS "+field)
			}
		}
		q := "SELECT " + projection + " FROM " + table
		where := []string{}
		params := []any{}
		if action == "buscar" {
			if err := adminRequireID(args, "id"); err != nil {
				return adminError(err)
			}
			where = append(where, adminColumn(e, "id")+"=?")
			params = append(params, args["id"])
			limit = 1
			offset = 0
		} else if e.parent != "" {
			if e.entity != "project" && e.entity != "note" {
				if err := adminRequireID(args, e.parent); err != nil {
					return adminError(err)
				}
			}
			if v, ok := args[e.parent]; ok {
				where = append(where, adminColumn(e, e.parent)+"=?")
				params = append(params, v)
			}
		}
		if e.entity == "task" {
			if status := getStr(args, "status"); status != "" {
				where = append(where, "status=?")
				params = append(params, mapTaskStatus(status))
			}
		}
		if e.entity == "note" {
			if search := getStr(args, "busca"); search != "" {
				where = append(where, "content LIKE ?")
				params = append(params, "%"+search+"%")
			}
		}
		if len(where) > 0 {
			q += " WHERE " + strings.Join(where, " AND ")
		}
		order := e.order
		if e.entity == "project" {
			order = "p.id DESC"
		}
		if e.entity == "project_member" {
			order = "pm.id DESC"
		}
		q += " ORDER BY " + order + " LIMIT ? OFFSET ?"
		params = append(params, limit+1, offset)
		raw, err := CallQueryParams(ctx, cfg, q, params)
		if err != nil {
			return adminError(err)
		}
		response, err := adminDecode(raw)
		if err != nil {
			return adminError(err)
		}
		rows := response.Results
		if rows == nil {
			rows = []map[string]any{}
		}
		var next *int
		if len(rows) > limit {
			rows = rows[:limit]
			n := offset + limit
			next = &n
		}
		// Defense in depth against future backend SELECT changes or unexpectedly added columns.
		allowed := map[string]bool{}
		for _, f := range strings.Split(e.fields, ",") {
			allowed[f] = true
		}
		for _, row := range rows {
			for key := range row {
				if !allowed[key] {
					delete(row, key)
				}
			}
		}
		return adminResult(adminOutput{Success: response.Success, Results: &rows, NextOffset: next})
	}
	mapped := map[string]string{"criar": "create", "editar": "update", "excluir": "delete", "adicionar": "create", "remover": "delete"}[action]
	if mapped == "" {
		return adminError(fmt.Errorf("ação inválida"))
	}
	if action == "editar" || action == "excluir" {
		if err := adminRequireID(args, "id"); err != nil {
			return adminError(err)
		}
	}
	if (action == "criar" || action == "adicionar" || action == "remover") && e.parent != "" && e.entity != "note" {
		if err := adminRequireID(args, e.parent); err != nil {
			return adminError(err)
		}
	}
	if e.entity == "project_member" {
		if err := adminRequireID(args, "client_id"); err != nil {
			return adminError(err)
		}
	}
	data := map[string]any{}
	for _, field := range strings.Split(e.editable, ",") {
		if value, ok := args[field]; ok {
			data[field] = value
		}
	}
	if value, ok := args["id"]; ok {
		data["id"] = value
	}
	if e.entity == "note" {
		if text, ok := args["texto"]; ok {
			data["content"] = text
		}
	}
	if e.entity == "task" {
		if status, ok := data["status"].(string); ok {
			data["status"] = mapTaskStatus(status)
		}
		if action == "criar" {
			if _, ok := data["status"]; !ok {
				data["status"] = "Backlog"
			}
		}
	}
	if e.entity == "changelog" {
		if value, ok := data["media_links"].(string); ok {
			links := []string{}
			for _, link := range strings.Split(value, ",") {
				if link = strings.TrimSpace(link); link != "" {
					links = append(links, link)
				}
			}
			data["media_links"] = links
		}
	}
	raw, err := CallAction(ctx, cfg, e.entity, mapped, data)
	if err != nil {
		return adminError(err)
	}
	response, err := adminDecode(raw)
	if err != nil {
		return adminError(err)
	}
	return adminResult(adminOutput{Success: response.Success, ID: response.ID, Info: response.Info})
}
func registerAdminSQL(s *server.MCPServer, cfg *Config) {
	schema := json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","minLength":1,"maxLength":16000},"limit":{"type":"integer","minimum":1,"maximum":50,"default":20},"offset":{"type":"integer","minimum":0,"maximum":1000000}},"required":["query"],"additionalProperties":false}`)
	s.AddTool(adminNewTool("consulta_sql", mcp.WithDescription("SQL administrativo avançado. Resultados tabulares paginados; jamais disponível a vendedores."), mcp.WithRawInputSchema(schema), mcp.WithRawOutputSchema(adminOutputSchema(""))), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(request)
		if err := adminValidate(args, schema); err != nil {
			return adminError(err)
		}
		query := strings.TrimSpace(getStr(args, "query"))
		limit, offset := adminPage(args)
		// Wrapping read-only SELECT bounds database work too, without rewriting user expressions.
		upper := strings.ToUpper(query)
		first := strings.Fields(upper)
		if len(first) > 0 && (first[0] == "SELECT" || first[0] == "WITH") {
			query = "SELECT * FROM (" + strings.TrimSuffix(query, ";") + ") LIMIT " + strconv.Itoa(limit+1) + " OFFSET " + strconv.Itoa(offset)
		}
		raw, err := CallQuery(ctx, cfg, query)
		if err != nil {
			return adminError(err)
		}
		response, err := adminDecode(raw)
		if err != nil {
			return adminError(err)
		}
		out := adminOutput{Success: response.Success, Info: response.Info}
		if response.Results != nil {
			columns := []string{}
			seen := map[string]bool{}
			for _, row := range response.Results {
				for key := range row {
					if !seen[key] {
						columns = append(columns, key)
						seen[key] = true
					}
				}
			}
			sort.Strings(columns)
			rows := response.Results
			if len(rows) > limit {
				rows = rows[:limit]
				n := offset + limit
				out.NextOffset = &n
			}
			out.Columns = columns
			out.Rows = [][]any{}
			for _, row := range rows {
				values := []any{}
				for _, key := range columns {
					value := row[key]
					switch value.(type) {
					case nil, string, float64, bool:
					default:
						value = fmt.Sprint(value)
					}
					values = append(values, value)
				}
				out.Rows = append(out.Rows, values)
			}
		}
		return adminResult(out)
	})
}
func registerAdminProposal(s *server.MCPServer, cfg *Config) {
	schema := json.RawMessage(`{"type":"object","properties":{"pagina":{"type":"string","maxLength":200},"limit":{"type":"integer","minimum":1,"maximum":50,"default":20},"offset":{"type":"integer","minimum":0,"maximum":1000000},"character":{"type":"integer","minimum":0,"maximum":10000000}},"additionalProperties":false}`)
	s.AddTool(adminNewTool("ver_proposta", mcp.WithDescription("Lista propostas ou lê até8000caracteres por chamada; continue com next_character."), mcp.WithRawInputSchema(schema), mcp.WithRawOutputSchema(adminOutputSchema(""))), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(request)
		if err := adminValidate(args, schema); err != nil {
			return adminError(err)
		}
		page := strings.Trim(strings.TrimPrefix(strings.Trim(getStr(args, "pagina"), "/"), "propostas/"), "/")
		limit, offset := adminPage(args)
		if page == "" {
			raw, err := CallAction(ctx, cfg, "proposal", "list", nil)
			if err != nil {
				return adminError(err)
			}
			response, err := adminDecode(raw)
			if err != nil {
				return adminError(err)
			}
			pages := response.Pages
			if offset > len(pages) {
				offset = len(pages)
			}
			pages = pages[offset:]
			var next *int
			if len(pages) > limit {
				pages = pages[:limit]
				n := offset + limit
				next = &n
			}
			return adminResult(adminOutput{Success: response.Success, Pages: pages, NextOffset: next})
		}
		if strings.Contains(page, "..") || strings.ContainsAny(page, "?#\\") {
			return adminError(fmt.Errorf("página inválida"))
		}
		text, err := GetProposalText(ctx, cfg, page)
		if err != nil {
			return adminError(err)
		}
		runes := []rune(text)
		start := 0
		if value, ok := args["character"].(float64); ok {
			start = int(value)
		}
		if start > len(runes) {
			start = len(runes)
		}
		end := start + 8000
		var next *int
		if end < len(runes) {
			next = &end
		} else {
			end = len(runes)
		}
		return adminResult(adminOutput{Success: true, Text: string(runes[start:end]), NextCharacter: next})
	})
}

func adminNewTool(name string, options ...mcp.ToolOption) mcp.Tool {
	tool := mcp.NewTool(name, options...)
	tool.InputSchema = mcp.ToolInputSchema{}
	return tool
}

func slicesContains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func adminColumn(e adminEntity, column string) string {
	if e.entity == "project" {
		return "p." + column
	}
	if e.entity == "project_member" {
		return "pm." + column
	}
	return column
}
