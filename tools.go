package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Helpers para extrair argumentos de forma segura
func getStr(args map[string]any, key string) string {
	v, _ := args[key].(string)
	return v
}

func getFloat(args map[string]any, key string) float64 {
	v, _ := args[key].(float64)
	return v
}

func getArgs(request mcp.CallToolRequest) map[string]any {
	args, _ := request.Params.Arguments.(map[string]any)
	return args
}

// mapTaskStatus converte os nomes de estágio exibidos no frontend
// para os valores armazenados no banco de dados.
func mapTaskStatus(status string) string {
	switch status {
	case "Bastidores":
		return "Backlog"
	case "Em Pauta":
		return "Em Estruturação"
	default:
		return status
	}
}

// RegisterAllTools registra todas as ferramentas agrupadas por entidade.
func RegisterAllTools(s *server.MCPServer, cfg *Config) {

	// ══════════════════════════════════════════════════════════════
	// 1. GERENCIAR CLIENTES
	// ══════════════════════════════════════════════════════════════
	s.AddTool(mcp.NewTool("gerenciar_clientes",
		mcp.WithDescription("Gerencia clientes do sistema. Ações: listar, buscar, criar, editar, excluir."),
		mcp.WithString("acao", mcp.Required(), mcp.Description("Ação a executar: listar | buscar | criar | editar | excluir")),
		mcp.WithString("id", mcp.Description("ID do cliente (para buscar, editar ou excluir)")),
		mcp.WithString("name", mcp.Description("Nome do cliente")),
		mcp.WithString("email", mcp.Description("E-mail do cliente")),
		mcp.WithString("avatar_url", mcp.Description("URL do avatar")),
		mcp.WithString("is_admin", mcp.Description("'true' para tornar admin, 'false' para remover")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		if args == nil { return mcp.NewToolResultError("argumentos ausentes"), nil }
		acao := getStr(args, "acao")
		switch acao {
		case "listar":
			r, err := CallQuery(ctx, cfg, "SELECT id, name, email, is_admin, avatar_url FROM clients")
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "buscar":
			id := getStr(args, "id")
			if id == "" { return mcp.NewToolResultError("id é obrigatório"), nil }
			r, err := CallQuery(ctx, cfg, fmt.Sprintf("SELECT * FROM clients WHERE id = %s", id))
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "criar":
			data := map[string]any{"name": getStr(args, "name"), "email": getStr(args, "email")}
			if v := getStr(args, "avatar_url"); v != "" { data["avatar_url"] = v }
			r, err := CallAction(ctx, cfg, "client", "create", data)
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "editar":
			data := map[string]any{"id": getStr(args, "id")}
			if v := getStr(args, "name"); v != "" { data["name"] = v }
			if v := getStr(args, "email"); v != "" { data["email"] = v }
			if v := getStr(args, "avatar_url"); v != "" { data["avatar_url"] = v }
			if v := getStr(args, "is_admin"); v == "true" { data["is_admin"] = true } else if v == "false" { data["is_admin"] = false }
			r, err := CallAction(ctx, cfg, "client", "update", data)
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "excluir":
			r, err := CallAction(ctx, cfg, "client", "delete", map[string]any{"id": getStr(args, "id")})
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		default:
			return mcp.NewToolResultError("ação inválida: " + acao), nil
		}
	})

	// ══════════════════════════════════════════════════════════════
	// 2. GERENCIAR PROJETOS
	// ══════════════════════════════════════════════════════════════
	s.AddTool(mcp.NewTool("gerenciar_projetos",
		mcp.WithDescription("Gerencia projetos. Ações: listar, buscar, criar, editar, excluir."),
		mcp.WithString("acao", mcp.Required(), mcp.Description("Ação: listar | buscar | criar | editar | excluir")),
		mcp.WithString("id", mcp.Description("ID do projeto")),
		mcp.WithString("client_id", mcp.Description("ID do cliente dono (para criar ou filtrar)")),
		mcp.WithString("name", mcp.Description("Nome do projeto")),
		mcp.WithString("scope_summary", mcp.Description("Resumo do escopo")),
		mcp.WithString("progress_percentage", mcp.Description("Progresso (0-100)")),
		mcp.WithString("next_payment_date", mcp.Description("Data do próximo pagamento")),
		mcp.WithString("payment_status", mcp.Description("Status do pagamento")),
		mcp.WithString("invoice_link", mcp.Description("Link da fatura")),
		mcp.WithString("github_repo", mcp.Description("Repositório GitHub")),
		mcp.WithString("github_token", mcp.Description("Token GitHub")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		if args == nil { return mcp.NewToolResultError("argumentos ausentes"), nil }
		acao := getStr(args, "acao")
		switch acao {
		case "listar":
			q := "SELECT p.*, c.name as client_name FROM projects p JOIN clients c ON p.client_id = c.id"
			if cid := getStr(args, "client_id"); cid != "" {
				q += fmt.Sprintf(" WHERE p.client_id = %s", cid)
			}
			r, err := CallQuery(ctx, cfg, q)
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "buscar":
			id := getStr(args, "id")
			if id == "" { return mcp.NewToolResultError("id é obrigatório"), nil }
			r, err := CallQuery(ctx, cfg, fmt.Sprintf("SELECT p.*, c.name as client_name FROM projects p JOIN clients c ON p.client_id = c.id WHERE p.id = %s", id))
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "criar":
			data := map[string]any{"client_id": getStr(args, "client_id"), "name": getStr(args, "name")}
			if v := getStr(args, "scope_summary"); v != "" { data["scope_summary"] = v }
			r, err := CallAction(ctx, cfg, "project", "create", data)
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "editar":
			data := map[string]any{"id": getStr(args, "id")}
			if v := getStr(args, "name"); v != "" { data["name"] = v }
			if v := getStr(args, "scope_summary"); v != "" { data["scope_summary"] = v }
			if v := getStr(args, "progress_percentage"); v != "" { data["progress_percentage"] = v }
			if v := getStr(args, "next_payment_date"); v != "" { data["next_payment_date"] = v }
			if v := getStr(args, "payment_status"); v != "" { data["payment_status"] = v }
			if v := getStr(args, "invoice_link"); v != "" { data["invoice_link"] = v }
			if v := getStr(args, "github_repo"); v != "" { data["github_repo"] = v }
			if v := getStr(args, "github_token"); v != "" { data["github_token"] = v }
			r, err := CallAction(ctx, cfg, "project", "update", data)
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "excluir":
			r, err := CallAction(ctx, cfg, "project", "delete", map[string]any{"id": getStr(args, "id")})
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		default:
			return mcp.NewToolResultError("ação inválida: " + acao), nil
		}
	})

	// ══════════════════════════════════════════════════════════════
	// 3. GERENCIAR ENTREGAS
	// ══════════════════════════════════════════════════════════════
	s.AddTool(mcp.NewTool("gerenciar_entregas",
		mcp.WithDescription("Gerencia entregas do roadmap de um projeto. Ações: listar, buscar, criar, editar, excluir."),
		mcp.WithString("acao", mcp.Required(), mcp.Description("Ação: listar | buscar | criar | editar | excluir")),
		mcp.WithString("id", mcp.Description("ID da entrega")),
		mcp.WithString("project_id", mcp.Description("ID do projeto")),
		mcp.WithString("title", mcp.Description("Título da entrega")),
		mcp.WithString("status", mcp.Description("Status da entrega: Bastidores | Em Pauta | Em Produção | No Ar")),
		mcp.WithString("description", mcp.Description("Descrição detalhada")),
		mcp.WithString("media_url", mcp.Description("URL de mídia (imagem/vídeo)")),
		mcp.WithString("due_date", mcp.Description("Data limite (YYYY-MM-DD)")),
		mcp.WithString("order_index", mcp.Description("Índice de ordenação")),
		mcp.WithString("client_approved", mcp.Description("'true' se aprovado pelo cliente")),
		mcp.WithString("implemented_at", mcp.Description("Data de implementação")),
		mcp.WithString("percentual", mcp.Description("Percentual de execução da entrega (0-100)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		if args == nil { return mcp.NewToolResultError("argumentos ausentes"), nil }
		acao := getStr(args, "acao")
		switch acao {
		case "listar":
			pid := getStr(args, "project_id")
			if pid == "" { return mcp.NewToolResultError("project_id é obrigatório"), nil }
			q := fmt.Sprintf("SELECT * FROM tasks WHERE project_id = %s ORDER BY order_index", pid)
			if st := getStr(args, "status"); st != "" {
				q = fmt.Sprintf("SELECT * FROM tasks WHERE project_id = %s AND status = '%s' ORDER BY order_index", pid, mapTaskStatus(st))
			}
			r, err := CallQuery(ctx, cfg, q)
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "buscar":
			r, err := CallQuery(ctx, cfg, fmt.Sprintf("SELECT * FROM tasks WHERE id = %s", getStr(args, "id")))
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "criar":
			data := map[string]any{
				"project_id": getStr(args, "project_id"),
				"title":      getStr(args, "title"),
				"status":     mapTaskStatus(getStr(args, "status")),
			}
			for _, k := range []string{"description", "media_url", "due_date", "order_index", "percentual"} {
				if v := getStr(args, k); v != "" { data[k] = v }
			}
			if v := getStr(args, "client_approved"); v == "true" { data["client_approved"] = true } else if v == "false" { data["client_approved"] = false }
			r, err := CallAction(ctx, cfg, "task", "create", data)
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "editar":
			data := map[string]any{"id": getStr(args, "id")}
			for _, k := range []string{"project_id", "title", "description", "media_url", "due_date", "order_index", "implemented_at", "percentual"} {
				if v := getStr(args, k); v != "" { data[k] = v }
			}
			if v := getStr(args, "status"); v != "" { data["status"] = mapTaskStatus(v) }
			if v := getStr(args, "client_approved"); v == "true" { data["client_approved"] = true } else if v == "false" { data["client_approved"] = false }
			r, err := CallAction(ctx, cfg, "task", "update", data)
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "excluir":
			r, err := CallAction(ctx, cfg, "task", "delete", map[string]any{"id": getStr(args, "id")})
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		default:
			return mcp.NewToolResultError("ação inválida: " + acao), nil
		}
	})

	// ══════════════════════════════════════════════════════════════
	// 4. GERENCIAR PARCELAS
	// ══════════════════════════════════════════════════════════════
	s.AddTool(mcp.NewTool("gerenciar_parcelas",
		mcp.WithDescription("Gerencia parcelas de pagamento. Ações: listar, criar, editar, excluir."),
		mcp.WithString("acao", mcp.Required(), mcp.Description("Ação: listar | criar | editar | excluir")),
		mcp.WithString("id", mcp.Description("ID da parcela")),
		mcp.WithString("project_id", mcp.Description("ID do projeto")),
		mcp.WithString("label", mcp.Description("Rótulo da parcela")),
		mcp.WithNumber("amount", mcp.Description("Valor da parcela")),
		mcp.WithString("status", mcp.Description("Status: Pendente | Pago | Atrasado")),
		mcp.WithString("due_date", mcp.Description("Data de vencimento")),
		mcp.WithNumber("order_index", mcp.Description("Índice de ordenação")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		if args == nil { return mcp.NewToolResultError("argumentos ausentes"), nil }
		acao := getStr(args, "acao")
		switch acao {
		case "listar":
			r, err := CallQuery(ctx, cfg, fmt.Sprintf("SELECT * FROM payment_installments WHERE project_id = %s ORDER BY order_index", getStr(args, "project_id")))
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "criar":
			data := map[string]any{"project_id": getStr(args, "project_id")}
			if v := getStr(args, "label"); v != "" { data["label"] = v }
			if v := getFloat(args, "amount"); v != 0 { data["amount"] = v }
			if v := getStr(args, "status"); v != "" { data["status"] = v }
			if v := getStr(args, "due_date"); v != "" { data["due_date"] = v }
			if v := getFloat(args, "order_index"); v != 0 { data["order_index"] = v }
			r, err := CallAction(ctx, cfg, "payment_installment", "create", data)
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "editar":
			data := map[string]any{"id": getStr(args, "id")}
			if v := getStr(args, "label"); v != "" { data["label"] = v }
			if v := getFloat(args, "amount"); v != 0 { data["amount"] = v }
			if v := getStr(args, "status"); v != "" { data["status"] = v }
			if v := getStr(args, "due_date"); v != "" { data["due_date"] = v }
			if v := getFloat(args, "order_index"); v != 0 { data["order_index"] = v }
			r, err := CallAction(ctx, cfg, "payment_installment", "update", data)
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "excluir":
			r, err := CallAction(ctx, cfg, "payment_installment", "delete", map[string]any{"id": getStr(args, "id")})
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		default:
			return mcp.NewToolResultError("ação inválida: " + acao), nil
		}
	})

	// ══════════════════════════════════════════════════════════════
	// 5. GERENCIAR AVISOS
	// ══════════════════════════════════════════════════════════════
	s.AddTool(mcp.NewTool("gerenciar_avisos",
		mcp.WithDescription("Gerencia avisos rápidos de um projeto. Ações: listar, criar, editar, excluir."),
		mcp.WithString("acao", mcp.Required(), mcp.Description("Ação: listar | criar | editar | excluir")),
		mcp.WithString("id", mcp.Description("ID do aviso")),
		mcp.WithString("project_id", mcp.Description("ID do projeto")),
		mcp.WithString("notice", mcp.Description("Texto do aviso")),
		mcp.WithString("url", mcp.Description("URL do link do aviso (opcional)")),
		mcp.WithString("link_label", mcp.Description("Texto do botão/link do aviso (opcional)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		if args == nil { return mcp.NewToolResultError("argumentos ausentes"), nil }
		acao := getStr(args, "acao")
		switch acao {
		case "listar":
			r, err := CallQuery(ctx, cfg, fmt.Sprintf("SELECT * FROM quick_notices WHERE project_id = %s", getStr(args, "project_id")))
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "criar":
			data := map[string]any{"project_id": getStr(args, "project_id"), "notice": getStr(args, "notice")}
			if v := getStr(args, "url"); v != "" { data["url"] = v }
			if v := getStr(args, "link_label"); v != "" { data["link_label"] = v }
			r, err := CallAction(ctx, cfg, "notice", "create", data)
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "editar":
			data := map[string]any{"id": getStr(args, "id"), "notice": getStr(args, "notice")}
			if v := getStr(args, "url"); v != "" { data["url"] = v }
			if v := getStr(args, "link_label"); v != "" { data["link_label"] = v }
			r, err := CallAction(ctx, cfg, "notice", "update", data)
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "excluir":
			r, err := CallAction(ctx, cfg, "notice", "delete", map[string]any{"id": getStr(args, "id")})
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		default:
			return mcp.NewToolResultError("ação inválida: " + acao), nil
		}
	})

	// ══════════════════════════════════════════════════════════════
	// 6. GERENCIAR DOCUMENTOS
	// ══════════════════════════════════════════════════════════════
	s.AddTool(mcp.NewTool("gerenciar_documentos",
		mcp.WithDescription("Gerencia documentos e links úteis do projeto. Ações: listar, criar, editar, excluir."),
		mcp.WithString("acao", mcp.Required(), mcp.Description("Ação: listar | criar | editar | excluir")),
		mcp.WithString("id", mcp.Description("ID do documento")),
		mcp.WithString("project_id", mcp.Description("ID do projeto")),
		mcp.WithString("title", mcp.Description("Título do documento")),
		mcp.WithString("url", mcp.Description("URL do documento")),
		mcp.WithString("type", mcp.Description("Tipo do documento: Contrato | Proposta | Relatório | Escopo | Outro")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		if args == nil { return mcp.NewToolResultError("argumentos ausentes"), nil }
		acao := getStr(args, "acao")
		switch acao {
		case "listar":
			r, err := CallQuery(ctx, cfg, fmt.Sprintf("SELECT * FROM documents WHERE project_id = %s", getStr(args, "project_id")))
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "criar":
			r, err := CallAction(ctx, cfg, "document", "create", map[string]any{"project_id": getStr(args, "project_id"), "title": getStr(args, "title"), "url": getStr(args, "url"), "type": getStr(args, "type")})
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "editar":
			data := map[string]any{"id": getStr(args, "id")}
			for _, k := range []string{"title", "url", "type"} {
				if v := getStr(args, k); v != "" { data[k] = v }
			}
			r, err := CallAction(ctx, cfg, "document", "update", data)
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "excluir":
			r, err := CallAction(ctx, cfg, "document", "delete", map[string]any{"id": getStr(args, "id")})
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		default:
			return mcp.NewToolResultError("ação inválida: " + acao), nil
		}
	})

	// ══════════════════════════════════════════════════════════════
	// 7. GERENCIAR MARCOS
	// ══════════════════════════════════════════════════════════════
	s.AddTool(mcp.NewTool("gerenciar_marcos",
		mcp.WithDescription("Gerencia marcos/milestones do projeto. Ações: listar, criar, editar, excluir."),
		mcp.WithString("acao", mcp.Required(), mcp.Description("Ação: listar | criar | editar | excluir")),
		mcp.WithString("id", mcp.Description("ID do marco")),
		mcp.WithString("project_id", mcp.Description("ID do projeto")),
		mcp.WithString("title", mcp.Description("Título do marco")),
		mcp.WithString("due_date", mcp.Description("Data limite")),
		mcp.WithString("type", mcp.Description("Tipo do marco: Entrega | Cliente | Reunião")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		if args == nil { return mcp.NewToolResultError("argumentos ausentes"), nil }
		acao := getStr(args, "acao")
		switch acao {
		case "listar":
			r, err := CallQuery(ctx, cfg, fmt.Sprintf("SELECT * FROM milestones WHERE project_id = %s", getStr(args, "project_id")))
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "criar":
			r, err := CallAction(ctx, cfg, "milestone", "create", map[string]any{"project_id": getStr(args, "project_id"), "title": getStr(args, "title"), "due_date": getStr(args, "due_date"), "type": getStr(args, "type")})
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "editar":
			data := map[string]any{"id": getStr(args, "id")}
			for _, k := range []string{"title", "due_date", "type"} {
				if v := getStr(args, k); v != "" { data[k] = v }
			}
			r, err := CallAction(ctx, cfg, "milestone", "update", data)
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "excluir":
			r, err := CallAction(ctx, cfg, "milestone", "delete", map[string]any{"id": getStr(args, "id")})
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		default:
			return mcp.NewToolResultError("ação inválida: " + acao), nil
		}
	})

	// ══════════════════════════════════════════════════════════════
	// 8. GERENCIAR DIÁRIO DO PROJETO
	// ══════════════════════════════════════════════════════════════
	s.AddTool(mcp.NewTool("gerenciar_diario",
		mcp.WithDescription("Gerencia o diário do projeto (atualizações e histórico de entregas). Ações: listar, criar, editar, excluir."),
		mcp.WithString("acao", mcp.Required(), mcp.Description("Ação: listar | criar | editar | excluir")),
		mcp.WithString("id", mcp.Description("ID do registro")),
		mcp.WithString("project_id", mcp.Description("ID do projeto")),
		mcp.WithString("description", mcp.Description("Descrição da atualização")),
		mcp.WithString("date", mcp.Description("Data (YYYY-MM-DD)")),
		mcp.WithString("media_links", mcp.Description("Links de mídia (Google Drive) separados por vírgula")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		if args == nil { return mcp.NewToolResultError("argumentos ausentes"), nil }
		acao := getStr(args, "acao")
		switch acao {
		case "listar":
			r, err := CallQuery(ctx, cfg, fmt.Sprintf("SELECT * FROM changelog WHERE project_id = %s ORDER BY date DESC", getStr(args, "project_id")))
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "criar":
			data := map[string]any{"project_id": getStr(args, "project_id"), "description": getStr(args, "description")}
			if v := getStr(args, "date"); v != "" { data["date"] = v }
			if v := getStr(args, "media_links"); v != "" {
				links := strings.Split(v, ",")
				for i := range links { links[i] = strings.TrimSpace(links[i]) }
				data["media_links"] = links
			}
			r, err := CallAction(ctx, cfg, "changelog", "create", data)
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "editar":
			data := map[string]any{"id": getStr(args, "id")}
			if v := getStr(args, "description"); v != "" { data["description"] = v }
			if v := getStr(args, "date"); v != "" { data["date"] = v }
			if v := getStr(args, "media_links"); v != "" {
				links := strings.Split(v, ",")
				for i := range links { links[i] = strings.TrimSpace(links[i]) }
				data["media_links"] = links
			}
			r, err := CallAction(ctx, cfg, "changelog", "update", data)
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "excluir":
			r, err := CallAction(ctx, cfg, "changelog", "delete", map[string]any{"id": getStr(args, "id")})
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		default:
			return mcp.NewToolResultError("ação inválida: " + acao), nil
		}
	})

	// ══════════════════════════════════════════════════════════════
	// 9. GERENCIAR MEMBROS
	// ══════════════════════════════════════════════════════════════
	s.AddTool(mcp.NewTool("gerenciar_membros",
		mcp.WithDescription("Gerencia membros de um projeto. Ações: listar, adicionar, remover."),
		mcp.WithString("acao", mcp.Required(), mcp.Description("Ação: listar | adicionar | remover")),
		mcp.WithString("project_id", mcp.Description("ID do projeto")),
		mcp.WithString("client_id", mcp.Description("ID do cliente")),
		mcp.WithString("role", mcp.Description("Papel: owner (Responsável) | viewer (Participante) | hidden (Oculto)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		if args == nil { return mcp.NewToolResultError("argumentos ausentes"), nil }
		acao := getStr(args, "acao")
		switch acao {
		case "listar":
			r, err := CallQuery(ctx, cfg, fmt.Sprintf("SELECT pm.*, c.name, c.email FROM project_members pm JOIN clients c ON pm.client_id = c.id WHERE pm.project_id = %s", getStr(args, "project_id")))
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "adicionar":
			data := map[string]any{"project_id": getStr(args, "project_id"), "client_id": getStr(args, "client_id")}
			if v := getStr(args, "role"); v != "" { data["role"] = v }
			r, err := CallAction(ctx, cfg, "project_member", "create", data)
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "remover":
			r, err := CallAction(ctx, cfg, "project_member", "delete", map[string]any{"project_id": getStr(args, "project_id"), "client_id": getStr(args, "client_id")})
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		default:
			return mcp.NewToolResultError("ação inválida: " + acao), nil
		}
	})

	// ══════════════════════════════════════════════════════════════
	// 10. GERENCIAR EVENTOS
	// ══════════════════════════════════════════════════════════════
	s.AddTool(mcp.NewTool("gerenciar_eventos",
		mcp.WithDescription("Gerencia eventos manuais do projeto. Ações: listar, criar, editar, excluir."),
		mcp.WithString("acao", mcp.Required(), mcp.Description("Ação: listar | criar | editar | excluir")),
		mcp.WithString("id", mcp.Description("ID do evento")),
		mcp.WithString("project_id", mcp.Description("ID do projeto")),
		mcp.WithString("title", mcp.Description("Título do evento")),
		mcp.WithString("date", mcp.Description("Data do evento")),
		mcp.WithString("author", mcp.Description("Autor do evento")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		if args == nil { return mcp.NewToolResultError("argumentos ausentes"), nil }
		acao := getStr(args, "acao")
		switch acao {
		case "listar":
			r, err := CallQuery(ctx, cfg, fmt.Sprintf("SELECT * FROM manual_events WHERE project_id = %s", getStr(args, "project_id")))
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "criar":
			data := map[string]any{"project_id": getStr(args, "project_id"), "title": getStr(args, "title"), "date": getStr(args, "date")}
			if v := getStr(args, "author"); v != "" { data["author"] = v }
			r, err := CallAction(ctx, cfg, "manual_event", "create", data)
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "editar":
			data := map[string]any{"id": getStr(args, "id")}
			for _, k := range []string{"title", "date", "author"} {
				if v := getStr(args, k); v != "" { data[k] = v }
			}
			r, err := CallAction(ctx, cfg, "manual_event", "update", data)
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "excluir":
			r, err := CallAction(ctx, cfg, "manual_event", "delete", map[string]any{"id": getStr(args, "id")})
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		default:
			return mcp.NewToolResultError("ação inválida: " + acao), nil
		}
	})

	// ══════════════════════════════════════════════════════════════
	// 11. GERENCIAR ANEXOS
	// ══════════════════════════════════════════════════════════════
	s.AddTool(mcp.NewTool("gerenciar_anexos",
		mcp.WithDescription("Gerencia anexos de tarefas. Ações: listar, criar, excluir."),
		mcp.WithString("acao", mcp.Required(), mcp.Description("Ação: listar | criar | excluir")),
		mcp.WithString("id", mcp.Description("ID do anexo")),
		mcp.WithString("task_id", mcp.Description("ID da tarefa")),
		mcp.WithString("filename", mcp.Description("Nome do arquivo")),
		mcp.WithString("file_url", mcp.Description("URL do arquivo")),
		mcp.WithString("file_type", mcp.Description("Tipo do arquivo")),
		mcp.WithString("uploaded_by_name", mcp.Description("Nome de quem enviou")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		if args == nil { return mcp.NewToolResultError("argumentos ausentes"), nil }
		acao := getStr(args, "acao")
		switch acao {
		case "listar":
			r, err := CallQuery(ctx, cfg, fmt.Sprintf("SELECT * FROM task_attachments WHERE task_id = %s ORDER BY uploaded_at DESC", getStr(args, "task_id")))
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "criar":
			data := map[string]any{"task_id": getStr(args, "task_id"), "filename": getStr(args, "filename"), "file_url": getStr(args, "file_url")}
			if v := getStr(args, "file_type"); v != "" { data["file_type"] = v }
			if v := getStr(args, "uploaded_by_name"); v != "" { data["uploaded_by_name"] = v }
			r, err := CallAction(ctx, cfg, "task_attachment", "create", data)
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		case "excluir":
			r, err := CallAction(ctx, cfg, "task_attachment", "delete", map[string]any{"id": getStr(args, "id")})
			if err != nil { return mcp.NewToolResultError(err.Error()), nil }
			return mcp.NewToolResultText(r), nil
		default:
			return mcp.NewToolResultError("ação inválida: " + acao), nil
		}
	})

	// ══════════════════════════════════════════════════════════════
	// 12. CONSULTA SQL (power tool)
	// ══════════════════════════════════════════════════════════════
	s.AddTool(mcp.NewTool("consulta_sql",
		mcp.WithDescription("Executa uma consulta SQL direta no banco de dados. Use apenas quando as outras ferramentas não forem suficientes."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Consulta SQL a executar")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		if args == nil { return mcp.NewToolResultError("argumentos ausentes"), nil }
		r, err := CallQuery(ctx, cfg, getStr(args, "query"))
		if err != nil { return mcp.NewToolResultError(err.Error()), nil }
		return mcp.NewToolResultText(r), nil
	})
}
