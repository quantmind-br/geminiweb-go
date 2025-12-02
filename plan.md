# Refactoring/Design Plan: Integração de Gems com Modo Chat Interativo (TUI)

## 1\. Executive Summary & Goals

O objetivo primário deste plano é integrar a funcionalidade de "Gems" (personas customizadas do lado do servidor) ao modo de **chat interativo (TUI)** da aplicação `geminiweb`, permitindo que o usuário inicie uma sessão de chat com uma Gem específica selecionada a partir do comando `geminiweb gems list`.

-----

### Key Goals & Outcomes

1.  **Habilitar Chat com Gem na TUI:** Modificar o TUI de Gems (`internal/tui/gems_model.go`) para permitir que o usuário selecione uma Gem e inicie um chat com ela.
2.  **Passagem de GemID para Sessão:** Propagar o ID da Gem selecionada do TUI de Gems para a função de inicialização do TUI de Chat, garantindo que o `ChatSession` seja criado com o `GemID` correto.
3.  **Atualização do Comando `gems list`:** Modificar o comando `geminiweb gems list` para que, em vez de apenas listar, ele inicie o TUI de Gems, onde a seleção do chat é feita.

## 2\. Current Situation Analysis

A aplicação `geminiweb-go` possui uma arquitetura bem definida baseada em Cobra para comandos, uma camada `api` com a lógica de cliente/sessão, e uma camada `tui` para interfaces interativas.

  * **Gems Management (`internal/api/gems.go`, `internal/commands/gems.go`, `internal/tui/gems_model.go`):** Existe um modelo TUI dedicado (`GemsModel`) para listar Gems e a lógica de API (`FetchGems`, `GetGem`) está implementada. O `GemsModel` atual permite apenas visualização e cópia de ID.
  * **Chat Session (`internal/api/session.go`, `internal/api/client.go`):** O `ChatSession` já suporta o campo `gemID` e a função `StartChatWithOptions` pode receber `WithGemID` ou `WithGem`. O `GenerateContent` (`internal/api/generate.go`) já inclui o `gemID` no payload, o que é crucial.
  * **Chat TUI (`internal/tui/model.go`, `internal/commands/chat.go`):** O TUI de Chat é iniciado pelo comando `geminiweb chat` (`internal/commands/chat.go`) usando `tui.RunChat(client, modelName)`. Não há um mecanismo para injetar um `GemID` nesse fluxo.

### Key Pain Points / Areas for Improvement

1.  **Desacoplamento de TUI de Chat:** O `tui.RunChat` e `commands.runChat` precisam ser modificados ou um novo ponto de entrada precisa ser criado para aceitar um `GemID` opcional e passá-lo para a sessão.
2.  **Lógica de Início de Chat:** A lógica para iniciar o chat, que atualmente reside em `commands/chat.go` e `commands/gems.go` (para o cliente), precisa ser reutilizada e adaptada para a integração Gem.

-----

## 3\. Proposed Solution / Refactoring Strategy

A estratégia será utilizar a infraestrutura existente de TUI e API, introduzindo um novo "comando" (mensagem) no `GemsModel` para iniciar o chat e um novo parâmetro em `tui.RunChat` para injetar o `GemID`.

### 3.1. High-Level Design / Architectural Overview

  * **Camada de Comando (`internal/commands`):** O `runGemsList` será mantido para iniciar o `GemsTUI`. O `GemsTUI` passará a ser o ponto de partida do chat com Gem.
  * **Camada TUI (`internal/tui`):**
      * `GemsModel`: Adicionar uma nova ação (e.g., tecla `s` ou `chat`) para iniciar o chat. Definir uma nova `tea.Msg` para a transição.
      * `Model`: O `RunChat` precisa ser modificado para aceitar um `GemID` inicial, ou uma nova função de inicialização de chat TUI será criada.
  * **Camada API (`internal/api`):** O `StartChatWithOptions` será o método chave para criar a sessão com o `GemID` injetado.

### 3.2. Key Components / Modules

