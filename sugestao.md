# Proposta: Arquitetura de Tool Executor para geminiweb-go

---

## 1. Análise do Estado Atual

### 1.1 Arquitetura Existente (Pontos Fortes)

- **Sistema de Personas** (`internal/config/personas.go`): Já suporta `SystemPrompt` injetado via `FormatSystemPrompt()`
- **Interface de Sessão** (`ChatSessionInterface`): Bem definida com métodos como `SendMessage`, `SetMetadata`
- **TUI com Model-View-Update**: Loop de eventos maduro em `(Model).Update()` com ~360 linhas
- **Streaming**: Respostas em chunks JSON terminando com `[["e",...]]`

### 1.2 Problemas da Proposta Original

| Problema | Impacto | Solução |
|:---------|:--------|:--------|
| Protocolo XML com regex | Frágil, não escala | JSON estruturado com parser dedicado |
| Arquitetura monolítica (`tools.go`) | Viola separação de responsabilidades | Pacote `internal/tools/` modular |
| Segurança vaga | Riscos críticos não mitigados | Modelo de segurança em camadas |
| Ignora streaming | Incompatível com arquitetura atual | Interceptação de chunks parciais |
| Loop síncrono | Não cancela operações | Context-aware com cancelamento |

---

## 2. Arquitetura Proposta: Tool Executor v2

### 2.1 Protocolo de Comunicação

**Rejeitar XML inline**. Usar blocos de código JSON fenced, que são:
- Mais robustos para parsing
- Suportam argumentos complexos
- Fáceis de distinguir de código normal

#### Instrução de Sistema (Persona "Coder")

```
Quando precisar usar uma ferramenta, responda EXATAMENTE com um bloco JSON:

` ` `tool
{
  "name": "bash",
  "args": {"command": "ls -la"},
  "reason": "Listar arquivos do diretório atual"
}
` ` `

Aguarde o resultado antes de continuar. Múltiplas ferramentas devem ser chamadas sequencialmente.

Ferramentas disponíveis:
- bash: Executa comandos shell (requer confirmação)
- read_file: Lê conteúdo de arquivo
- write_file: Escreve/modifica arquivo (requer confirmação)
- search: Busca padrões em arquivos
```

#### Formato de Resposta de Tool

```json
{
  "tool": "bash",
  "success": true,
  "output": "total 64\ndrwxr-xr-x ...",
  "truncated": false,
  "execution_time_ms": 45
}
```

### 2.2 Estrutura de Pacotes

```
internal/
├── tools/
│   ├── registry.go      # Registry de ferramentas + interface Tool
│   ├── parser.go        # Parser de blocos ```tool
│   ├── executor.go      # Executor com context e timeout
│   ├── policy.go        # Políticas de confirmação/segurança
│   ├── bash.go          # Implementação: BashTool
│   ├── file_read.go     # Implementação: FileReadTool
│   ├── file_write.go    # Implementação: FileWriteTool
│   ├── search.go        # Implementação: SearchTool
│   └── tools_test.go    # Testes unitários
├── api/
│   └── session.go       # (sem alterações na interface)
└── tui/
    └── model.go         # Adicionar interceptação no case responseMsg
```

### 2.3 Interfaces e Tipos

```go
// internal/tools/registry.go

package tools

import (
    "context"
)

// Tool define a interface base para todas as ferramentas
type Tool interface {
    Name() string
    Description() string
    Execute(ctx context.Context, args map[string]any) (*Result, error)
    RequiresConfirmation(args map[string]any) bool
}

// Result representa o resultado de uma execução
type Result struct {
    Output        string        `json:"output"`
    Success       bool          `json:"success"`
    Truncated     bool          `json:"truncated"`
    ExecutionTime time.Duration `json:"execution_time_ms"`
    Error         string        `json:"error,omitempty"`
}

// ToolCall representa uma chamada parseada
type ToolCall struct {
    Name   string         `json:"name"`
    Args   map[string]any `json:"args"`
    Reason string         `json:"reason"`
}

