# Refactoring/Design Plan: Modo Interativo para Gerenciamento de Gems (Personas)

## 1\. Executive Summary & Goals

O objetivo primário é estender a funcionalidade de listagem de **Gems** (`geminiweb gems list`) do CLI para um **Modo Interativo** mais completo, permitindo que o usuário visualize, filtre e, crucialmente, **inicie um chat com o Gem selecionado** diretamente da interface TUI.

### Key Goals:

1.  **Habilitar Chat Rápido:** Permitir que o usuário inicie uma sessão de chat com o Gem selecionado (tecla `c`) a partir da lista TUI.
2.  **Melhorar a Descoberta (UX):** Apresentar a lista de Gems em um formato interativo (`TUI - Text User Interface`) com busca em tempo real e visualização de detalhes.
3.  **Encapsular a Lógica de Seleção:** Isolar a TUI de seleção de Gems para ser reutilizada tanto pelo comando `gems list` quanto dentro da sessão de chat (`/gems`).

## 2\. Current Situation Analysis

O gerenciamento de Gems já existe, implementado em `internal/api/gems.go` e exposto no comando `internal/commands/gems.go`.

  * **API Layer (`internal/api/gems.go`):** Possui métodos como `FetchGems`, `CreateGem`, `UpdateGem`, e `DeleteGem`, que utilizam o endpoint `batchexecute`. A estrutura `models.GemJar` armazena e permite a recuperação por ID ou nome.
  * **Command Layer (`internal/commands/gems.go`):** O comando `gems list` usa `tui.RunGemsTUI` para abrir uma interface TUI interativa.
  * **TUI Layer (`internal/tui/gems_model.go`):** A implementação atual (`GemsModel`) já carrega e lista os Gems, mas a lógica de transição para o chat e a infraestrutura de retorno do Gem selecionado **existem, mas precisam ser integradas** ao fluxo de inicialização do chat principal.

O arquivo `internal/tui/gems_model.go` já define a estrutura `GemsTUIResult` e o fluxo de iniciar o chat com a tecla `c`, o que indica que a maior parte da fundação está pronta, mas o comando chamador precisa ser adaptado para aceitar o resultado e iniciar a sessão de chat.

## 3\. Proposed Solution / Refactoring Strategy

A estratégia se concentra em refatorar o fluxo de controle no pacote `internal/commands` e garantir que a lógica de inicialização de sessão utilize o Gem ID retornado pelo TUI.

### 3.1. High-Level Design / Architectural Overview

O fluxo será:

1.  O comando `gems list` (ou `chat /gems`) chama o `tui.RunGemsTUI`.
2.  O `GemsModel` gerencia a seleção e retorna `GemsTUIResult` contendo o `GemID`.
3.  O `commands/gems.go` (ou `commands/chat.go` para `/gems`) recebe o resultado.
4.  Se um `GemID` for retornado, o fluxo de inicialização de chat é invocado com esse ID.

<!-- end list -->

```mermaid
graph TD
    subgraph "CLI/Commands"
        A[geminiweb gems list] --> B{tui.RunGemsTUI}
        C[Chat TUI /gems] --> B
    end

    subgraph "TUI"
        B --> D[GemsModel]
        D -- Seleção OK (GemID) --> E{Retorno: GemsTUIResult}
    end

    subgraph "Chat Initialization"
        E --> F{Verificar GemID}
        F -- GemID Válido --> G[api.NewClient]
        G --> H[api.StartChatWithOptions(WithGemID)]
        H --> I[tui.RunChatWithSession]
    end
```

### 3.2. Key Components / Modules

| Componente | Localização | Responsabilidades da Mudança |
| :--- | :--- | :--- |
| **`runGemsList`** | `internal/commands/gems.go` | Receber `GemsTUIResult` e iniciar a sessão de chat se `GemID` não for vazio. |
| **`RunGemsTUI`** | `internal/tui/gems_model.go` | (Já implementado) Retornar `GemsTUIResult` com ID e nome do Gem para iniciar o chat. |
| **`Model.Update`** | `internal/tui/model.go` | Implementar a lógica para lidar com o modo de seleção de Gems (`m.selectingGem`), incluindo filtragem e navegação, e aplicar o Gem ID à sessão de chat. |
| **`loadGemsForChat`** | `internal/tui/model.go` | (Já implementado) Adicionar um comando para carregar os Gems quando `/gems` for digitado na sessão de chat principal. |
| **`createChatSession`** | `internal/commands/session.go` | (Auxiliar) Garantir que a criação de sessão propague o `gemID` para `api.ChatSession`. |