| Componente | Modificação Principal | Responsabilidade |
| :--- | :--- | :--- |
| **`internal/tui/gems_model.go`** | Adicionar `tea.Msg` `startChatMsg`, lógica de seleção de chat (tecla). | Interagir com o usuário para selecionar a Gem e emitir o comando de início de chat. |
| **`internal/tui/model.go`** | Adicionar `GemID` opcional ao `RunChat` ou criar `RunGemChat`. | Gerenciar a nova sessão de chat, inicializando-a com o `GemID` da Gem selecionada. |
| **`internal/commands/gems.go`** | Implementar a lógica de `RunE` para tratar a `tea.Msg` de `GemsModel` (se a arquitetura Bubble Tea for mantida no ponto de entrada). | Iniciar o TUI de Gems. Se o TUI retornar um `GemID`, iniciar o TUI de Chat com esse ID. |
| **`internal/api/client.go`** | Nenhuma alteração essencial é necessária, mas o uso de `StartChatWithOptions` será enfatizado. | Prover o método `StartChatWithOptions` que aceita o `WithGemID`. |

### 3.3. Detailed Action Plan / Phases

#### Phase 1: Modificação do TUI de Gems e Definição da Transição

**Objective(s):** Permitir a seleção de uma Gem e a emissão de uma mensagem para iniciar o chat com o ID da Gem.
**Priority:** High

| Task | Rationale/Goal | Estimated Effort (Optional) | Deliverable/Criteria for Completion |
| :--- | :--- | :--- | :--- |
| **1.1: Criar Mensagem de Início de Chat** | Permitir que o `GemsModel` retorne o ID da Gem selecionada para a função chamadora. | S | `startChatMsg` (struct com `gemID string`) definida em `internal/tui/gems_model.go`. |
| **1.2: Implementar Ação de Chat no `GemsModel`** | No `GemsModel.Update`, adicionar lógica para a tecla, e.g., `c` (para chat) ou `s` (select). Se uma Gem estiver selecionada, retornar `startChatMsg` e `tea.Quit`. | S | Lógica de tecla implementada, retornando a nova `startChatMsg` e encerrando o TUI de Gems. |
| **1.3: Atualizar `GemsModel.View`** | Adicionar uma dica de atalho (`[c]hat` ou similar) na barra de status para a nova funcionalidade. | S | Barra de status do `GemsModel` atualizada com o atalho `[c]hat` ou similar na `gemsViewList`. |

#### Phase 2: Refatoração do Início do Chat e Implementação da Conexão

**Objective(s):** Fazer o `commands/gems.go` receber o ID da Gem e iniciar o `ChatTUI` corretamente.
**Priority:** High

| Task | Rationale/Goal | Estimated Effort (Optional) | Deliverable/Criteria for Completion |
| :--- | :--- | :--- | :--- |
| **2.1: Refatorar `tui.RunChat`** | O TUI de chat precisa de um ponto de entrada para um `ChatSession` já configurado com um `GemID`. | M | Modificar a assinatura de `tui.RunChat(client, modelName)` para `tui.RunChat(client, modelName, initialGemID string)` ou criar `tui.RunGemChat`. Optar por `RunChatWithOptions` para simplificar. |
| **2.2: Criar Opção de Chat com Gem na TUI** | Criar um novo tipo no `internal/tui/model.go` ou adaptar o `NewChatModel` para aceitar um `GemID` e usá-lo na inicialização da sessão. | S | `NewChatModel` atualizado para receber `initialGemID string`. |
| **2.3: Atualizar `internal/commands/gems.go` (runGemsList)** | O `runGemsList` agora precisa executar o `GemsTUI` e inspecionar o resultado (se for `startChatMsg`), então iniciar o `ChatTUI`. | M | `runGemsList` passa a ser um *wrapper* que executa `tui.RunGemsTUI` e, se um `startChatMsg` for retornado, inicia o `ChatTUI` (Task 2.4). |
| **2.4: Lógica de Inicialização do `ChatSession` com GemID** | Implementar a lógica de inicialização do `ChatSession` dentro do `RunChat` para usar o `initialGemID` via `client.StartChatWithOptions(api.WithGemID(gemID))`. | S | `tui.RunChat` (ou similar) chama `client.StartChatWithOptions(api.WithGemID(gemID))` se `gemID` não for vazio. |

