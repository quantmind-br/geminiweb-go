# Refactoring/Design Plan: Gerenciamento de Conversas e Favoritos

## 1. Executive Summary & Goals

O objetivo primário é estender a funcionalidade de **`history`** (histórico de conversas) para incluir recursos de gerenciamento, manipulação e um mecanismo de favoritos, melhorando significativamente a **usabilidade** e **capacidade de organização** do aplicativo CLI.

### Key Goals:

1. **Melhorar a Ergonomia de Uso:** Introduzir aliases amigáveis (`@last`, índices numéricos) para evitar digitação de UUIDs.
2. **Implementar Operações de Gerenciamento:** Adicionar comandos para exportar, deletar, renomear e favoritar conversas com confirmações apropriadas.
3. **Desenvolver Mecanismo de Favoritos:** Criar sistema de favoritos persistente com feedback visual claro.
4. **Reordenação Visual via TUI:** Implementar interface visual interativa para reorganizar conversas (sem comando CLI de `move`).
5. **Separar Contextos de Uso:** Seletor rápido (`/history`) vs. gerenciador completo (`/manage`).

---

## 2. Current Situation Analysis

O projeto utiliza o pacote **`internal/history`** para persistir as conversas em arquivos JSON (um por conversa) no diretório `~/.geminiweb/history`. O `Store` atual (`internal/history/store.go`) já possui operações básicas de CRUD, como `CreateConversation`, `GetConversation`, `ListConversations`, `AddMessage`, `UpdateMetadata`, `DeleteConversation` e `ClearAll`.

**Limitações Atuais:**

* **Ergonomia Ruim:** Comandos exigem UUIDs completos, dificultando o uso.
* **Gerenciamento CLI Básico:** O comando `geminiweb history` só oferece `list`, `show`, `delete` e `clear`. As funcionalidades de exportação e renomeação não existem.
* **Ausência de Favoritos:** Não há um mecanismo para marcar ou gerenciar conversas favoritas.
* **Sem Feedback Visual:** Operações não confirmam sucesso/falha de forma clara.
* **Interação em TUI Limitada:** A seleção de histórico (`internal/tui/history_selector.go`) é apenas para retomada; não há interface para gerenciar as conversas.
* **Sem Reordenação:** Não é possível reorganizar a ordem das conversas.

---

## 3. Proposed Solution / Refactoring Strategy

A estratégia proposta introduz melhorias de UX em camadas, começando por quick wins de ergonomia, seguido por um mecanismo de **metadados globais** para gerenciar ordem e favoritos, e finalmente um **TUI de gerenciamento visual**.

### 3.1. High-Level Design / Architectural Overview

O novo design envolve:

1. **Sistema de Aliases:** Resolver `@last`, `@1`, `@2` para IDs reais antes de executar comandos.
2. **Novo Arquivo de Metadados:** Criar `meta.json` no diretório `~/.geminiweb/history` para armazenar ordem e favoritos.
3. **Refatorar `history.Store`:** Adicionar métodos para manipular ordem e favoritos.
4. **Comandos CLI Ergonômicos:** Suporte a aliases, confirmações, e feedback claro.
5. **TUI de Gerenciamento:** Interface visual para reordenar (teclas `j/k` ou `↑/↓`), favoritar, renomear e deletar.

```mermaid
graph TD
    subgraph "CLI Layer"
        AliasResolver[Alias Resolver<br/>@last, @1, @2 → UUID]
        HistoryCmd[geminiweb history <cmd>]
    end

    subgraph "TUI Layer"
        QuickSelector[/history<br/>Seletor Rápido]
        ManagerTUI[/manage<br/>Gerenciador Visual]
    end

    subgraph "Internal Packages"
        HStore[internal/history/Store]
        HConv[~/.geminiweb/history/*.json]
        HMeta[~/.geminiweb/history/meta.json]
    end

    HistoryCmd --> AliasResolver
    AliasResolver --> HStore
    QuickSelector --> HStore
    ManagerTUI --> HStore

    HStore -->|Lê/Escreve| HConv
    HStore -->|Lê/Escreve Ordem/Favoritos| HMeta
```

### 3.2. Key Components / Modules