### 3.3. Detailed Action Plan / Phases

#### Phase 1: Integrazione del comando `gems list` con Chat (High Priority)

| Task | Rationale/Goal | Effort | Deliverable/Criteria for Completion |
| :--- | :--- | :--- | :--- |
| 1.1: **Refatorar `runGemsList`** | Usar o resultado `GemsTUIResult` para iniciar o chat. | M | `runGemsList` chama `tui.RunGemsTUI` e, se um Gem for selecionado, passa o controle para o fluxo de inicialização de chat. |
| 1.2: **Unificar Criação de Sessão** | Criar função auxiliar em `internal/commands` para centralizar a lógica de `api.NewClient` e `client.Init()`. | S | Nova função (e.g., `initClientAndSession(gemID, model)`) para evitar duplicação de código. |
| 1.3: **Verificar Dependências do `chat`** | Garantir que `internal/commands/session.go:createChatSession` e `internal/commands/chat.go:runChat` suportem o GemID retornado e o propaguem corretamente. | S | Teste de integração: `geminiweb gems list` -\> `c` -\> Nova sessão iniciada com o Gem. |

#### Phase 2: Integração do comando `/gems` no TUI de Chat (Medium Priority)

*O TUI já possui os campos `selectingGem`, `gemsList`, `gemsCursor`, etc. no `internal/tui/model.go`.*

| Task | Rationale/Goal | Effort | Deliverable/Criteria for Completion |
| :--- | :--- | :--- | :--- |
| 2.1: **Adaptar `handleKeyMsg` (`internal/tui/model.go`)** | Adicionar a lógica para o comando `/gems` e o atalho `Ctrl+G` para transição para o modo `selectingGem`. | S | O chat TUI entra no modo de seleção de Gem. |
| 2.2: **Adaptar `updateGemSelection` (`internal/tui/model.go`)** | Implementar a lógica de navegação/seleção (`up/down`, `enter`, filtro por digitação) no modo de seleção de Gem. | M | Seleção de Gem atualiza `m.session.SetGem(gemID)` e `m.activeGemName`. |
| 2.3: **Refatorar `renderGemSelector` (`internal/tui/model.go`)** | Garantir que o overlay de seleção renderize corretamente a lista de Gems e o filtro. Reutilizar estilos do `config_model.go`. | M | Overlay de seleção de Gem funcional e responsivo. |
| 2.4: **Atualizar Header do Chat** | Exibir o nome do Gem ativo (`m.activeGemName`) no cabeçalho do chat. | S | `Model.View()` exibe `📦 <Gem Name>` no cabeçalho quando um Gem está ativo. |

#### Phase 3: Melhorias de UX e Busca (Low Priority)

| Task | Rationale/Goal | Effort | Deliverable/Criteria for Completion |
| :--- | :--- | :--- | :--- |
| 3.1: **Atualizar `GemsModel` com busca ativa** | Permitir busca em tempo real na lista de Gems enquanto o usuário digita. | S | Filtragem de Gems no `GemsModel` é imediata. |
| 3.2: **Refinar `GemsModel.View`** | Melhorar a formatação da descrição na lista para evitar quebras de layout (tradução `truncateTitle`). | S | Listagem de Gems visualmente agradável e funcional em diferentes tamanhos de terminal. |

### 3.4. Data Model Changes

Não são necessárias alterações no modelo de dados persistente. A lógica se baseia nos modelos existentes:

  * `models.Gem` (ID, Name, Prompt, Description).
  * `models.GemJar` (Cache de Gems no cliente).
  * `internal/api/session.go:ChatSession` (campo `gemID` já existe para contexto).
  * `internal/tui/gems_model.go:GemsTUIResult` (já existe para retorno).

