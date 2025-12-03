# Refactoring/Design Plan: Gerenciamento de Conversas e Favoritos

## 1\. Executive Summary & Goals

O objetivo primário é estender a funcionalidade de **`history`** (histórico de conversas) para incluir recursos de gerenciamento, manipulação e um mecanismo de favoritos, melhorando significativamente a **usabilidade** e **capacidade de organização** do aplicativo CLI.

### Key Goals:

1.  **Implementar Operações de Gerenciamento:** Adicionar comandos para exportar, deletar, mover e renomear conversas.
2.  **Desenvolver Mecanismo de Favoritos:** Criar um menu de conversas favoritas persistente, permitindo acesso rápido.
3.  **Melhorar a Interação do Usuário:** Integrar as novas funcionalidades ao TUI (Terminal User Interface) para uma experiência mais intuitiva.

-----

## 2\. Current Situation Analysis

O projeto utiliza o pacote **`internal/history`** para persistir as conversas em arquivos JSON (um por conversa) no diretório `~/.geminiweb/history`. O `Store` atual (`internal/history/store.go`) já possui operações básicas de CRUD, como `CreateConversation`, `GetConversation`, `ListConversations`, `AddMessage`, `UpdateMetadata`, `DeleteConversation` e `ClearAll`.

**Limitações Atuais:**

  * **Gerenciamento CLI Básico:** O comando `geminiweb history` só oferece `list`, `show`, `delete` e `clear`. As funcionalidades de exportação, renomeação e manipulação de ordem não existem.
  * **Ausência de Favoritos:** Não há um mecanismo para marcar ou gerenciar conversas favoritas.
  * **Modelo de Conversa Fixo:** A estrutura `history.Conversation` não suporta metadados adicionais, como um flag de "favorito" ou um campo para ordem de exibição.
  * **Interação em TUI Limitada:** A seleção de histórico (`internal/tui/history_selector.go`) é apenas para retomada; não há interface para gerenciar as conversas.

-----

## 3\. Proposed Solution / Refactoring Strategy

A estratégia proposta é introduzir um novo mecanismo de **metadados globais** para gerenciar a ordem e o status de favorito das conversas, mantendo a estrutura de conversas individualizadas (JSON) em `internal/history`.

### 3.1. High-Level Design / Architectural Overview

O novo design envolve:

1.  **Novo Arquivo de Metadados:** Criar um arquivo `history_meta.json` (ou similar) no diretório `~/.geminiweb/history` para armazenar a lista ordenada de IDs de conversas e metadados como a flag de `Favorito`.
2.  **Refatorar `history.Store`:** Adicionar métodos para manipular a ordem e a flag de favoritos, lendo e escrevendo no novo arquivo de metadados.
3.  **Implementar Novos Comandos CLI:** Adicionar subcomandos em `geminiweb history` para as novas funcionalidades (exportar, renomear, mover).
4.  **Estender TUI:** Atualizar o seletor de histórico ou criar um novo TUI de gerenciamento para as novas operações.

<!-- end list -->

```mermaid
graph TD
    subgraph CLI / TUI
        Cobra[geminiweb history <cmd>]
        ChatTUI[/history command]
        ManagerTUI[Novo TUI de Gerenciamento]
    end

    subgraph Internal Packages
        HStore[internal/history/Store]
        HConv[internal/history/Conversation.json]
        HMeta[internal/history/HistoryMeta.json (Novo)]
    end

    Cobra -->|Chama| HStore
    ChatTUI -->|Chama| HStore
    ManagerTUI -->|Chama| HStore

    HStore -->|Lê/Escreve| HConv
    HStore -->|Lê/Escreve Ordem/Favoritos| HMeta
```

### 3.2. Key Components / Modules

| Componente | Localização | Responsabilidades |
| :--- | :--- | :--- |
| **`HistoryMeta`** (Novo) | `internal/history/store.go` | Estrutura para armazenar a ordem de exibição e os favoritos (lista de IDs). |
| **`Store` Refatorado** | `internal/history/store.go` | Gerenciar a persistência/carregamento de `HistoryMeta`. Adicionar métodos para `ToggleFavorite`, `ReorderConversation`, `RenameConversation`. |
| **`Conversation`** Refatorado | `internal/history/store.go` | Adicionar campos transientes (não persistidos no JSON da conversa) como `IsFavorite`, `OrderIndex` (preenchidos a partir de `HistoryMeta`). *Alternativa: Adicionar a flag de `IsFavorite` ao JSON da conversa, se a ordem global for a única preocupação do metafile.* **Manter o metafile para ordem e favoritos é o ideal.**|
| **`history` Command** | `internal/commands/history.go` | Implementar `export`, `rename`, `move`, `favorite`. |
| **`ChatModel` Extendido** | `internal/tui/model.go` | Adicionar lógica para o comando `/favorite` dentro do chat. |

### 3.3. Detailed Action Plan / Phases