// Registry gerencia ferramentas disponíveis
type Registry struct {
    tools map[string]Tool
    mu    sync.RWMutex
}

func NewRegistry() *Registry {
    r := &Registry{tools: make(map[string]Tool)}
    // Registrar ferramentas padrão
    r.Register(&BashTool{})
    r.Register(&FileReadTool{})
    r.Register(&FileWriteTool{})
    r.Register(&SearchTool{})
    return r
}

func (r *Registry) Register(t Tool) { ... }
func (r *Registry) Get(name string) (Tool, bool) { ... }
func (r *Registry) List() []Tool { ... }
```

### 2.4 Modelo de Segurança em Camadas

```go
// internal/tools/policy.go

package tools

// ConfirmationPolicy define quando solicitar confirmação
type ConfirmationPolicy int

const (
    PolicyAlwaysConfirm ConfirmationPolicy = iota  // Sempre perguntar
    PolicyConfirmDangerous                          // Apenas operações perigosas
    PolicyNeverConfirm                              // Modo YOLO (não recomendado)
)

// SecurityConfig define restrições de segurança
type SecurityConfig struct {
    // Bash
    AllowedCommands   []string      // Whitelist (vazia = permitir tudo)
    BlockedCommands   []string      // Blacklist (rm -rf, dd, etc.)
    MaxExecutionTime  time.Duration // Timeout padrão
    WorkingDirectory  string        // Restringir a este diretório

    // Arquivos
    AllowedPaths      []string      // Paths permitidos (glob)
    BlockedPaths      []string      // Paths bloqueados (.env, .ssh, etc.)
    MaxFileSize       int64         // Limite de leitura/escrita

    // Recursos
    MaxOutputSize     int           // Truncar outputs maiores
    MaxConcurrent     int           // Máximo de tools paralelas
}

// DefaultSecurityConfig retorna configuração segura padrão
func DefaultSecurityConfig() *SecurityConfig {
    return &SecurityConfig{
        BlockedCommands:  []string{"rm -rf /", "dd", "mkfs", ":(){:|:&};:"},
        BlockedPaths:     []string{".env", ".ssh/*", "*.pem", "*credentials*"},
        MaxExecutionTime: 30 * time.Second,
        MaxFileSize:      10 * 1024 * 1024, // 10MB
        MaxOutputSize:    100 * 1024,       // 100KB
        MaxConcurrent:    1,                // Sequencial por padrão
    }
}
```

### 2.5 Implementação do BashTool

```go
// internal/tools/bash.go

package tools

import (
    "context"
    "os/exec"
    "strings"
)

type BashTool struct {
    config *SecurityConfig
}

func (b *BashTool) Name() string { return "bash" }

func (b *BashTool) Description() string {
    return "Executa comandos shell no sistema"
}

func (b *BashTool) RequiresConfirmation(args map[string]any) bool {
    // Sempre requer confirmação para bash
    return true
}

func (b *BashTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
    cmdStr, ok := args["command"].(string)
    if !ok {
        return &Result{Success: false, Error: "argumento 'command' ausente ou inválido"}, nil
    }

    // Verificar blacklist
    for _, blocked := range b.config.BlockedCommands {
        if strings.Contains(cmdStr, blocked) {
            return &Result{
                Success: false,
                Error:   fmt.Sprintf("comando bloqueado: contém '%s'", blocked),
            }, nil
        }
    }

    // Criar contexto com timeout
    execCtx, cancel := context.WithTimeout(ctx, b.config.MaxExecutionTime)
    defer cancel()

    // Executar
    start := time.Now()
    cmd := exec.CommandContext(execCtx, "bash", "-c", cmdStr)

    if b.config.WorkingDirectory != "" {
        cmd.Dir = b.config.WorkingDirectory
    }

    output, err := cmd.CombinedOutput()
    elapsed := time.Since(start)

    result := &Result{
        ExecutionTime: elapsed,
        Success:       err == nil,
    }

    // Truncar se necessário
    if len(output) > b.config.MaxOutputSize {
        result.Output = string(output[:b.config.MaxOutputSize])
        result.Truncated = true
    } else {
        result.Output = string(output)
    }

    if err != nil {
        result.Error = err.Error()
    }

    return result, nil
}
```

### 2.6 Parser de Tool Calls

```go
// internal/tools/parser.go