### 3.5. API Design / Interface Changes

Não são necessárias alterações nas interfaces de API existentes (`GeminiClientInterface` ou `ChatSessionInterface`), pois o campo `GemID` e os métodos `SetGem` já existem.

## 4\. Key Considerations & Risk Mitigation

### 4.1. Technical Risks & Challenges

| Risco | Descrição | Mitigação |
| :--- | :--- | :--- |
| **Reutilização de TUI** | Tentar reutilizar o `GemsModel` diretamente no `ChatModel` pode introduzir complexidade no ciclo de vida do Bubble Tea. | **Mitigação:** Em vez de incorporar o `GemsModel` no `ChatModel`, a nova estratégia é que o `ChatModel` **simule** a lógica de seleção de Gems (tarefa 2.2) em seu próprio método (`updateGemSelection`), evitando a complexidade de aninhar múltiplos *programas* ou *models* que não são totalmente independentes. |
| **State Consistency** | Garantir que `m.session.SetGem()` em `internal/tui/model.go` se propague corretamente para as chamadas `GenerateContent`. | **Mitigação:** Verificação em `internal/tui/model.go:sendMessageWithAttachments` que o `GemID` seja lido de `m.session.GetGemID()` e passado para `api.GenerateOptions`. (O código atual de `api/session.go` já faz isso). |
| **Tradução de Estado** | O `gems list` termina o programa TUI e inicia um novo. | **Mitigação:** O comando `gems list` deve encapsular a lógica de `client.Init()` e `client.Close()` para o novo chat, utilizando o GemID retornado como argumento de inicialização. |

### 4.2. Dependencies

  * **Phase 1** é independente.
  * **Phase 2** depende da finalização da Fase 1 para a lógica de inicialização de chat.
  * O trabalho é quase totalmente interno aos pacotes `internal/commands` e `internal/tui`, sem dependências externas.

### 4.3. Non-Functional Requirements (NFRs) Addressed

| NFR | Como o Plano Contribui |
| :--- | :--- |
| **Usabilidade (UX)** | A lista interativa (TUI) com busca e seleção de Gem para iniciar o chat é muito mais ergonômica do que copiar/colar IDs ou digitar o nome/ID na CLI. |
| **Eficiência** | O atalho `c` permite iniciar a sessão de chat em dois toques a partir da lista de Gems. O `/gems` dentro do chat permite a troca de persona sem sair da sessão. |
| **Descoberta** | A interface TUI expõe a lista completa de Gems, descrições e o tipo (sistema/customizado), facilitando a descoberta de novas personas. |

## 5\. Success Metrics / Validation Criteria

1.  O comando `geminiweb gems list` abre o TUI, permite a navegação, e pressionar `c` em um Gem abre uma sessão de chat com o Gem correto ativado.
2.  Dentro de uma sessão de chat, digitar `/gems` abre o seletor de Gem em overlay, e a seleção de um Gem atualiza o cabeçalho do chat e o contexto da sessão (`session.GetGemID()` retorna o ID correto).
3.  A filtragem (digitação) no seletor de Gem (`GemsModel`) é em tempo real e não causa crashes ou lentidão perceptível.

## 6\. Assumptions Made

  * O Gem ID, uma vez definido na sessão de chat (`session.SetGem`), é incluído corretamente no payload JSON para o endpoint `/StreamGenerate`. (Verificado: `internal/api/generate.go:buildPayloadWithGem` já suporta `gemID`).
  * O `GeminiClient` será inicializado e fechado corretamente em torno da nova sessão de chat iniciada a partir do `gems list`.

## 7\. Open Questions / Areas for Further Investigation

| Questão | Decisão |
| :--- | :--- |
| O filtro no seletor de Gems deve ser persistente? | Não. O filtro deve ser *ad hoc* para a sessão de seleção. |
| A TUI de Gems deve permitir criação/edição? | Não. Manter a modificação (create/update/delete) restrita aos comandos CLI explícitos para simplicidade e segurança. |
| Deve haver um Gem "None" (sem persona)? | Sim. O Gem de sistema "default" ou "none" deve ser incluído na lista se o `FetchGems` retornar todos os tipos. |