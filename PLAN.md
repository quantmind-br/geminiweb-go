# Análise Crítica: Plano de Exportação de Conversas no TUI

## Resumo Executivo

O documento `shotgun-prompt-20251210-183012_response.md` propõe a implementação de um comando `/export` no TUI para exportar conversas para `.md` e `.json`. Embora bem estruturado, contém **inconsistências com o código atual**, **lacunas técnicas significativas** e **decisões de design questionáveis**.

---

## 1. Inconsistências com o Código Atual

### 1.1. Interface `FullHistoryStore` Incompleta

**Problema**: O plano afirma que `ExportToJSON` já existe no `*history.Store` e apenas precisa ser adicionada à interface. Porém, a interface atual (`internal/tui/model.go:70-80`) **já possui `ExportToMarkdown` mas NÃO possui `ExportToJSON`**:

```go
// Estado atual (internal/tui/model.go:70-80)
type FullHistoryStore interface {
    HistoryStoreInterface
    ListConversations() ([]*history.Conversation, error)
    GetConversation(id string) (*history.Conversation, error)
    CreateConversation(model string) (*history.Conversation, error)
    DeleteConversation(id string) error
    ToggleFavorite(id string) (bool, error)
    MoveConversation(id string, newIndex int) error
    SwapConversations(id1, id2 string) error
    ExportToMarkdown(id string) (string, error)  // Já existe
    // ExportToJSON AUSENTE!
}
```

**Correção**: O plano deve especificar que `ExportToJSON(id string) ([]byte, error)` precisa ser adicionado à interface.

### 1.2. Assinatura de `ExportToJSON` Incorreta

**Problema**: O plano mostra a assinatura como `ExportToJSON(id string) ([]byte, error)`, mas não considera `ExportToJSONWithOptions` que permite incluir/excluir metadata e thoughts.

**Correção**: Decidir se o TUI deve expor opções de exportação ou usar valores padrão.

### 1.3. Padrão de Comandos Ignorado

**Problema**: O plano não menciona que comandos existentes (`/file`, `/favorite`) já usam um padrão consistente de:
1. Validar argumentos
2. Expandir `~` para home directory
3. Retornar `(tea.Model, tea.Cmd)` para operações assíncronas

**Correção**: Seguir explicitamente o padrão de `handleFileCommand` (`internal/tui/model.go:707-738`).

---

## 2. Lacunas Técnicas

### 2.1. Nenhum Plano de Testes

**Problema Crítico**: O documento não menciona testes unitários ou de integração.

**Correção Proposta**:
```
Testes Necessários:
├── TestExportCommand_ValidPath_Markdown
├── TestExportCommand_ValidPath_JSON
├── TestExportCommand_InvalidFormat
├── TestExportCommand_NoConversation
├── TestExportCommand_PathExpansion (~)
├── TestExportCommand_PermissionDenied
├── TestExportCommand_FileExists (comportamento?)
└── TestExportResultMsg_HandledInUpdate
```

### 2.2. Comportamento de Arquivo Existente Não Definido

**Problema**: O que acontece se `/export chat.md` for executado e `chat.md` já existir?

**Decisões Necessárias**:
| Opção | Prós | Contras |
|-------|------|---------|
| Sobrescrever silenciosamente | Simples | Perda de dados potencial |
| Adicionar sufixo numérico (`chat-1.md`) | Seguro | Proliferação de arquivos |
| Falhar com erro | Explícito | UX ruim para atualização intencional |
| Flag `--force` / `--overwrite` | Controle total | Complexidade do comando |

**Recomendação**: Sobrescrever com aviso no feedback: `✓ Exported to chat.md (overwritten)`.

### 2.3. Sanitização de Nome de Arquivo