| Componente | Localização | Responsabilidades |
| :--- | :--- | :--- |
| **`AliasResolver`** (Novo) | `internal/history/resolver.go` | Converter aliases (`@last`, `@1`, substring de título) para UUIDs. |
| **`HistoryMeta`** (Novo) | `internal/history/meta.go` | Estrutura para armazenar ordem de exibição e favoritos. |
| **`Store` Refatorado** | `internal/history/store.go` | Gerenciar persistência de `HistoryMeta`. Métodos para `ToggleFavorite`, `MoveConversation`, `UpdateTitle`. |
| **`HistoryManagerModel`** (Novo) | `internal/tui/history_manager.go` | TUI visual para gerenciamento completo com reordenação por teclas. |
| **`history` Command** | `internal/commands/history.go` | Comandos `export`, `rename`, `favorite`, `delete` com aliases e confirmações. |
| **`ChatModel` Extendido** | `internal/tui/model.go` | Comandos `/favorite` e `/manage` dentro do chat. |

### 3.3. Detailed Action Plan / Phases

#### Phase 0: UX Quick Wins & Ergonomia (High Priority)

*Foco: Reduzir fricção de uso imediatamente, antes de features complexas.*

| Task | Rationale/Goal | Effort | Deliverable |
| :--- | :--- | :--- | :--- |
| 0.1: **Implementar `AliasResolver`** | Permitir `@last` (última conversa), `@1`, `@2` (por índice), e busca por substring do título. | M | `ResolveAlias(alias string) (string, error)` retorna UUID ou erro claro. |
| 0.2: **Integrar Aliases nos Comandos Existentes** | Aplicar resolver em `history show`, `history delete`. | S | Comandos existentes aceitam aliases. |
| 0.3: **Adicionar Confirmação em `history delete`** | Prevenir deleção acidental. Mostrar resumo antes de confirmar. | S | Prompt: "Deletar 'Título' (15 msgs, 3 dias)? [y/N]" com `--force` para bypass. |
| 0.4: **Melhorar Output de `history list`** | Mostrar índices numéricos, indicador de favorito (★), e timestamps relativos. | S | Lista com formato: `[1] ★ Título da Conversa (há 2h)` |
| 0.5: **Adicionar Exemplos ao `--help`** | Melhorar descoberta de funcionalidades. | S | Help mostra exemplos práticos com aliases. |
| 0.6: **Feedback de Sucesso/Erro Consistente** | Toda operação deve confirmar resultado. | S | Mensagens como "✓ Conversa deletada" ou "✗ Erro: conversa não encontrada". |

**Aliases Suportados:**

| Alias | Resolução |
| :--- | :--- |
| `@last` | Última conversa modificada |
| `@first` | Primeira conversa na lista |
| `1`, `2`, `3` | Por índice na lista (1-based) |
| `"substring"` | Busca fuzzy no título (erro se múltiplos matches) |

#### Phase 1: Data Model & Core Store Logic (High Priority)

| Task | Rationale/Goal | Effort | Deliverable |
| :--- | :--- | :--- | :--- |
| 1.1: **Definir `ConversationMeta` & `HistoryMeta`** | Estruturas para persistir ordem e favoritos globalmente. | S | Structs definidas em `internal/history/meta.go`. |
| 1.2: **Implementar `LoadMeta` & `SaveMeta`** | Persistir/carregar metadados com inicialização automática. | M | Métodos funcionando com criação de arquivo default. |
| 1.3: **Refatorar `ListConversations`** | Retornar conversas na ordem do metafile, com `IsFavorite` e `OrderIndex` populados. Limpar IDs órfãos silenciosamente. | M | Lista ordenada com campos calculados preenchidos. |
| 1.4: **Implementar `UpdateTitle`** | Atualizar título no arquivo da conversa E no metafile. | S | `UpdateTitle(id, newTitle string) error` |
| 1.5: **Implementar `ToggleFavorite`** | Alternar status de favorito no metafile. | S | `ToggleFavorite(id string) (isFavorite bool, err error)` |
| 1.6: **Implementar `MoveConversation`** | Mover conversa para nova posição na ordem. | M | `MoveConversation(id string, newIndex int) error` |
| 1.7: **Implementar `SwapConversations`** | Trocar posição de duas conversas (útil para TUI). | S | `SwapConversations(id1, id2 string) error` |

#### Phase 2: CLI Commands & Export (Medium Priority)