#### Phase 3: Teste e Finalização

**Objective(s):** Verificar o fluxo de trabalho completo e garantir a estabilidade.
**Priority:** Medium

| Task | Rationale/Goal | Estimated Effort (Optional) | Deliverable/Criteria for Completion |
| :--- | :--- | :--- | :--- |
| **3.1: Teste de Integração (Manual/E2E)** | Garantir que o fluxo `geminiweb gems list` -\> Seleção de Gem -\> Chat com Gem funciona e que o prompt da Gem está sendo enviado ao `GenerateContent`. | L | Um chat é iniciado com a Gem, e a primeira resposta demonstra a ativação do prompt da Gem. |
| **3.2: Refatoração de Código de Inicialização** | Unificar a lógica de criação do cliente e inicialização (`createGemsClient` e `runChat` cliente logic) em uma função reutilizável, se possível, para evitar duplicação. | S | Seções de criação/inicialização de cliente em `commands/chat.go` e `commands/gems.go` simplificadas ou unificadas. |

### 3.4. API Design / Interface Changes

**Interface Modificada:**

  * **`internal/tui/gems_model.go`**:

      * Definir um novo tipo de mensagem para a transição:
        ```go
        type startChatMsg struct {
        	gemID string
        }
        ```

  * **`internal/tui/model.go`**:

      * **Atualização de `RunChat`:** A função TUI que inicia o chat precisa de um ponto de injeção para o Gem ID.
        ```go
        // Exemplo:
        // func RunChat(client api.GeminiClientInterface, modelName string, initialGemID string) error { ... }
        //
        // Ou, para manter a modularidade do TUI:
        func RunChatWithOptions(client api.GeminiClientInterface, opts ...api.ChatOption) error {
            m := NewChatModelWithOptions(client, opts...)
            // ... (rest of TUI setup)
        }
        ```
        *Decisão:* Modificaremos `tui.RunChat` para aceitar um `*api.ChatSession` pré-configurado, transferindo a responsabilidade da criação do `ChatSession` para `commands/gems.go`.

  * **`internal/tui/model.go`**:

      * **Atualização de `NewChatModel`:**
        ```go
        // Modificar NewChatModel para aceitar a sessão, não apenas o client
        func NewChatModel(session api.ChatSessionInterface) Model { ... }
        // ... E atualizar tui.RunChat (e tui.RunGemChat) para usar esta nova assinatura.
        ```

  * **`internal/commands/gems.go`**:

      * **Atualização de `runGemsList`**:
        ```go
        func runGemsList(cmd *cobra.Command, args []string) error {
            client, err := createGemsClient()
            // ...
            
            // Rodar TUI de Gems e obter o GemID (se um chat for iniciado)
            gemID, err := tui.RunGemsTUI(client, gemsIncludeHidden) // Retornar string/erro
            
            if gemID == "" {
                return nil // Saiu do TUI sem iniciar chat
            }
            
            // Iniciar o chat TUI
            modelName := getModel()
            model := models.ModelFromName(modelName)
            
            session := client.StartChatWithOptions(
                api.WithChatModel(model),
                api.WithGemID(gemID),
            )
            
            return tui.RunChat(session) // Novo RunChat aceita a sessão
        }
        ```

  * **`internal/tui/model.go` (final):**

      * **Assinatura de `RunChat` (nova)**:
        ```go
        func RunChat(session api.ChatSessionInterface) error {
            m := NewChatModel(session)
            // ... (restante da inicialização do BubbleTea)
        }
        ```

## 4\. Key Considerations & Risk Mitigation

### 4.1. Technical Risks & Challenges

