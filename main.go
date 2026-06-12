package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/mark3labs/mcp-go/server"
)

const setupHTML = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>MCP Gerenciador de Projetos - Configuração</title>
<style>
  @import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap');
  *,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
  :root{
    --bg:#0a0a0f;--surface:#12121a;--surface2:#1a1a26;--border:#2a2a3a;
    --border-focus:#6c5ce7;--primary:#6c5ce7;--primary-hover:#7c6ef7;
    --primary-glow:rgba(108,92,231,.35);--success:#00b894;
    --success-glow:rgba(0,184,148,.25);--danger:#e17055;
    --text:#e8e8f0;--text-muted:#8888a0;--radius:12px;
  }
  body{font-family:'Inter',system-ui,-apple-system,sans-serif;background:var(--bg);color:var(--text);min-height:100vh;display:flex;align-items:center;justify-content:center;padding:2rem;background-image:radial-gradient(ellipse at 20% 50%,rgba(108,92,231,.08) 0%,transparent 50%),radial-gradient(ellipse at 80% 20%,rgba(0,184,148,.06) 0%,transparent 50%)}
  .card{background:var(--surface);border:1px solid var(--border);border-radius:20px;padding:2.5rem;width:100%;max-width:520px;box-shadow:0 25px 60px rgba(0,0,0,.4);position:relative;overflow:hidden}
  .card::before{content:'';position:absolute;top:0;left:0;right:0;height:3px;background:linear-gradient(90deg,var(--primary),var(--success),var(--primary));background-size:200% 100%;animation:shimmer 3s ease infinite}
  @keyframes shimmer{0%,100%{background-position:0% 50%}50%{background-position:100% 50%}}
  .logo{width:48px;height:48px;background:linear-gradient(135deg,var(--primary),var(--success));border-radius:14px;display:flex;align-items:center;justify-content:center;font-size:1.4rem;font-weight:700;margin-bottom:1.2rem;box-shadow:0 8px 24px var(--primary-glow)}
  h1{font-size:1.35rem;font-weight:700;margin-bottom:.35rem;letter-spacing:-.02em}
  .subtitle{color:var(--text-muted);font-size:.85rem;margin-bottom:2rem;line-height:1.5}
  .field{margin-bottom:1.4rem}
  label{display:block;font-size:.8rem;font-weight:600;text-transform:uppercase;letter-spacing:.06em;color:var(--text-muted);margin-bottom:.5rem}
  input{width:100%;padding:.85rem 1rem;background:var(--surface2);border:1px solid var(--border);border-radius:var(--radius);color:var(--text);font-family:inherit;font-size:.92rem;transition:border-color .2s,box-shadow .2s;outline:none}
  input:focus{border-color:var(--border-focus);box-shadow:0 0 0 3px var(--primary-glow)}
  input::placeholder{color:#55556a}
  .actions{display:flex;gap:.75rem;margin-top:2rem}
  button{flex:1;padding:.85rem 1.2rem;border:none;border-radius:var(--radius);font-family:inherit;font-size:.9rem;font-weight:600;cursor:pointer;transition:transform .15s,box-shadow .2s,background .2s;display:flex;align-items:center;justify-content:center;gap:.5rem}
  button:active{transform:scale(.97)}
  .btn-test{background:var(--surface2);color:var(--text);border:1px solid var(--border)}
  .btn-test:hover{background:var(--border);box-shadow:0 4px 16px rgba(0,0,0,.3)}
  .btn-save{background:linear-gradient(135deg,var(--primary),#5a4bd1);color:#fff}
  .btn-save:hover{background:linear-gradient(135deg,var(--primary-hover),#6c5ce7);box-shadow:0 8px 24px var(--primary-glow)}
  .toast{position:fixed;bottom:2rem;left:50%;transform:translateX(-50%) translateY(80px);padding:.85rem 1.5rem;border-radius:var(--radius);font-size:.88rem;font-weight:500;opacity:0;transition:transform .35s cubic-bezier(.4,0,.2,1),opacity .35s;pointer-events:none;z-index:100;max-width:90vw;text-align:center}
  .toast.show{opacity:1;transform:translateX(-50%) translateY(0)}
  .toast.success{background:var(--success);color:#fff;box-shadow:0 8px 24px var(--success-glow)}
  .toast.error{background:var(--danger);color:#fff;box-shadow:0 8px 24px rgba(225,112,85,.25)}
  .spinner{width:16px;height:16px;border:2px solid transparent;border-top-color:currentColor;border-radius:50%;animation:spin .6s linear infinite;display:none}
  button.loading .spinner{display:inline-block}
  button.loading span{opacity:.6}
  @keyframes spin{to{transform:rotate(360deg)}}
  .config-path{margin-top:1.8rem;padding:.7rem 1rem;background:var(--surface2);border-radius:8px;font-size:.75rem;color:var(--text-muted);word-break:break-all;line-height:1.5}
  .config-path strong{color:var(--text);font-weight:600}
</style>
</head>
<body>
<div class="card">
  <div class="logo">GP</div>
  <h1>MCP Gerenciador de Projetos</h1>
  <p class="subtitle">Configure a conexão com sua API de gerenciamento de projetos.</p>
  <form id="configForm" autocomplete="off">
    <div class="field">
      <label for="apiUrl">URL da API</label>
      <input type="url" id="apiUrl" name="api_url" placeholder="https://clientes.seudominio.com.br" required>
    </div>
    <div class="field">
      <label for="apiKey">Chave da API</label>
      <input type="password" id="apiKey" name="api_key" placeholder="Insira sua chave de acesso" required>
    </div>
    <div class="actions">
      <button type="button" class="btn-test" id="btnTest" onclick="testarConexao()">
        <div class="spinner"></div>
        <span>Testar Conexão</span>
      </button>
      <button type="submit" class="btn-save" id="btnSave">
        <div class="spinner"></div>
        <span>Salvar</span>
      </button>
    </div>
  </form>
  <div class="config-path">
    <strong>Arquivo de configuração:</strong> <span id="configPath"></span>
  </div>
</div>
<div class="toast" id="toast"></div>
<script>
function showToast(msg,type){const t=document.getElementById('toast');t.textContent=msg;t.className='toast '+type;requestAnimationFrame(()=>t.classList.add('show'));setTimeout(()=>t.classList.remove('show'),3500)}
function setLoading(btn,loading){if(loading)btn.classList.add('loading');else btn.classList.remove('loading');btn.disabled=loading}
async function testarConexao(){const btn=document.getElementById('btnTest');const url=document.getElementById('apiUrl').value.trim();const key=document.getElementById('apiKey').value.trim();if(!url){showToast('Preencha a URL da API.','error');return}setLoading(btn,true);try{const res=await fetch('/test',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({api_url:url,api_key:key})});const data=await res.json();if(data.ok)showToast('Conexão realizada com sucesso!','success');else showToast('Falha: '+(data.error||'Erro desconhecido'),'error')}catch(e){showToast('Erro ao testar conexão.','error')}setLoading(btn,false)}
document.getElementById('configForm').addEventListener('submit',async function(e){e.preventDefault();const btn=document.getElementById('btnSave');const url=document.getElementById('apiUrl').value.trim();const key=document.getElementById('apiKey').value.trim();if(!url||!key){showToast('Preencha todos os campos.','error');return}setLoading(btn,true);try{const res=await fetch('/save',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({api_url:url,api_key:key})});const data=await res.json();if(data.ok)showToast('Configuração salva com sucesso!','success');else showToast('Erro ao salvar: '+(data.error||''),'error')}catch(e){showToast('Erro ao salvar configuração.','error')}setLoading(btn,false)});
fetch('/config').then(r=>r.json()).then(data=>{if(data.api_url)document.getElementById('apiUrl').value=data.api_url;if(data.api_key)document.getElementById('apiKey').value=data.api_key;if(data.config_path)document.getElementById('configPath').textContent=data.config_path}).catch(()=>{});
</script>
</body>
</html>`

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "--setup" {
			runSetupServer()
			return
		}
	}

	cfg := LoadConfig()
	if cfg.ApiUrl == "" {
		fmt.Fprintln(os.Stderr, "Configuração não encontrada. Execute com --setup primeiro.")
		fmt.Fprintf(os.Stderr, "  %s --setup\n", os.Args[0])
		os.Exit(1)
	}

	s := server.NewMCPServer("gerenciador-projetos", "1.0.0")
	RegisterAllTools(s, cfg)
	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao iniciar servidor MCP: %v\n", err)
		os.Exit(1)
	}
}

func runSetupServer() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, setupHTML)
	})

	mux.HandleFunc("GET /config", func(w http.ResponseWriter, r *http.Request) {
		cfg := LoadConfig()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"api_url":     cfg.ApiUrl,
			"api_key":     cfg.ApiKey,
			"config_path": ConfigPath(),
		})
	})

	mux.HandleFunc("POST /test", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ApiUrl string `json:"api_url"`
			ApiKey string `json:"api_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, false, "Corpo da requisição inválido")
			return
		}
		if body.ApiUrl == "" {
			writeJSON(w, false, "URL da API não informada")
			return
		}
		req, err := http.NewRequest("GET", body.ApiUrl, nil)
		if err != nil {
			writeJSON(w, false, "URL inválida: "+err.Error())
			return
		}
		if body.ApiKey != "" {
			req.Header.Set("Authorization", "Bearer "+body.ApiKey)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			writeJSON(w, false, "Não foi possível conectar: "+err.Error())
			return
		}
		defer resp.Body.Close()
		io.ReadAll(resp.Body)
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			writeJSON(w, true, "")
		} else {
			writeJSON(w, false, fmt.Sprintf("A API retornou status %d", resp.StatusCode))
		}
	})

	mux.HandleFunc("POST /save", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ApiUrl string `json:"api_url"`
			ApiKey string `json:"api_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, false, "Corpo da requisição inválido")
			return
		}
		cfg := &Config{ApiUrl: body.ApiUrl, ApiKey: body.ApiKey}
		if err := cfg.Save(); err != nil {
			writeJSON(w, false, "Erro ao salvar: "+err.Error())
			return
		}
		writeJSON(w, true, "")
	})

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  ╔══════════════════════════════════════════════╗")
	fmt.Fprintln(os.Stderr, "  ║  MCP Gerenciador de Projetos — Setup         ║")
	fmt.Fprintln(os.Stderr, "  ║  http://localhost:9111                        ║")
	fmt.Fprintln(os.Stderr, "  ╚══════════════════════════════════════════════╝")
	fmt.Fprintln(os.Stderr, "")

	if err := http.ListenAndServe(":9111", mux); err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao iniciar servidor de setup: %v\n", err)
		os.Exit(1)
	}
}

func writeJSON(w http.ResponseWriter, ok bool, errMsg string) {
	w.Header().Set("Content-Type", "application/json")
	resp := map[string]any{"ok": ok}
	if errMsg != "" {
		resp["error"] = errMsg
	}
	json.NewEncoder(w).Encode(resp)
}