| Task | Rationale/Goal | Effort | Deliverable |
| :--- | :--- | :--- | :--- |
| 2.1: **Adicionar `history rename <ref> <title>`** | Renomear via CLI com suporte a aliases. | S | Comando com feedback: "✓ Renomeado para 'Novo Título'" |
| 2.2: **Adicionar `history favorite <ref>`** | Toggle de favorito via CLI. | S | Feedback: "★ Adicionado aos favoritos" ou "☆ Removido dos favoritos" |
| 2.3: **Adicionar `history export <ref> [-o file] [-f format]`** | Exportar conversa para arquivo. | M | Formatos: `markdown` (default), `json`. Auto-detecta por extensão do arquivo. |
| 2.4: **Melhorar `history delete`** | Confirmação visual com resumo da conversa. | S | Mostra título, mensagens, data antes de confirmar. |
| 2.5: **Adicionar `history search <query>`** | Buscar em títulos e conteúdo das conversas. | M | Lista conversas que contêm o termo, com snippet do match. |

**Nota:** Não há comando `history move` no CLI. A reordenação é feita exclusivamente via TUI visual (Phase 3).

#### Phase 3: TUI de Gerenciamento Visual (Medium Priority)

| Task | Rationale/Goal | Effort | Deliverable |
| :--- | :--- | :--- | :--- |
| 3.1: **Criar `HistoryManagerModel`** | TUI completo para gerenciamento com reordenação visual. | L | Novo arquivo `internal/tui/history_manager.go`. |
| 3.2: **Implementar Reordenação por Teclas** | `Ctrl+↑/↓` ou `Ctrl+j/k` move item selecionado na lista. | M | Feedback visual imediato da nova posição. |
| 3.3: **Ações Inline no Gerenciador** | Teclas de atalho: `f` favoritar, `r` renomear, `d` deletar, `e` exportar. | M | Ações executam sem sair do TUI. |
| 3.4: **Renomeação Inline** | Pressionar `r` abre campo de edição inline no item. | M | Edição in-place com Enter para confirmar, Esc para cancelar. |
| 3.5: **Filtros e Busca no TUI** | `/` para buscar, `Tab` para alternar entre Todos/Favoritos. | M | Filtro em tempo real com highlight de matches. |
| 3.6: **Comando `/manage` no Chat** | Abre o gerenciador sem sair do chat. | S | Ao fechar gerenciador, retorna ao chat. |
| 3.7: **Indicadores Visuais de Favoritos** | Estrela (★) ao lado de conversas favoritas em todas as listas. | S | Consistência visual em `/history`, `/manage`, e CLI. |

**Layout do TUI de Gerenciamento:**

```
┌─ Gerenciador de Conversas ─────────────────────────────────┐
│ [Todos] [★ Favoritos]                          Buscar: ___ │
├────────────────────────────────────────────────────────────┤
│  1. ★ Projeto Go - Refatoração API         (há 2h)    ↕   │
│  2.   Debug de conexão WebSocket           (ontem)    ↕   │
│▸ 3.   Análise de código legado             (3 dias)   ↕   │  ← Selecionado
│  4. ★ Estudos de ML com Python             (1 sem)    ↕   │
│  5.   Configuração de ambiente             (2 sem)    ↕   │
├────────────────────────────────────────────────────────────┤
│ ↑↓: Navegar  Ctrl+↑↓: Mover  f: Favoritar  r: Renomear    │
│ d: Deletar   e: Exportar     /: Buscar     Enter: Abrir   │
│ Tab: Filtrar                 q: Voltar     ?: Ajuda       │
└────────────────────────────────────────────────────────────┘
```

**Fluxo de Reordenação:**

1. Usuário navega até a conversa desejada com `↑/↓`
2. Pressiona `Ctrl+↑` ou `Ctrl+↓` para mover a conversa
3. A lista é atualizada visualmente em tempo real
4. Ao mover, o metafile é salvo automaticamente
5. Indicador visual `↕` mostra que o item é "movível"

**Fluxo de Renomeação Inline:**

1. Usuário pressiona `r` no item selecionado
2. O título se transforma em campo de texto editável
3. Texto atual é selecionado para fácil substituição
4. `Enter` confirma, `Esc` cancela
5. Feedback: linha pisca verde brevemente ao salvar

#### Phase 4: Integração Final & Polish (Low Priority)