| Risco | Mitigação |
| :--- | :--- |
| **Acoplamento de TUI:** O TUI de Gems precisa iniciar o TUI de Chat, o que pode quebrar a separação de preocupações. | O `RunGemsTUI` retornará o `GemID` selecionado (se houver), permitindo que a camada de comando (`commands/gems.go`) orquestre a transição, mantendo o TUI de Gems e o TUI de Chat desacoplados. |
| **Inicialização Dupla de Cliente:** O `createGemsClient` em `gems.go` já inicializa o cliente. Se for usado em `runGemsList` seguido por um `ChatTUI`, o cliente pode ser fechado prematuramente. | Garantir que o `createGemsClient` não feche o cliente se um chat TUI for iniciado. A responsabilidade de fechar o cliente deve ser transferida para a função de comando de nível superior (`runGemsList`). |
| **Modelos Inconsistentes:** Se a Gem exigir um modelo específico, mas o usuário tiver configurado outro. | Usar `api.WithChatModel` no `StartChatWithOptions` para priorizar o modelo padrão do usuário, a menos que a lógica de Gem no futuro force um modelo. O chat TUI pode ser iniciado com o modelo preferencial (lido via `getModel()`). |

### 4.2. Dependencies

  * **Interna:** Dependência do `commands/gems.go` na nova interface de `tui.RunGemsTUI` e na nova função de `tui.RunChat`.
  * **API:** Dependência do `api.GeminiClient.StartChatWithOptions` e `api.WithGemID`. (Já estão implementados, apenas o uso muda).

### 4.3. Non-Functional Requirements (NFRs) Addressed

| NFR | Contribuição do Plano |
| :--- | :--- |
| **Usabilidade:** | Adiciona um fluxo de trabalho intuitivo no TUI para iniciar um chat com Gem, em vez de exigir que o usuário copie e cole IDs de Gem. |
| **Manutenibilidade:** | Centraliza a lógica de inicialização de sessão de chat com opções (incluindo Gem ID) no `api.Client`, e a orquestração na camada `commands`, promovendo a separação de preocupações. |
| **Extensibilidade:** | O novo `tui.RunChat(session)` permite que o `ChatTUI` seja iniciado por qualquer parte do código com uma sessão pré-configurada (histórico, gem, modelo), facilitando futuras extensões. |

-----

## 5\. Success Metrics / Validation Criteria

  * O comando `geminiweb gems list` abre o TUI de Gems.
  * Dentro do TUI de Gems, uma nova opção de atalho (e.g., `c`) é visível para a Gem selecionada.
  * Ao pressionar o atalho, o TUI de Gems fecha e o TUI de Chat é iniciado automaticamente.
  * O campo `gemID` da sessão de chat resultante (`internal/api/session.go`) é preenchido com o ID da Gem selecionada.
  * A primeira resposta no chat reflete as instruções da Gem (se ela tiver um System Prompt).

-----

## 6\. Assumptions Made

  * O comando `geminiweb chat` será refatorado para usar a nova assinatura `tui.RunChat(session)` (onde a sessão é criada sem um GemID).
  * A inicialização do `GeminiClient` em `commands/gems.go` (`createGemsClient`) é suficiente para autenticação antes de iniciar o TUI de Gems.
  * O retorno de `tui.RunGemsTUI` pode ser uma struct/string contendo o GemID, se a arquitetura Bubble Tea for usada fora do `tea.Run()` para orquestração. **Assumindo que `tea.Run()` pode retornar a `tea.Model` final, que pode ser inspecionada para obter o `GemID` via uma interface.**

-----

## 7\. Open Questions / Areas for Further Investigation

  * **Implementação de Retorno do TUI:** Como o `tui.RunGemsTUI` (que usa `tea.NewProgram().Run()`) pode retornar o `GemID` selecionado?
      * *Plano de Resposta:* Modificar `RunGemsTUI` para retornar uma string (`GemID`) e um erro, e inspecionar o `tea.Model` resultante no `runGemsList`.
  * **Transição de Sessão:** O `ChatSession` pode ser criado com o `client` e o `GemID` antes de iniciar o TUI. O TUI de Chat precisa ser atualizado para aceitar o `ChatSession` completo, em vez de apenas o `client` e o `modelName`. **(Abordado no Task 2.4 - nova assinatura `tui.RunChat(session)`).**