package tools

import (
    "encoding/json"
    "regexp"
)

var toolBlockRegex = regexp.MustCompile("(?s)```tool\n(.+?)\n```")

// ParseToolCalls extrai todas as chamadas de ferramenta de um texto
func ParseToolCalls(text string) ([]ToolCall, string) {
    matches := toolBlockRegex.FindAllStringSubmatch(text, -1)

    var calls []ToolCall
    cleanText := text

    for _, match := range matches {
        var call ToolCall
        if err := json.Unmarshal([]byte(match[1]), &call); err != nil {
            continue // Ignorar blocos malformados
        }
        calls = append(calls, call)
        cleanText = strings.Replace(cleanText, match[0], "", 1)
    }

    return calls, strings.TrimSpace(cleanText)
}
```

### 2.7 Integração com TUI

Modificação no `internal/tui/model.go`, case `responseMsg`:

```go
case responseMsg:
    m.loading = false
    m.lastOutput = msg.output
    responseText := msg.output.Text()

    // [NOVO] Verificar se há tool calls na resposta
    toolCalls, cleanText := tools.ParseToolCalls(responseText)

    if len(toolCalls) > 0 {
        // Processar ferramentas
        for _, call := range toolCalls {
            tool, exists := m.toolRegistry.Get(call.Name)
            if !exists {
                m.err = fmt.Errorf("ferramenta desconhecida: %s", call.Name)
                continue
            }

            // Verificar se precisa confirmação
            if tool.RequiresConfirmation(call.Args) {
                // Exibir prompt de confirmação na TUI
                // (implementar modal de confirmação)
                confirmed := m.showConfirmation(call)
                if !confirmed {
                    continue
                }
            }

            // Executar ferramenta
            ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
            result, err := tool.Execute(ctx, call.Args)
            cancel()

            if err != nil {
                m.err = err
                continue
            }

            // Enviar resultado de volta ao modelo
            toolResultMsg := formatToolResult(call.Name, result)
            cmd = m.sendMessage(toolResultMsg)
            m.loading = true
            return m, tea.Batch(cmd, animationTick())
        }
    }

    // Continuar processamento normal com texto limpo
    // ...
```

---

## 3. Estrutura de Histórico para Tool Calls

### 3.1 Extensão do Modelo de Mensagem

```go
// internal/history/store.go

type Message struct {
    ID        string    `json:"id"`
    Role      string    `json:"role"` // "user", "assistant", "tool"
    Content   string    `json:"content"`
    Thoughts  string    `json:"thoughts,omitempty"`
    Timestamp time.Time `json:"timestamp"`

    // Novos campos para tools
    ToolCall   *ToolCallRecord   `json:"tool_call,omitempty"`
    ToolResult *ToolResultRecord `json:"tool_result,omitempty"`
}

type ToolCallRecord struct {
    Name   string         `json:"name"`
    Args   map[string]any `json:"args"`
    Reason string         `json:"reason"`
}

