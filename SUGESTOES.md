# Sugestões de melhorias para `geminiweb-go`

Este documento reúne oportunidades de **bugs**, **segurança/privacidade**, **testes/CI**, **novas features** e **otimizações** encontradas ao revisar o código do `geminiweb-go`. A pasta `Gemini-API/` foi usada apenas como referência de funcionalidades/paridade.

## 0) Prioridade máxima (ações imediatas)

### 0.1) Segurança: remoção urgente de credenciais versionadas
- Existem **cookies reais** em arquivos versionados (ex.: `request1.txt`, `request2.txt`, `requests/geminiwebrequest.txt`, `requests/geminirequest2.txt`). Isso compromete a conta e qualquer ambiente/CI que clone o repositório.
- Ação recomendada:
  - Remover esses arquivos do repositório (e do histórico do Git, se necessário).
  - Adicionar padrões à `.gitignore` (`requests/`, `request*.txt`, `logs/` etc.).
  - Considerar rodar um *secret scanner* (ex.: gitleaks) em CI e/ou pre-commit.
  - Rotacionar/invalidar cookies imediatamente (logout, revogar sessões do Google, etc.).

### 0.2) Testes/CI: `go test ./...` falha e pode travar
- A suíte `internal/commands` tem testes que:
  - **Falham** por expectativa de comando inexistente (`sync`) em `internal/commands/root_test.go`.
  - **Podem travar** por chamar `RunE` de comandos interativos (ex.: `internal/commands/chat_test.go` chama `chatCmd.RunE` e dispara a TUI).
  - **Podem vazar dados sensíveis** por executar auto-login real (ex.: `internal/commands/autologin_test.go` chama `runAutoLogin("auto")`).
- Ação recomendada: separar testes unitários e integração, e tornar `make test` estável.

### 0.3) Privacidade: permissões de arquivos do histórico
- Conversas e metadados são gravados com permissão `0644` em `internal/history/store.go` e `internal/history/meta.go`. Em máquinas multiusuário isso permite leitura por terceiros.
- Ação recomendada:
  - Diretórios: `0700` (`~/.geminiweb/`, `~/.geminiweb/history/`).
  - Arquivos: `0600` (`history/*.json`, `meta.json`, `personas.json` se considerado sensível).
  - Escrita atômica (tmp + rename) para reduzir risco de corrupção.

## 1) Bugs / comportamentos inconsistentes

### 1.1) `chat` aceita argumentos e o teste chama execução real
- `internal/commands/chat.go`: `chatCmd` não define `Args`, então `geminiweb chat foo` não é rejeitado.
- `internal/commands/chat_test.go`: `TestChatCommand_Args` chama `chatCmd.RunE(...)`, o que dispara `runChat()` (TUI) e pode travar o teste.
- Sugestão:
  - Definir `Args: cobra.NoArgs` em `chatCmd`.
  - No teste, validar `chatCmd.Args(...)` ou executar via Cobra com buffers, sem chamar `RunE` diretamente.

### 1.2) Detecção de “raw output” não cobre `stdout` pipe sem `stdin` pipe
- `internal/commands/root.go`: `rawOutput` só é ativado quando há `stdin` pipe e `stdout` não é TTY.
- Caso comum: `geminiweb "oi" | cat` (stdout não-tty, stdin tty) ainda produz saída decorada/ANSI.
- Sugestão: considerar `rawOutput` quando `stdout` não é TTY (ou oferecer `--raw/--no-ansi` e ativar automaticamente em não-TTY).

### 1.3) `/save` (TUI) pode baixar imagem gerada sem `FullSize`
- `internal/tui/model.go` usa `DownloadSelectedImages`.
- `internal/api/download.go`: `DownloadSelectedImages` usa `output.Images()` (que converte `GeneratedImage` → `WebImage`), perdendo a distinção e chamando `DownloadImage` para tudo; imagens geradas podem não receber `=s2048`.
- Sugestão: em `DownloadSelectedImages`, decidir por índice:
  - `< len(candidate.WebImages)` → `DownloadImage`
  - `>= len(candidate.WebImages)` → `DownloadGeneratedImage` (preservando `FullSize`).

### 1.4) Auto-login “silencioso” na inicialização não parece respeitar `WithBrowserRefresh`
- `internal/api/client.go`: `attemptInitialAuth()` tenta extração do browser quando cookies não existem, mesmo sem `c.browserRefresh` estar habilitado.
- Sugestão:
  - Alinhar com o comentário de `NewClient` (“extrai do browser se browserRefresh estiver enabled”) **ou**
  - Formalizar esse comportamento como feature (ex.: `WithInitialBrowserLogin(true)`), com opção explícita para desligar.

### 1.5) Config de Markdown não é aplicada no modo “query”
- `internal/render/config.go` carrega opções do config, mas `internal/commands/query.go` usa `render.MarkdownWithWidth` (defaults) e ignora config do usuário.
- Sugestão: no output do `runQuery`, renderizar com `render.LoadOptionsFromConfigWithWidth(...)` (ou opção equivalente).

### 1.6) Itens de configuração “sem efeito”
- `config.Config.AutoClose` e `config.Config.Verbose` aparecem na UI (`internal/tui/config_model.go`), mas não são usados no fluxo principal (API/commands).
- `persona` existe como comando (`internal/commands/persona.go`), mas não é aplicado em `chat`/`query`.
- Sugestão: implementar ou remover para evitar “config placebo”.

