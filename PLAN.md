# Refactoring/Design Plan: Integração de Gems e Atalho no Modo Chat

## 1\. Executive Summary & Goals

O objetivo primário é integrar a funcionalidade de gerenciamento e seleção de Gems (Personas do lado do servidor) diretamente no ambiente de chat TUI (Terminal User Interface) e adicionar um atalho (`/gems`) para acessá-la de forma rápida, complementando o comando CLI `geminiweb gems list`.

### Key Goals:

1.  **Acesso Rápido a Gems no TUI:** Implementar o comando `/gems` dentro do `internal/tui/model.go` para abrir o seletor de Gems.
2.  **Transição Sem Perda de Contexto:** Permitir que o usuário selecione ou altere o Gem ativo sem sair da sessão de chat em andamento.
3.  **Atualização de Sessão:** Atualizar o `GemID` da `ChatSession` ativa (em `internal/api/session.go`) e o cabeçalho do TUI para refletir o Gem selecionado.
4.  **UX no Chat:** Adicionar o nome do Gem ativo ao cabeçalho do chat TUI para feedback visual.

## 2\. Current Situation Analysis

O projeto já implementa a gestão completa de Gems:

  * **API:** O pacote `internal/api` possui `FetchGems`, `CreateGem`, `UpdateGem`, e `DeleteGem` (em `internal/api/gems.go`) que usam `BatchExecute` para interagir com a API.
  * **Cliente/Sessão:** `internal/api/client.go` e `internal/api/session.go` já têm métodos (`SetGem`, `GetGemID`, `WithGemID`) para configurar um Gem em uma sessão.
  * **TUI de Gerenciamento:** O `internal/tui/gems_model.go` implementa um seletor TUI interativo para listar e selecionar Gems (usado pelo comando CLI `geminiweb gems list`).
  * **Comandos CLI:** O `internal/commands/gems.go` já lida com o fluxo CLI (`gems list`, `gems create`, etc.).
  * **Chat TUI:** O `internal/tui/model.go` é o núcleo do chat interativo, mas **não** possui o comando `/gems` implementado nem a lógica de transição para o seletor de Gems.

**Limitação Atual:** O usuário precisa sair do chat TUI, executar `geminiweb gems list` e iniciar um **novo** chat para usar um Gem; não há como mudar dinamicamente.

## 3\. Proposed Solution / Refactoring Strategy

A estratégia consiste em integrar **a funcionalidade de seleção de Gems** do TUI existente como um overlay leve dentro do `Model` principal do chat (em `internal/tui/model.go`).  
**Não** será usado o `GemsModel` como sub-modelo completo, pois ele é acoplado ao fluxo `RunGemsTUI`/`tea.NewProgram`. Em vez disso, a lógica de navegação, filtragem e renderização será replicada/adaptada diretamente no chat, mantendo a transição sem perda de contexto.

### 3.1. High-Level Design / Architectural Overview

O `internal/tui/model.go` será o orquestrador. Ele introduzirá um novo estado (`selectingGem = true`) para mostrar o seletor de Gems como um overlay.

```mermaid
graph TD
    A[internal/tui/model.go]
    C[internal/api/GeminiClientInterface]
    D[internal/api/ChatSessionInterface]

    A -- "handleKeyMsg /gems" --> A_SetState[A.selectingGem = true]
    A_SetState --> A_LoadCmd[A.loadGemsForChat() (async)]
    A_LoadCmd --> C
    C -- FetchGems --> A
    A -- "Gem selecionado" --> D_SetGem[D.SetGem(newID)]
    D_SetGem --> A_UpdateHeader[A.activeGemName = newName]
```

### 3.2. Key Components / Modules

| Componente | Localização | Modificação | Responsabilidades |
| :--- | :--- | :--- | :--- |
| **`Model` (Chat)** | `internal/tui/model.go` | **Principal (Extensão)** | Implementar o comando `/gems`, gerenciar o estado `selectingGem`, e orquestrar a exibição/esconder do seletor. Receber o resultado e atualizar a `session`. |
| **`updateGemSelection`** | `internal/tui/model.go` | **Novo Método** | Lógica de atualização e renderização do seletor de Gems (incluindo navegação, filtragem, seleção) dentro do contexto do `Model` principal. |
| **`gemsLoadedForChatMsg`** | `internal/tui/model.go` | **Novo Message Type** | Mensagem assíncrona para notificar o `Model` que os Gems foram carregados via `client.FetchGems()`. |
| **`ChatSession`** | `internal/api/session.go` | **Existente** | O método `SetGem(gemID string)` já lida com a configuração do Gem no contexto da sessão (usado para gerar o payload correto). |
| **`GemsModel` (referência)** | `internal/tui/gems_model.go` | **Sem integração direta** | Serve como referência de UX e lógica (navegação/filtragem); a implementação efetiva do overlay fica no `Model` principal. |