| Task | Rationale/Goal | Effort | Deliverable |
| :--- | :--- | :--- | :--- |
| 4.1: **Comando `/favorite` no Chat** | Toggle de favorito da conversa atual sem sair. | S | Feedback no header: "★ Conversa Favorita" |
| 4.2: **Header do Chat com Status** | Mostrar ★ se a conversa atual é favorita. | S | Header atualiza dinamicamente. |
| 4.3: **Atalho `/favorites`** | Lista rápida apenas de favoritos para retomada. | S | Abre seletor filtrado por favoritos. |
| 4.4: **Onboarding para Novos Usuários** | Mensagem amigável quando `history list` está vazio. | S | "Nenhuma conversa salva. Use 'geminiweb chat' para começar." |
| 4.5: **Soft Delete com Trash** | Conversas deletadas vão para trash por 7 dias. | M | `history restore <ref>` recupera do trash. |

### 3.4. Data Model Changes

**Novas Estruturas (em `internal/history/meta.go`):**

```go
// ConversationMeta armazena metadados globais por conversa
type ConversationMeta struct {
    ID         string `json:"id"`
    Title      string `json:"title"`      // Cache do título para listagem rápida
    IsFavorite bool   `json:"is_favorite"`
}

// HistoryMeta armazena a ordem e os favoritos
type HistoryMeta struct {
    Version int                           `json:"version"` // Para migração futura
    Order   []string                      `json:"order"`   // IDs na ordem de exibição
    Meta    map[string]*ConversationMeta  `json:"meta"`    // Metadados por ID
}
```

**Campos Calculados em `Conversation` (não persistidos):**

```go
type Conversation struct {
    // ... campos existentes ...

    // Campos populados a partir do HistoryMeta (não salvos no JSON individual)
    IsFavorite bool `json:"-"` // Preenchido por ListConversations
    OrderIndex int  `json:"-"` // Posição na lista (1-based para exibição)
}
```

### 3.5. API Design / Interface Changes

**Novo: `AliasResolver` Interface:**

```go
// AliasResolver resolve referências amigáveis para IDs
type AliasResolver interface {
    // Resolve converte alias para UUID
    // Aliases: "@last", "@first", "1", "2", "substring do título"
    Resolve(ref string) (id string, err error)

    // MustResolve é como Resolve mas panic em erro (para testes)
    MustResolve(ref string) string
}
```

**Modificações na Interface `HistoryStoreInterface`:**

```go
type HistoryStoreInterface interface {
    // Métodos existentes...
    ListConversations() ([]*Conversation, error)
    GetConversation(id string) (*Conversation, error)
    DeleteConversation(id string) error

    // Novos métodos:
    UpdateTitle(id, title string) error
    ToggleFavorite(id string) (isFavorite bool, err error)
    MoveConversation(id string, newIndex int) error
    SwapConversations(id1, id2 string) error

    // Para exportação:
    ExportToMarkdown(id string) (string, error)
    ExportToJSON(id string) ([]byte, error)

    // Para busca:
    SearchConversations(query string) ([]*SearchResult, error)
}

type SearchResult struct {
    Conversation *Conversation
    MatchSnippet string // Trecho onde o termo foi encontrado
    MatchField   string // "title" ou "content"
}
```

---

## 4. Key Considerations & Risk Mitigation

### 4.1. Technical Risks & Challenges

| Risco | Descrição | Mitigação |
| :--- | :--- | :--- |
| **Consistência de Dados** | Metafile pode ficar inconsistente com arquivos de conversa. | `ListConversations` limpa IDs órfãos silenciosamente. `DeleteConversation` sempre atualiza metafile. |
| **Conflito de Alias** | Busca por substring pode retornar múltiplos matches. | Retornar erro claro: "Múltiplas conversas encontradas: [lista]. Use ID ou seja mais específico." |
| **Performance de Busca** | Busca em conteúdo pode ser lenta com muitas conversas. | Limitar busca a títulos por default. Flag `--content` para busca em mensagens. |
| **UX de Reordenação** | Usuário pode não descobrir Ctrl+↑/↓. | Mostrar dica na barra de status. Incluir `?` para tela de ajuda. |

### 4.2. Dependencies

