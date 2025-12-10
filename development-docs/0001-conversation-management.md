# TASKS.md - Gerenciamento de Conversas e Favoritos

## Project Briefing

**Objetivo:** Implementar sistema completo de gerenciamento de conversas com aliases ergonômicos, favoritos, reordenação visual, e comandos CLI/TUI aprimorados.

**Escopo:**
- Phase 0: Aliases (`@last`, índices numéricos) e UX quick wins
- Phase 1: Data model com HistoryMeta para ordem/favoritos
- Phase 2: Comandos CLI (rename, favorite, export, search)
- Phase 3: TUI de gerenciamento visual com reordenação
- Phase 4: Polish e integração final

**Arquivos Principais a Criar/Modificar:**
- `internal/history/meta.go` (novo) - Estruturas HistoryMeta
- `internal/history/resolver.go` (novo) - AliasResolver
- `internal/history/store.go` - Refatorar para usar meta.json
- `internal/history/export.go` (novo) - Exportação Markdown/JSON
- `internal/commands/history.go` - Novos subcomandos
- `internal/tui/history_manager.go` (novo) - TUI de gerenciamento

---

## Phase 0: UX Quick Wins & Ergonomia

### 0.1 Implementar AliasResolver
- [x] Criar `internal/history/resolver.go`
- [x] Implementar `Resolver` struct com método `Resolve(ref string) (string, error)`
- [x] Suportar aliases: `@last`, `@first`, índices numéricos (`1`, `2`, `3`)
- [x] Suportar busca por substring no título (case-insensitive)
- [x] Retornar erro claro em caso de múltiplos matches
- [x] Escrever testes em `resolver_test.go`

### 0.2 Integrar Aliases nos Comandos Existentes
- [x] Modificar `runHistoryShow` para usar `Resolver`
- [x] Modificar `runHistoryDelete` para usar `Resolver`
- [x] Testar comandos com aliases

### 0.3 Adicionar Confirmação em `history delete`
- [x] Implementar prompt de confirmação interativo
- [x] Mostrar resumo: título, mensagens, data
- [x] Adicionar flag `--force` para bypass
- [x] Testar fluxo de confirmação

### 0.4 Melhorar Output de `history list`
- [x] Adicionar índices numéricos na listagem
- [x] Mostrar indicador de favorito (★)
- [x] Converter timestamps para formato relativo ("há 2h", "ontem")
- [x] Formato: `[1] ★ Título da Conversa (há 2h)`

### 0.5 Adicionar Exemplos ao `--help`
- [x] Atualizar help de `history` com exemplos práticos
- [x] Documentar aliases suportados
- [x] Exemplos de uso real

### 0.6 Feedback de Sucesso/Erro Consistente
- [x] Padronizar mensagens com "✓" para sucesso e "✗" para erro
- [x] Aplicar em todos os comandos history

---

## Phase 1: Data Model & Core Store Logic

### 1.1 Definir ConversationMeta & HistoryMeta
- [x] Criar `internal/history/meta.go`
- [x] Definir `ConversationMeta` struct (ID, Title, IsFavorite)
- [x] Definir `HistoryMeta` struct (Version, Order, Meta map)
- [x] Adicionar campos calculados em `Conversation` (IsFavorite, OrderIndex)

### 1.2 Implementar LoadMeta & SaveMeta
- [x] Implementar `(*Store).loadMeta() (*HistoryMeta, error)`
- [x] Implementar `(*Store).saveMeta(*HistoryMeta) error`
- [x] Criar meta.json automaticamente se não existir
- [x] Path: `~/.geminiweb/history/meta.json`
- [x] Testes para load/save

### 1.3 Refatorar ListConversations
- [x] Carregar metadados junto com conversas
- [x] Ordenar por `HistoryMeta.Order`
- [x] Preencher `IsFavorite` e `OrderIndex` calculados
- [x] Limpar IDs órfãos silenciosamente
- [x] Adicionar novas conversas ao final da ordem
- [x] Testes de ordenação

### 1.4 Atualizar CreateConversation e DeleteConversation
- [x] `CreateConversation`: adicionar ID ao meta.json Order
- [x] `DeleteConversation`: remover ID do meta.json
- [x] Manter consistência de dados

### 1.5 Implementar ToggleFavorite
- [x] `(*Store).ToggleFavorite(id string) (bool, error)`
- [x] Retornar novo estado (favorito ou não)
- [x] Atualizar meta.json
- [x] Testes

### 1.6 Implementar MoveConversation
- [x] `(*Store).MoveConversation(id string, newIndex int) error`
- [x] Validar índice (1-based para usuário, 0-based interno)
- [x] Reordenar array Order
- [x] Testes de reordenação

### 1.7 Implementar SwapConversations
- [x] `(*Store).SwapConversations(id1, id2 string) error`
- [x] Trocar posições de duas conversas
- [x] Testes

---

## Phase 2: CLI Commands & Export

### 2.1 Adicionar `history rename`
- [x] Implementar `historyRenameCmd`
- [x] Suporte a aliases
- [x] Feedback: "✓ Renomeado para 'Novo Título'"
- [x] Testes

### 2.2 Adicionar `history favorite`
- [x] Implementar `historyFavoriteCmd`
- [x] Toggle de favorito
- [x] Feedback: "★ Adicionado aos favoritos" ou "☆ Removido"
- [x] Testes