type ToolResultRecord struct {
    ToolName      string        `json:"tool_name"`
    Success       bool          `json:"success"`
    Output        string        `json:"output"`
    Truncated     bool          `json:"truncated"`
    ExecutionTime time.Duration `json:"execution_time_ms"`
    Error         string        `json:"error,omitempty"`
}
```

---

## 4. Persona "Coder" Padrão

Adicionar em `internal/config/personas.go`:

```go
func DefaultPersonas() []Persona {
    return []Persona{
        // ... personas existentes ...
        {
            Name:        "Coder",
            Description: "Agente de desenvolvimento com acesso a ferramentas do sistema",
            Model:       "pro", // Modelo mais capaz para raciocínio
            SystemPrompt: `Você é um agente de desenvolvimento de software com acesso às seguintes ferramentas:

## Ferramentas Disponíveis

1. **bash** - Executa comandos shell
   - Args: {"command": "string"}
   - Requer confirmação do usuário

2. **read_file** - Lê conteúdo de arquivos
   - Args: {"path": "string", "lines": number (opcional)}

3. **write_file** - Cria ou modifica arquivos
   - Args: {"path": "string", "content": "string"}
   - Requer confirmação do usuário

4. **search** - Busca padrões em arquivos
   - Args: {"pattern": "string", "path": "string (opcional)", "type": "regex|literal"}

## Formato de Chamada

Para usar uma ferramenta, responda com um bloco:

` ` `tool
{
  "name": "nome_da_ferramenta",
  "args": {...},
  "reason": "Breve explicação do motivo"
}
` ` `

## Diretrizes

- Execute UMA ferramenta por vez e aguarde o resultado
- Sempre explique seu raciocínio antes de usar uma ferramenta
- Se um comando falhar, analise o erro antes de tentar novamente
- Prefira comandos seguros e reversíveis
- Nunca execute comandos destrutivos sem extrema necessidade`,
        },
    }
}
```

---

## 5. Plano de Implementação

### Fase 1: Infraestrutura (Prioridade Alta)
1. Criar pacote `internal/tools/` com interfaces base
2. Implementar `Registry` e `Parser`
3. Implementar `BashTool` com modelo de segurança
4. Adicionar testes unitários

### Fase 2: Integração TUI (Prioridade Alta)
1. Adicionar interceptação no case `responseMsg`
2. Implementar modal de confirmação
3. Implementar loop de tool execution

### Fase 3: Ferramentas Adicionais (Prioridade Média)
1. `FileReadTool`
2. `FileWriteTool`
3. `SearchTool`

### Fase 4: Persistência (Prioridade Média)
1. Estender `Message` com campos de tool
2. Atualizar serialização/deserialização
3. Atualizar exportadores (Markdown, JSON)

### Fase 5: Polimento (Prioridade Baixa)
1. Logging estruturado com `slog`
2. Métricas de execução
3. Configuração de políticas via arquivo

---

## 6. Considerações de Segurança

### 6.1 Princípios

1. **Defesa em Profundidade**: Múltiplas camadas de validação
2. **Princípio do Menor Privilégio**: Ferramentas só acessam o necessário
3. **Fail-Safe Defaults**: Configurações padrão são restritivas

### 6.2 Mitigações Específicas

| Risco | Mitigação |
|:------|:----------|
| Execução de comandos maliciosos | Blacklist + confirmação obrigatória |
| Acesso a arquivos sensíveis | Whitelist de paths + bloqueio de padrões |
| DoS via comandos longos | Timeout configurável (padrão 30s) |
| Output flooding | Truncamento automático (100KB) |
| Escape do diretório | Validação de path + `WorkingDirectory` |

### 6.3 Recomendações Avançadas (Futuro)

- Execução em container isolado (Docker/Podman)
- Namespaces de usuário (Linux)
- Seccomp profiles para syscalls
- Auditoria de comandos executados

---

## 7. Alternativa: Integração com Gems

As **Gems** do Gemini são personas server-side. Uma alternativa mais simples seria:

1. Criar uma Gem "Coder" no console do Gemini
2. Configurar com o system prompt de ferramentas
3. Usar `--gem "Coder"` para ativar

**Vantagem**: Sem necessidade de código adicional para o prompt.
**Desvantagem**: Ainda requer implementação do parser e executor local.

---

## 8. Conclusão

A implementação de um Tool Executor robusto requer:

1. **Protocolo bem definido** (JSON > XML)
2. **Arquitetura modular** (pacote dedicado)
3. **Segurança em camadas** (blacklist, confirmação, timeout, truncamento)
4. **Integração com streaming** (interceptação de chunks)
5. **Observabilidade** (logs, métricas)

A proposta original apresentava ideias válidas, mas carecia de detalhamento técnico e considerações de segurança. Este documento fornece uma especificação mais completa e alinhada com a arquitetura existente do geminiweb-go.
