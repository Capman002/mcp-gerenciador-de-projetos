# MCP Gerenciador de Projetos

Projeto pessoal para o gerenciador de projetos da agência **ePixel**.  
Repositório público com fins de **aprendizado** sobre **MCP (Model Context Protocol)** e **Golang**.

---

## Sobre

Servidor MCP escrito em Go que conecta uma IA (Antigravity CLI, Claude Desktop, Cursor, etc.) à API do sistema de gerenciamento de projetos da ePixel.

### Ferramentas disponíveis (14 tools)

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
| `anotacoes` | listar, criar, editar, excluir | Caderno de anotações compartilhado entre conversas e IAs |
| `ver_proposta` | — | Texto completo de uma proposta do portal (sem `pagina`, lista as propostas) |

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

### Modo remoto (HTTP)

Com `MCP_HTTP_ADDR` (ex.: `127.0.0.1:3102`) e `MCP_HTTP_TOKEN` (≥ 32 caracteres) definidos,
o servidor fala MCP Streamable HTTP em `/mcp/<MCP_HTTP_TOKEN>` em vez de stdio. Em produção
ele roda dentro do container do portal, exposto em `https://clientes.epixel.com.br/mcp/<token>`.

## Estrutura

```
main.go      → Entry point + UI de configuração (--setup)
config.go    → Leitura/escrita de config em ~/.mcp-gerenciador/config.json
api.go       → Helpers HTTP para chamar a API SvelteKit
tools.go     → 12 ferramentas agrupadas por entidade
```


## Captação com acesso individual

A conexão administrativa existente continua em `/mcp/<MCP_HTTP_TOKEN>` e reúne
as 14 ferramentas de Projetos e as 13 de Captação. O vendedor usa exclusivamente
`/mcp/sales/<credencial individual>`, emitida e revogável no portal pelo administrador.
A conexão individual administrativa usa `/mcp/admin/<credencial individual admin>`
e preserva o autor real nas ações. Não compartilhe conexões administrativas com vendedores.

Cada requisição da conexão do vendedor verifica a credencial e o papel atual em
`GET /api/mcp/access`. O catálogo e a execução contêm somente ferramentas Captação:

- `captacao_listar`, `captacao_ficha`, `captacao_historico`
- `captacao_contato`, `captacao_descartar`, `captacao_visita`
- `captacao_mapa`, `captacao_rota`, `captacao_localizacao`
- `captacao_fontes`, `captacao_rotina`, `captacao_exportar`, `captacao_enviar_crm`

`captacao-tools.json` é o contrato compartilhado com o backend: entradas e saídas
possuem JSON Schema validado no MCP e na API. Listas possuem páginas de até 50
itens e cursores; ficha por seção evita carregar análises e históricos sem pedido.
A resposta Captação usa `structuredContent` sem repetir o JSON em `content`.
O cliente MCP precisa consumir respostas estruturadas. Consultas e ações do
vendedor usam sempre sua credencial individual; não há fallback para a chave admin.

A sincronização do catálogo deve preservar o arquivo byte a byte entre os dois
repositórios. `go test ./...` cobre separação de catálogos, rejeição de ferramentas
administrativas, limites/tipos, revogação, troca de papel e encaminhamento da credencial.