#### Phase 1: Data Model Refactoring & Core Store Logic (High Priority)

| Task | Rationale/Goal | Estimated Effort | Deliverable/Criteria for Completion |
| :--- | :--- | :--- | :--- |
| 1.1: **Definir `ConversationMeta` & `HistoryMeta`** | Criar as estruturas em `internal/history/store.go` para persistir ordem e favoritos globalmente. | S | Estruturas `ConversationMeta` e `HistoryMeta` definidas. |
| 1.2: **Implementar `LoadMeta` & `SaveMeta`** | Adicionar métodos ao `Store` para persistir e carregar o arquivo de metadados (`~/.geminiweb/history/meta.json`). | M | `LoadMeta` e `SaveMeta` funcionando, com inicialização default. |
| 1.3: **Refatorar `ListConversations`** | Usar a ordem definida em `HistoryMeta` para retornar a lista de conversas. Adicionar os campos `IsFavorite` e `OrderIndex` à struct `Conversation` (como campos *calculados* ou *populados*). | M | `ListConversations()` retorna conversas ordenadas, com `IsFavorite` preenchido. |
| 1.4: **Implementar `UpdateTitle` (Renomear)** | Refatorar para garantir que o título seja atualizado tanto no arquivo da conversa quanto no metafile (se necessário para busca/exibição). | S | Método `UpdateTitle(id, newTitle string)` em `Store` funcionando. |
| 1.5: **Implementar `ToggleFavorite`** | Adicionar método ao `Store` que altera o status `IsFavorite` no `HistoryMeta` e salva. | S | Método `ToggleFavorite(id string)` em `Store` funcionando. |
| 1.6: **Implementar `ReorderConversation`** | Adicionar método ao `Store` que altera a posição de um `id` na lista de ordem do `HistoryMeta`. | M | Método `ReorderConversation(id string, newIndex int)` em `Store` funcionando. |

#### Phase 2: CLI & Export Functionality (Medium Priority)

| Task | Rationale/Goal | Estimated Effort | Deliverable/Criteria for Completion |
| :--- | :--- | :--- | :--- |
| 2.1: **Adicionar `history rename <id> <new_title>`** | Permitir renomear via CLI. Reutiliza 1.4. | S | Comando CLI integrado. |
| 2.2: **Adicionar `history move <id> <index>`** | Permitir mover para nova posição na lista (por índice). Reutiliza 1.6. | S | Comando CLI integrado. |
| 2.3: **Adicionar `history favorite <id>`** | Permitir adicionar/remover de favoritos via CLI. Reutiliza 1.5. | S | Comando CLI integrado. |
| 2.4: **Implementar `history export <id> -f <format>`** | Permitir exportar conversas. Suportar `json` (nativa) e `markdown` (formatando os `Messages`). | M | Novo comando `export` e lógica de formatação de mensagens para Markdown. |

#### Phase 3: TUI Integration & Favorites Menu (Medium Priority)

| Task | Rationale/Goal | Estimated Effort | Deliverable/Criteria for Completion |
| :--- | :--- | :--- | :--- |
| 3.1: **Refatorar `history_selector.go`** | Adicionar filtros e ordenação por favoritos no seletor, e ações no TUI para renomear, deletar, e favoritar (usando `HistoryStore` estendido). | L | Seletor de histórico permite busca, filtro por favoritos, e ações de gerenciamento. |
| 3.2: **Implementar Comando `/favorite` no Chat** | Permitir que o usuário favorite a conversa atual sem sair do chat. | S | Comando `/favorite` adicionado e integração com `ToggleFavorite` (1.5). |
| 3.3: **Criar `FavoriteConversationsModel` (Menu Favoritos)** | Implementar um novo TUI ou sub-menu dentro do `chat` para exibir *apenas* as conversas favoritas e retomá-las. | M | Novo TUI/menu de favoritos acessível e funcional. |

### 3.4. Data Model Changes (if applicable)

O campo `Messages` em `history.Conversation` já contém `Role`, `Content`, `Thoughts` e `Timestamp`.

**Novas Estruturas (em `internal/history/store.go`):**

```go
// ConversationMeta armazena metadados globais por conversa
type ConversationMeta struct {
	ID         string `json:"id"`
	IsFavorite bool   `json:"is_favorite"`
	Title      string `json:"title"` // Copiar o título para o metafile para evitar ler todos os arquivos ao listar
}

// HistoryMeta armazena a ordem e os favoritos
type HistoryMeta struct {
	// A lista de IDs na ordem em que devem ser exibidos
	Order   []string                      `json:"order"` 
	MetaMap map[string]*ConversationMeta  `json:"meta_map"`
	// Outros metadados globais
	Version int `json:"version"` // Para migração futura
}

// Conversa atualizada para facilitar o TUI (campos não persistidos no arquivo JSON individual)
type Conversation struct {
	// ... campos existentes (ID, Title, Model, CreatedAt, UpdatedAt, Messages, CID, RID, RCID)
	// Adicionar:
	IsFavorite bool `json:"-"` // Preenchido a partir do HistoryMeta
	OrderIndex int  `json:"-"` // Posição na lista Order do HistoryMeta
}
```