**Problema**: O plano menciona usar o título da conversa como nome padrão, mas não aborda:
- Caracteres inválidos (`/`, `\`, `:`, `*`, `?`, `"`, `<`, `>`, `|`)
- Nomes reservados no Windows (`CON`, `PRN`, `NUL`)
- Títulos muito longos (>255 caracteres)

**Correção Proposta**: Adicionar função `sanitizeFilename`:
```go
func sanitizeFilename(title string) string {
    // Substituir caracteres inválidos por underscore
    // Truncar para 200 caracteres
    // Adicionar extensão apropriada
}
```

### 2.4. Exportação de Conversa Não Salva

**Problema**: O plano assume que `m.conversation != nil` significa que há um ID válido. Porém, uma conversa pode existir em memória mas ainda não ter sido salva (ex: usuário iniciou chat mas não enviou nenhuma mensagem).

**Cenários não cobertos**:
1. `m.conversation == nil` - Conversa não iniciada
2. `m.conversation.ID == ""` - Conversa em memória, não persistida
3. `m.conversation.Messages == nil` - Conversa vazia

**Correção**: Implementar exportação direta da memória como fallback:
```go
if m.conversation == nil || m.conversation.ID == "" {
    // Exportar m.messages diretamente, sem passar pelo store
    return exportInMemoryConversation(m.messages, format, path)
}
```

### 2.5. Diretório Padrão Não Especificado

**Problema**: Se o usuário executar `/export chat.md` (sem caminho), onde o arquivo será salvo?

**Opções**:
| Opção | Comportamento |
|-------|---------------|
| CWD | Diretório atual do processo |
| `~/.geminiweb/exports/` | Diretório dedicado |
| `~/Downloads/` | Convenção de navegadores |

**Recomendação**: CWD para caminhos relativos, com feedback mostrando caminho absoluto.

### 2.6. Concorrência

**Problema**: Não há proteção contra múltiplas exportações simultâneas.

**Cenário**: Usuário executa `/export file1.md`, não espera, executa `/export file2.md`.

**Correção**: Adicionar flag `m.exporting bool` ou usar o sistema de `m.loading` existente.

---

## 3. Problemas de Design

### 3.1. Sintaxe do Comando Ambígua

**Problema**: O plano mostra `/export <filename> [-f json]` mas não especifica:
- Ordem das flags: `/export -f json file.md` é válido?
- Prioridade: Se `-f json` e `.md` conflitarem, qual vence?

**Recomendação**: Definir claramente:
```
/export <path>           # Formato inferido pela extensão (.md padrão)
/export <path> -f <fmt>  # Formato forçado, ignora extensão
/export -f <fmt>         # Usa título como nome, formato especificado
```

### 3.2. Feedback via `m.err` é Anti-Pattern

**Problema**: O plano sugere usar `m.err` para feedback de sucesso (`✓ Exported to...`). Isso mistura erros com mensagens de sucesso.

**Estado Atual no Código**: Já existe esse anti-pattern para `/favorite`:
```go
// internal/tui/model.go:336
m.err = fmt.Errorf("★ Added to favorites")  // <- Não é erro!
```

**Correção Proposta**: Introduzir campo `m.feedback string` separado de `m.err error`:
```go
type Model struct {
    // ...
    err      error   // Erros reais
    feedback string  // Mensagens de sucesso (limpa após N segundos)
}
```

### 3.3. Localização da Lógica

**Problema**: O plano coloca toda a lógica em `internal/tui/model.go`, aumentando o tamanho do arquivo.

**Estado Atual**: `model.go` já tem ~1600 linhas.

**Alternativa**: Criar `internal/tui/export.go` para:
- `exportResultMsg` struct
- `exportCommand` function
- `handleExportCommand` method

---

## 4. Melhorias Propostas

### 4.1. Estrutura de Mensagens Refatorada

```go
// internal/tui/messages.go (novo arquivo)
type exportResultMsg struct {
    path      string    // Caminho do arquivo exportado
    format    string    // "markdown" ou "json"
    size      int64     // Tamanho em bytes
    overwrite bool      // Se sobrescreveu arquivo existente
    err       error     // Erro, se houver
}
```

### 4.2. Suporte a Exportação de Conversa em Memória

```go
func (m Model) handleExportCommand(args string) (tea.Model, tea.Cmd) {
    path, format := parseExportArgs(args)

    // Prioridade: conversa salva > conversa em memória
    if m.conversation != nil && m.conversation.ID != "" {
        return m, exportFromStore(m.fullHistoryStore, m.conversation.ID, format, path)
    }

    if len(m.messages) > 0 {
        return m, exportFromMemory(m.messages, format, path)
    }

    m.err = fmt.Errorf("no conversation to export")
    return m, nil
}
```

### 4.3. Parser de Argumentos Robusto

```go
func parseExportArgs(args string) (path, format string, err error) {
    // Suportar:
    // /export file.md
    // /export file.md -f json
    // /export -f json file.md
    // /export -f json (usa título como nome)

    parts := strings.Fields(args)
    // ... lógica de parsing
}
```

### 4.4. Validação de Caminho Completa

```go
func validateExportPath(path string) (string, error) {
    // 1. Expandir ~
    // 2. Converter para absoluto
    // 3. Verificar se diretório pai existe
    // 4. Verificar permissões de escrita
    // 5. Sanitizar nome do arquivo
    // 6. Retornar caminho limpo
}
```

---

## 5. Plano de Implementação Revisado

### Fase 1: Infraestrutura (Prioridade: Alta)

| # | Task | Arquivo | Descrição |
|---|------|---------|-----------|
| 1.1 | Adicionar `ExportToJSON` à interface | `internal/tui/model.go` | Incluir método na `FullHistoryStore` |
| 1.2 | Criar `exportResultMsg` | `internal/tui/model.go` | Struct com `path`, `format`, `size`, `err` |
| 1.3 | Criar `parseExportArgs` | `internal/tui/model.go` | Parser para `/export <path> [-f format]` |
| 1.4 | Criar `validateExportPath` | `internal/tui/model.go` | Validação e sanitização de caminho |

### Fase 2: Implementação Core (Prioridade: Alta)

| # | Task | Arquivo | Descrição |
|---|------|---------|-----------|
| 2.1 | Criar `exportCommand` | `internal/tui/model.go` | tea.Cmd assíncrono para I/O |
| 2.2 | Criar `handleExportCommand` | `internal/tui/model.go` | Handler do comando `/export` |
| 2.3 | Adicionar case em `Update` | `internal/tui/model.go` | Processar `exportResultMsg` |
| 2.4 | Registrar comando no switch | `internal/tui/model.go` | `case "export":` |

### Fase 3: Robustez (Prioridade: Média)

| # | Task | Descrição |
|---|------|-----------|
| 3.1 | Implementar `exportFromMemory` | Exportar sem ID (conversa não salva) |
| 3.2 | Implementar `sanitizeFilename` | Limpar título para uso como nome |
| 3.3 | Tratamento de arquivo existente | Sobrescrever com indicação no feedback |
| 3.4 | Adicionar dica na status bar | `/export` na lista de comandos |

### Fase 4: Testes (Prioridade: Alta)

| # | Test | Cobertura |
|---|------|-----------|
| 4.1 | `TestParseExportArgs` | Todos os formatos de entrada |
| 4.2 | `TestValidateExportPath` | Expansão, sanitização, permissões |
| 4.3 | `TestExportCommand` | Sucesso markdown/json, erros |
| 4.4 | `TestExportFromMemory` | Conversa não salva |

---

## 6. Decisões Técnicas Pendentes

| # | Questão | Recomendação | Justificativa |
|---|---------|--------------|---------------|
| 1 | Arquivo existente | Sobrescrever com indicação | Simples, evita proliferação |
| 2 | Diretório padrão | CWD | Consistente com comportamento Unix |
| 3 | Conversa não salva | Exportar da memória | UX melhor que erro |
| 4 | Formato padrão (sem extensão) | Markdown | Mais legível |
| 5 | Flag `-f` vs extensão | Flag tem prioridade | Explícito vence implícito |
| 6 | Separar `m.feedback` de `m.err` | Sim, refatorar | Melhor semântica |

---

## 7. Conclusão

O plano original é um bom ponto de partida mas requer:

1. **Correção de inconsistências** com o código atual (interface `FullHistoryStore`)
2. **Adição de plano de testes** (crítico para qualidade)
3. **Definição de comportamentos de borda** (arquivo existente, conversa não salva)
4. **Sanitização de entrada** (path traversal, caracteres inválidos)
5. **Separação semântica** de erros vs feedback

Estimativa de esforço revisada: **2-3 dias** de desenvolvimento + **1 dia** de testes.
