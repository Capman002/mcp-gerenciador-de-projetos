# MCP Gerenciador de Projetos

Projeto pessoal para o gerenciador de projetos da agência **ePixel**.  
Repositório público com fins de **aprendizado** sobre **MCP (Model Context Protocol)** e **Golang**.

---

## Sobre

Servidor MCP escrito em Go que conecta uma IA (Antigravity CLI, Claude Desktop, Cursor, etc.) à API do sistema de gerenciamento de projetos da ePixel.

### Ferramentas disponíveis (12 tools)

| Ferramenta | Ações | Descrição |
|---|---|---|
| `gerenciar_clientes` | listar, buscar, criar, editar, excluir | Clientes do sistema |
| `gerenciar_projetos` | listar, buscar, criar, editar, excluir | Projetos e configurações |
| `gerenciar_entregas` | listar, buscar, criar, editar, excluir | Entregas do roadmap (status: Bastidores, Em Pauta, Em Produção, No Ar) |
| `gerenciar_parcelas` | listar, criar, editar, excluir | Parcelas de pagamento |
| `gerenciar_avisos` | listar, criar, editar, excluir | Avisos rápidos (com link opcional) |
| `gerenciar_documentos` | listar, criar, editar, excluir | Documentos e links (tipos: Contrato, Proposta, Relatório, Escopo, Outro) |
| `gerenciar_marcos` | listar, criar, editar, excluir | Marcos do projeto (tipos: Entrega, Cliente, Reunião) |
| `gerenciar_diario` | listar, criar, editar, excluir | Diário do projeto (atualizações e histórico) |
| `gerenciar_membros` | listar, adicionar, remover | Equipe do projeto (roles: owner, viewer, hidden) |
| `gerenciar_eventos` | listar, criar, editar, excluir | Eventos manuais |
| `gerenciar_anexos` | listar, criar, excluir | Anexos de entregas |
| `consulta_sql` | — | Consulta SQL direta (power tool) |

## Configuração

### 1. Compilar

```bash
go build -o mcp-gerenciador.exe .
```

### 2. Configurar via UI

```bash
./mcp-gerenciador --setup
```

Abre uma interface web em `http://localhost:9111` para configurar URL e chave da API.

### 3. Adicionar ao seu cliente MCP

**Antigravity CLI** (`~/.gemini/antigravity-cli/mcp_config.json`):

```json
{
  "mcpServers": {
    "gerenciador-projetos": {
      "command": "/caminho/para/mcp-gerenciador.exe"
    }
  }
}
```

A configuração de URL e chave é lida de `~/.mcp-gerenciador/config.json`, salva via `--setup`.

## Estrutura

```
main.go      → Entry point + UI de configuração (--setup)
config.go    → Leitura/escrita de config em ~/.mcp-gerenciador/config.json
api.go       → Helpers HTTP para chamar a API SvelteKit
tools.go     → 12 ferramentas agrupadas por entidade
```