### 3.3. Detailed Action Plan / Phases

**Escala de esforço:** S = até ~0,5 dia, M = 1–2 dias, L = 3+ dias.

#### Phase 1: Integração de Estado e Comando (High Priority)

| Task | Rationale/Goal | Effort | Deliverable/Criteria for Completion |
| :--- | :--- | :--- | :--- |
| 1.1: **Adicionar Estado de Seleção** | Rastrear o modo de seleção de Gem. | S | Adicionar `selectingGem bool`, `gemsList []*models.Gem`, `gemsCursor int`, `gemsFilter string`, `gemsLoading bool`, `activeGemName string` a `internal/tui/model.go:Model`. |
| 1.2: **Definir `gemsLoadedForChatMsg`** | Mensagem para carregar Gems de forma assíncrona. | S | Adicionar `gemsLoadedForChatMsg` struct a `internal/tui/model.go`. |
| 1.3: **Implementar `loadGemsForChat()`** | Função para buscar Gems na API. | M | Novo método `loadGemsForChat()` em `internal/tui/model.go` que chama `client.FetchGems(false)`. |
| 1.4: **Registrar `/gems` no `Update`** | Ativar o modo de seleção de Gem. | S | Adicionar `case "gems", "gem": m.selectingGem = true; return m, m.loadGemsForChat()` no `switch` de comandos em `internal/tui/model.go:Update`. |
| 1.5: **Implementar `updateGemSelection()`** | Lógica de navegação/seleção de Gems como um sub-loop de `Update`. | M | Novo método `(m Model) updateGemSelection(msg tea.Msg)` para gerenciar a lógica da overlay de seleção. |
| 1.6: **Implementar `renderGemSelector()`** | Visualização de seleção de Gems (overlay modal). | M | Novo método `(m Model) renderGemSelector()` para exibir a lista de Gems e o filtro. |

#### Phase 2: Fluxo de Dados e UX (Medium Priority)

| Task | Rationale/Goal | Effort | Deliverable/Criteria for Completion |
| :--- | :--- | :--- | :--- |
| 2.1: **Atualizar Sessão na Seleção** | Persistir a escolha do Gem. | S | No `updateGemSelection`, após a seleção, chamar `m.session.SetGem(selectedGem.ID)` e atualizar `m.activeGemName`. |
| 2.2: **Exibir Gem Ativo no Cabeçalho** | Fornecer feedback visual constante. | S | Modificar `internal/tui/model.go:View` para incluir `m.activeGemName` no `headerStyle`. |
| 2.3: **Resolver Gem Inicial** | Mostrar o Gem ativo já no início da sessão. | S | Ao criar o model, se `session.GetGemID()` != "", resolver o nome via cache (`client.Gems()/GetGem`) ou `FetchGems(false)` e definir `m.activeGemName`. Cobre Gem definido por `--gem` ou estado prévio na mesma execução. |
| 2.4: **Tratamento de Cancelamento** | Permitir que o usuário saia do seletor com `Esc` sem alterar o Gem. | S | `updateGemSelection` deve resetar o estado `selectingGem` no `Esc`. |
| 2.5: **Atualizar Ajuda/Docs** | Tornar o recurso descobrível. | S | Atualizar a ajuda do comando `chat` e/ou `README.md` para mencionar `/gems` e o indicador de Gem ativo no cabeçalho. |
| 2.6: **Cobertura de Testes** | Evitar regressões no TUI. | S | Adicionar/ajustar testes em `internal/tui/model_test.go` para: abrir `/gems`, filtrar, selecionar, cancelar e resolver Gem inicial. |

#### Phase 3: Refatoração e Robustez (Low Priority)

| Task | Rationale/Goal | Effort | Deliverable/Criteria for Completion |
| :--- | :--- | :--- | :--- |
| 3.1: **Persistir GemID no Histórico (Opcional)** | Permitir retomar conversas com o mesmo Gem. | M | Adicionar `GemID` a `history.Conversation`, salvar ao trocar Gem e restaurar ao recriar/switchar sessão. Garantir compatibilidade com históricos antigos (campo opcional). |
| 3.2: **Limpeza de Código** | Consolidar a implementação final. | S | Remover dependências indiretas do `GemsModel` e manter apenas a lógica necessária no `Model` principal, evitando acoplamento/circularidade. |