## 2) Segurança & privacidade (além do vazamento de cookies)

- **Higiene do repositório**: `logs/` e `requests/` deveriam ficar fora do Git (ou sanitizados), pois podem conter dados sensíveis de debug.
- **Redução de prints no core**: `internal/api/client.go` usa `fmt.Printf` para avisos (salvar cookies). Sugestão: injetar logger (ou retornar *warning* estruturado) para não poluir stdout/stderr e facilitar testes.
- **Filtro de domínio de cookies**: `internal/browser/browser.go` filtra domínio com `strings.Contains(cookie.Domain, "google.com")`. Sugestão: usar `HasSuffix`/match mais restrito (ex.: `.google.com`, `accounts.google.com`) para evitar falsos positivos.

## 3) Testes & CI (qualidade e confiabilidade)

- **Separar unit vs integração**:
  - Auto-login e extração real de cookies deveriam estar sob build tag (`//go:build integration`) e nunca rodar por padrão.
  - O mesmo vale para qualquer teste que dependa de browser instalado/perfis reais.
- **Evitar executar TUIs**: testes de comandos devem validar `Args`, flags e wiring; lógica interativa deve ser testada em funções puras ou com mocks.
- **Consertar expectativas desatualizadas**:
  - Remover `sync` dos testes em `internal/commands/root_test.go` (ou reintroduzir conscientemente se for feature real do produto).
- **Adicionar testes direcionados a bugs reais**:
  - Caso `DownloadSelectedImages` (web vs generated).
  - Permissões/atômicidade do histórico.
  - “raw output” em stdout não-tty.

## 4) Features (alto valor) sugeridas para o `geminiweb-go`

### 4.1) Streaming de resposta (CLI e TUI)
- A API já é “streaming” por chunks; hoje o código espera o final (`[["e",`) antes de renderizar.
- Sugestão:
  - Expor API de streaming (callback/chan) em `internal/api`.
  - No CLI, `--stream` para imprimir conforme chega.
  - Na TUI, atualizar viewport incrementalmente.

### 4.2) Seleção de *candidates* (múltiplas respostas)
- O modelo suporta múltiplos candidates (`models.ModelOutput.Candidates`) e `ChatSession.ChooseCandidate`.
- Sugestão:
  - CLI: flag `--candidate N` e `--list-candidates`.
  - TUI: atalho para trocar candidate e re-renderizar.

### 4.3) Proxy e perfis de rede
- A pasta original (`Gemini-API/`) expõe `proxy`; no Go não há uma opção clara.
- Sugestão: opção via config/flag (respeitando o padrão do projeto) para proxy HTTP(S) e/ou upstream (se suportado pelo `tls-client`).

### 4.4) Auto-close e gerenciamento de recursos
- Paridade com `auto_close`/`close_delay` do projeto base:
  - Implementar timer de inatividade em `GeminiClient` que faz `Close()` e encerra rotator quando ocioso.
  - Integrar com `config.AutoClose` e um `CloseDelay` configurável.

### 4.5) Personas locais aplicadas ao chat/query
- Integrar `persona`:
  - `geminiweb chat --persona coder` (inserir prompt inicial ou prefixar instruções).
  - Slash command `/persona` na TUI.
  - Persistir persona usada por conversa no histórico.

### 4.6) Attachments múltiplos no modo “query”
- Hoje o `query` suporta 1 `--image` e upload automático de prompt grande.
- Sugestão: permitir `--attach` repetível (imagens + docs) e/ou `--file` de anexos (evitar conflito com `--file` de prompt).

### 4.7) UX de CLI
- `completion` (bash/zsh/fish/powershell) via Cobra.
- `doctor`/`diagnose`: checar cookies, token, conectividade, detecção de bloqueio/captcha, permissões de diretórios.
- `--no-ansi`/`--plain` e `--json` (saída estruturada para scripts).

## 5) Otimizações e refatorações pontuais

- **Downloads/Uploads sem `io.ReadAll`**:
  - `internal/api/download.go`: stream para arquivo com `io.Copy` (evita alocar imagem inteira em memória).
  - `internal/api/upload.go`: usar `io.Pipe` + `multipart.Writer` para streaming de arquivos grandes (evita buffer de 50MB+).
- **Compilar regex uma vez**:
  - `internal/api/generate.go` e `internal/api/download.go` usam `regexp.MatchString`/`MustCompile` local; mover para `var re = regexp.MustCompile(...)`.
- **Reduzir escopo de locks**:
  - `internal/api/client.go`: `RefreshFromBrowser` faz I/O e `GetAccessToken` sob lock; mover I/O/HTTP para fora do lock quando possível.
- **Alinhar fingerprint e headers**:
  - TLS profile é Chrome 133 (`profiles.Chrome_133`), mas `models.DefaultHeaders()` usa UA/`Sec-CH-UA` de Chrome 131. Alinhar versões e separar headers por endpoint (document vs XHR) para consistência.

## 6) Documentação

- `README.md` na raiz parece truncado/corrompido (termina com `</arg_value>`/`</tool_call>`). Sugestão: corrigir e adicionar:
  - Guia rápido de instalação/uso.
  - Explicação clara de `auto-login`, `browser-refresh`, histórico e gems.
  - Seção “Segurança” (cookies, arquivos locais, permissões).