### 2.3 Implementar Exportação
- [x] Criar `internal/history/export.go`
- [x] `ExportToMarkdown(id string) (string, error)`
- [x] `ExportToJSON(id string) ([]byte, error)`
- [x] Formato Markdown com headers, timestamps, roles
- [x] Adicionar `historyExportCmd` com flags `-o file` e `-f format`
- [x] Auto-detectar formato pela extensão do arquivo
- [x] Testes de exportação

### 2.4 Melhorar `history delete`
- [x] Confirmação visual com resumo completo
- [x] Mostrar título, contagem de mensagens, data
- [x] Testes do fluxo

### 2.5 Adicionar `history search`
- [x] `(*Store).SearchConversations(query string) ([]*SearchResult, error)`
- [x] Buscar em títulos por default
- [x] Flag `--content` para busca em mensagens
- [x] Mostrar snippet do match
- [x] Testes de busca

---

## Phase 3: TUI de Gerenciamento Visual

### 3.1 Criar HistoryManagerModel
- [x] Criar `internal/tui/history_manager.go`
- [x] Struct com list, filtro, modo (normal/rename/search)
- [x] Implementar Init, Update, View

### 3.2 Implementar Navegação e Lista
- [x] Renderizar lista com índices e favoritos (★)
- [x] Navegação com `↑/↓` ou `j/k`
- [x] Timestamps relativos
- [x] Indicador de seleção (▸)

### 3.3 Implementar Reordenação por Teclas
- [x] `Ctrl+↑/↓` ou `Ctrl+j/k` para mover item
- [x] Feedback visual imediato
- [x] Salvar ordem automaticamente
- [x] Indicador `↕` de item movível

### 3.4 Ações Inline
- [x] `f` para toggle favorito
- [x] `d` para deletar (com confirmação)
- [x] `e` para exportar
- [x] `Enter` para abrir conversa

### 3.5 Renomeação Inline
- [x] `r` abre campo de edição
- [x] Texto atual pré-selecionado
- [x] `Enter` confirma, `Esc` cancela
- [x] Feedback visual ao salvar

### 3.6 Filtros e Busca
- [x] `/` para abrir busca
- [x] `Tab` para alternar Todos/Favoritos
- [x] Filtro em tempo real
- [x] Highlight de matches

### 3.7 Comando `/manage` no Chat
- [x] Adicionar comando ao ChatModel
- [x] Retornar ao chat após fechar gerenciador
- [x] Integrar com conversas existentes

### 3.8 Layout e Estilo
- [x] Header com tabs [Todos] [★ Favoritos]
- [x] Barra de status com atalhos
- [x] Estilização consistente com tema

---

## Phase 4: Integração Final & Polish

### 4.1 Comando `/favorite` no Chat
- [x] Toggle de favorito da conversa atual
- [x] Feedback no header: "★ Conversa Favorita"

### 4.2 Header do Chat com Status de Favorito
- [x] Mostrar ★ se conversa é favorita
- [x] Atualização dinâmica

### 4.3 Atalho `/favorites`
- [x] Lista rápida apenas de favoritos (via `/manage` com Tab)
- [x] Seletor filtrado

### 4.4 Onboarding para Novos Usuários
- [x] Mensagem amigável quando lista está vazia
- [x] Dicas de uso

---

## Validation & Testing

- [x] Todos os testes passando: `make test`
- [x] Cobertura >= 80% nos novos arquivos
- [x] Lint sem erros: `make lint`
- [x] Build funcional: `make build`
- [x] Testes manuais de fluxo completo:
  - [x] `history list` mostra índices e favoritos
  - [x] `history show @last` funciona
  - [x] `history delete 1` pede confirmação
  - [x] `history rename @last "Novo"` funciona
  - [x] `history favorite 1` toggle funciona
  - [x] `history export @last -o test.md` funciona
  - [x] `/manage` no TUI funciona
  - [x] Reordenação visual funciona

---

## Status Legend

- `[ ]` Pending
- `[~]` In Progress
- `[x]` Completed

---

## Implementation Summary

### Completed: 2025-12-02

All phases of the conversation management and favorites system have been implemented successfully.

### Files Created
- `internal/history/meta.go` - Core meta structures and operations (favorites, order)
- `internal/history/resolver.go` - AliasResolver for @last, @first, numeric indices, substring search
- `internal/history/export.go` - Export to Markdown/JSON, search functionality, relative time formatting
- `internal/tui/history_manager.go` - Full TUI history manager with reordering, rename, search
- `internal/history/meta_test.go` - Tests for meta operations
- `internal/history/resolver_test.go` - Tests for resolver
- `internal/history/export_test.go` - Tests for export and search

### Files Modified
- `internal/history/store.go` - Added meta.json integration, computed fields
- `internal/commands/history.go` - Complete rewrite with new commands and aliases
- `internal/commands/history_test.go` - Updated tests for new behavior
- `internal/tui/model.go` - Added /manage and /favorite commands
- `internal/tui/model_test.go` - Added mock methods for new interfaces

### Key Features Delivered
1. **Aliases**: `@last`, `@first`, numeric indices (1, 2, 3...), substring search
2. **Favorites**: Toggle favorites, filter list, star indicator (★)
3. **Reordering**: Ctrl+↑/↓ in TUI, persistent meta.json
4. **Export**: Markdown and JSON formats with options
5. **Search**: Title and content search with snippets
6. **UX**: Confirmations, relative times, consistent feedback (✓/✗)
7. **TUI Manager**: Full-featured history management interface

### Test Coverage
All tests passing (76 tests across history, commands, and tui packages).