* **Phase 0** não depende de outras phases e pode ser implementada imediatamente.
* **Phase 1** (Data Model) é pré-requisito para Phases 2, 3 e 4.
* **Phase 2** (CLI) e **Phase 3** (TUI) podem ser desenvolvidas em paralelo após Phase 1.
* **Phase 4** (Polish) depende de todas as anteriores.

### 4.3. Non-Functional Requirements (NFRs) Addressed

| NFR | Como o Plano Contribui |
| :--- | :--- |
| **Usabilidade** | Aliases eliminam necessidade de UUIDs. Confirmações previnem erros. TUI visual para reordenação é intuitivo. |
| **Descoberta** | Help com exemplos, onboarding para novos usuários, atalhos visíveis na barra de status. |
| **Feedback** | Toda operação confirma resultado. Indicadores visuais (★) são consistentes. |
| **Eficiência** | Ações via teclas de atalho no TUI. Aliases reduzem digitação. |
| **Confiabilidade** | Confirmações em operações destrutivas. Soft delete permite recuperação. |

---

## 5. Success Metrics / Validation Criteria

1. **Ergonomia:** Usuário consegue executar `history show @last` e `history delete 1` sem erros.
2. **Confirmações:** `history delete` sem `--force` sempre pede confirmação.
3. **Favoritos:** Status de favorito persiste após reinício e aparece em todas as listas com ★.
4. **Reordenação Visual:** No TUI, `Ctrl+↑/↓` move conversa e a nova ordem persiste.
5. **Feedback:** Toda operação CLI exibe mensagem de sucesso ou erro clara.
6. **Descoberta:** `history --help` mostra exemplos práticos com aliases.

---

## 6. Assumptions Made

* Aliases numéricos (`1`, `2`, `3`) são **1-based** para intuitividade (não 0-based).
* A busca por substring no título é **case-insensitive**.
* O formato de exportação default é **Markdown** (mais útil para usuário).
* A reordenação é **exclusivamente via TUI** (não há comando CLI `move`).
* O `HistoryMeta.json` é a **fonte da verdade** para ordem e favoritos.

---

## 7. Open Questions / Areas for Further Investigation

| Questão | Decisão |
| :--- | :--- |
| Exportação deve incluir metadados da API (`CID`, `RID`)? | Incluir no JSON, não no Markdown. |
| Limpeza de IDs órfãos deve ser silenciosa? | Sim, durante `ListConversations`. |
| Soft delete: quanto tempo manter no trash? | 7 dias, configurável via `config`. |
| Busca em conteúdo por default? | Não, apenas títulos. Flag `--content` para busca completa. |
| Atalho para mover no TUI: `Ctrl+↑/↓` ou `Shift+↑/↓`? | `Ctrl+↑/↓` (Shift pode conflitar com seleção de texto). |

---

## 8. Resumo de Comandos e Atalhos

### CLI Commands

| Comando | Descrição | Exemplo |
| :--- | :--- | :--- |
| `history list` | Listar conversas com índices e ★ | `history list --favorites` |
| `history show <ref>` | Mostrar conversa | `history show @last` |
| `history delete <ref>` | Deletar com confirmação | `history delete 1 --force` |
| `history rename <ref> <title>` | Renomear conversa | `history rename @last "Novo Nome"` |
| `history favorite <ref>` | Toggle favorito | `history favorite 1` |
| `history export <ref>` | Exportar conversa | `history export @last -o chat.md` |
| `history search <query>` | Buscar em títulos | `history search "API"` |

### TUI Commands (dentro do chat)

| Comando | Descrição |
| :--- | :--- |
| `/history` | Seletor rápido de conversas |
| `/manage` | Gerenciador visual completo |
| `/favorite` | Toggle favorito da conversa atual |
| `/favorites` | Listar apenas favoritos |

### TUI Manager Keybindings

| Tecla | Ação |
| :--- | :--- |
| `↑/↓` ou `j/k` | Navegar na lista |
| `Ctrl+↑/↓` ou `Ctrl+j/k` | Mover conversa selecionada |
| `Enter` | Abrir conversa |
| `f` | Toggle favorito |
| `r` | Renomear (inline) |
| `d` | Deletar (com confirmação) |
| `e` | Exportar |
| `/` | Buscar |
| `Tab` | Alternar filtro (Todos/Favoritos) |
| `q` ou `Esc` | Voltar ao chat |
| `?` | Mostrar ajuda |