### 3.4. Data Model Changes

Nenhum. A interface `ChatSessionInterface` (em `internal/tui/model.go`) já expõe `SetGem(gemID string)` e `GetGemID() string`, e o `api.ChatSession` já implementa isso.

### 3.5. API Design / Interface Changes

**Interface Modificada/Revisada (em `internal/tui/model.go`):**

```go
type ChatSessionInterface interface {
    // ... métodos existentes
    SetGem(gemID string)
    GetGemID() string
}

// Sem alteração real, apenas confirmação do que é exposto.
```

## 4\. Key Considerations & Risk Mitigation

### 4.1. Technical Risks & Challenges

| Risco | Descrição | Mitigação |
| :--- | :--- | :--- |
| **Re-autenticação** | A busca de Gems (`FetchGems`) pode falhar devido a cookies expirados. | A chamada a `FetchGems` usa o `GeminiClient`, que já possui auto-refresh de cookies via browser (`RefreshFromBrowser`). Se falhar, o erro deve ser propagado para o TUI. |
| **Performance de Load** | O carregamento de todos os Gems (`FetchGems`) pode ser lento (chamada de rede). | A operação deve ser executada de forma assíncrona (`tea.Cmd`) para não bloquear a UI, com um indicador de carregamento (spinner) visível (`m.gemsLoading`). |
| **UX de Overlay** | O seletor de Gems precisa ser renderizado como um overlay para não perturbar a interface do chat. | O `renderGemSelector()` renderizará a lista de Gems de forma centralizada e em uma caixa delimitada, com o `View` principal verificando `m.selectingGem` e renderizando o overlay por cima do chat (sem a biblioteca `bubbles/list` para manter o controle total do layout). |

### 4.2. Dependencies

  * **Task 1.3:** Depende de `client.FetchGems(false)` (existe em `internal/api/gems.go`).
  * **Task 1.5, 1.6, 2.1:** Depende da lógica de navegação e filtragem do `GemsModel` (será re-implementada/adaptada em `internal/tui/model.go` para ser leve).

### 4.3. Non-Functional Requirements (NFRs) Addressed

| NFR | Como o Plano Contribui |
| :--- | :--- |
| **Usabilidade** | Atalho `/gems` simplifica a troca de persona. O feedback visual (`m.activeGemName` no cabeçalho) mantém o contexto do usuário. |
| **Performance** | O carregamento de Gems é assíncrono para não travar a UI (Task 1.3). |
| **Manutenibilidade**| A lógica de seleção é isolada no método `updateGemSelection`, seguindo o padrão de sub-modelos do Bubble Tea, mantendo o `Model` do chat limpo. |

## 5\. Success Metrics / Validation Criteria

1.  O comando `/gems` é reconhecido e abre o seletor.
2.  A lista de Gems é carregada de forma assíncrona, e um Gem pode ser selecionado com `Enter`.
3.  Após a seleção, o seletor desaparece, e o nome do Gem aparece no cabeçalho do chat.
4.  O GemID da `ChatSession` é atualizado corretamente, e o próximo `SendMessage` usa a nova persona.
5.  Pressionar `Esc` no seletor fecha o overlay e retorna ao chat sem alterar o Gem ativo.

## 6\. Assumptions Made

  * O `GeminiClient` está inicializado e autenticado quando o chat é iniciado.
  * A busca de Gems (`FetchGems`) é relativamente rápida; o overlay deve mostrar spinner enquanto carrega.
  * Gems são carregados sob demanda ao abrir `/gems` e podem ser reusados de cache quando disponível; re-fetch é aceitável quando necessário.

## 7\. Open Questions / Areas for Further Investigation

| Questão | Decisão |
| :--- | :--- |
| Devo usar o `GemsModel` inteiro como sub-modelo? | **Não.** O `GemsModel` está acoplado ao `RunGemsTUI` (que chama `tea.NewProgram`). É melhor replicar a lógica de seleção/renderização diretamente em `internal/tui/model.go` (`updateGemSelection` e `renderGemSelector`) para um controle de overlay mais leve e mais granular. |
| Como lidar com o estado do filtro de Gems? | Implementar uma filtragem simples por substring no nome/descrição, usando `m.gemsFilter` e atualizando `m.gemsCursor` (Task 1.5). |
| Devo mostrar o prompt do Gem no seletor? | Apenas o nome e uma descrição truncada. O prompt completo será muito grande para um overlay rápido. |
