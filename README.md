# MCP Gerenciador de Projetos - ePixel

Este é um projeto pessoal utilizado para o gerenciador de projetos da agência ePixel. 
O repositório foi tornado público exclusivamente a fim de aprendizado de outras pessoas sobre **MCP (Model Context Protocol)** e desenvolvimento em **Golang**.

## Sobre
Este é um **Servidor MCP** escrito em Go que atua como uma ponte (tradutor) entre uma IA (como Claude Desktop, Cursor, etc) e a API do sistema SvelteKit da ePixel.

Ele expõe duas ferramentas (`tools`) para a IA:
- `admin_action`: Permite criar, editar e excluir qualquer entidade do sistema de forma estruturada.
- `query_db`: Permite a leitura de dados do banco através de consultas SQL.

## Variáveis de Ambiente

Para rodar este MCP, você precisará definir as seguintes variáveis de ambiente:
- `SVELTEKIT_API_URL`: A URL do seu servidor em produção (ex: `https://clientes.epixel.com.br`) ou local.
- `MCP_API_KEY`: A chave secreta definida no servidor SvelteKit.

## Compilando e Rodando

```bash
go build -o mcp-gerenciador
```