### 3.5. API Design / Interface Changes (if applicable)

**Modificações na Interface `HistoryStoreInterface` (`internal/tui/model.go`):**

```go
// HistoryStoreInterface (existing methods)
// ...
// New methods:
UpdateTitle(id, title string) error
DeleteConversation(id string) error // Already exists, but will need to update metafile
ToggleFavorite(id string) error
ReorderConversation(id string, newIndex int) error
```

-----

## 4\. Key Considerations & Risk Mitigation

### 4.1. Technical Risks & Challenges

| Risco | Descrição | Mitigação |
| :--- | :--- | :--- |
| **Consistência de Dados** | Inconsistência entre os arquivos `Conversation.json` e `HistoryMeta.json` (ex: um conversa deletada ainda referenciada no metafile). | `DeleteConversation` deve sempre remover a entrada do metafile. `ListConversations` deve limpar o metafile de IDs órfãos (se a conversa não existir no disco) durante o carregamento. |
| **Conflito de Escrita** | Múltiplas operações TUI/CLI tentando escrever o `HistoryMeta.json` simultaneamente (menos provável em um app CLI single-user). | O `Store` deve usar um `sync.Mutex` ao ler/escrever o arquivo `HistoryMeta.json` (já é feito no `Store.mu`). |
| **Exportação Markdown** | Conversões complexas de conteúdo (código, tabelas) do formato de resposta para Markdown limpo para exportação. | Reutilizar o pacote `internal/render` para a lógica de formatação, garantindo que o output seja limpo e portátil. |

### 4.2. Dependencies

  * **`internal/history/store.go`**: Depende da implementação correta e thread-safe de `LoadMeta`, `SaveMeta`, `ToggleFavorite`, `ReorderConversation` para todos os novos comandos.
  * **`internal/commands/history.go`**: Depende da nova interface do `Store` para implementar os comandos `rename`, `move`, `favorite`.
  * **`internal/tui/model.go`**: Depende do `Store` para o comando `/favorite`.

### 4.3. Non-Functional Requirements (NFRs) Addressed

| NFR | Como o Plano Contribui |
| :--- | :--- |
| **Usabilidade** | Novos comandos CLI e integração TUI (favoritos, renomear, mover) tornam o gerenciamento de conversas mais fácil e intuitivo. |
| **Confiabilidade** | O mecanismo de metadados garante que o estado de Favorito/Ordem seja persistente e recuperável após o reinício da aplicação. A lógica de mitigação de consistência de dados (4.1) aumenta a confiabilidade. |
| **Manutenibilidade** | A separação da ordem/favoritos (`HistoryMeta`) do conteúdo da conversa (`Conversation.json`) mantém o princípio de separação de preocupações e facilita futuras extensões. |

-----

## 5\. Success Metrics / Validation Criteria

1.  **Funcionalidade Básica:** Todos os novos comandos CLI (`rename`, `move`, `favorite`, `export`) são implementados e funcionam conforme o esperado (testados com unit tests no pacote `internal/history`).
2.  **Persistência de Favoritos:** O status de favorito de uma conversa é mantido após o fechamento e reabertura do aplicativo.
3.  **Ordenação de Conversas:** A ordem de exibição das conversas (`ListConversations`) reflete a ordem definida no `HistoryMeta.json` e a operação `move` funciona corretamente.
4.  **Integração TUI:** O TUI de chat e o seletor de histórico refletem o status de favorito e a ordem das conversas, e permitem ativar/desativar favoritos.

-----

## 6\. Assumptions Made

  * O formato de exportação para **Markdown** será uma simples concatenação formatada das mensagens, usando `internal/render` para a formatação final do conteúdo.
  * A ordenação será baseada em **índices de array** (começando em 0). A operação `move` exigirá o ID da conversa e o novo índice.
  * O arquivo `HistoryMeta.json` é a **fonte da verdade** para ordem e status de favoritos.

-----

## 7\. Open Questions / Areas for Further Investigation

  * **Exportação de Metadados:** Deve-se incluir os metadados da API (`CID`, `RID`, `RCID`) no arquivo de exportação (ex: JSON)? *Decisão: Incluir no JSON, não no Markdown.*
  * **Gestão de Metadados Órfãos:** A limpeza de referências inválidas no `HistoryMeta.json` deve ser feita de forma silenciosa ou apenas em caso de erro? *Decisão: Limpeza silenciosa durante o carregamento de `ListConversations`.*
  * **Comando de Exportação:** Qual deve ser o *output default* se o formato não for especificado? *Decisão: `markdown` é o mais útil para o usuário, mas JSON é o mais fiel. Usar `markdown` como default para usabilidade.*