# Plano de Refatoração/Design: Modo Interativo para Gerenciamento de Gems (Personas)

## 1. Resumo Executivo e Objetivos

O objetivo primário (já implementado nesta base) foi estender a funcionalidade de listagem de **Gems** (`geminiweb gems list`) para um modo interativo completo, permitindo que o usuário visualize, filtre e **inicie um chat com o Gem selecionado** diretamente da TUI.

### Objetivos principais

1. **Habilitar chat rápido:** iniciar uma sessão de chat com o Gem selecionado (tecla `c`) a partir da lista TUI.
2. **Melhorar a descoberta (UX):** lista interativa com busca em tempo real e visualização de detalhes.
3. **Encapsular a lógica de seleção:** seletor reutilizável entre `gems list` e o comando `/gems` dentro do chat.

> **Status atual:** objetivos 1–3 concluídos.

## 2. Análise da Situação Atual

O gerenciamento de Gems já existe em `internal/api/gems.go` e é exposto via `internal/commands/gems.go`. Hoje:

  * **Camada API (`internal/api/gems.go`):** `FetchGems`, `CreateGem`, `UpdateGem`, `DeleteGem`, usando `BatchExecute`; cache via `models.GemJar`.
  * **Camada CLI (`internal/commands/gems.go`):** `runGemsList` chama `tui.RunGemsTUI` e interpreta `GemsTUIResult` para iniciar chat quando o usuário pressiona `c`.
  * **TUI de listagem (`internal/tui/gems_model.go`):** lista, filtra (`/`), mostra detalhes e sinaliza início de chat via `GemsTUIResult`.
  * **TUI do chat (`internal/tui/model.go`):** seletor inline acionado por `/gems` ou `Ctrl+G`, com filtro por digitação, navegação e aplicação do Gem ativo com `session.SetGem()` e `activeGemName`.

Pontos ainda discutíveis:

  * Ao iniciar chat a partir de `gems list`, o cliente vem de `createGemsClient()` com `WithAutoRefresh(false)` e o chat não integra histórico; validar se isso é aceitável ou se deve seguir o fluxo padrão do comando `chat`.
  * Não há opção explícita de “sem Gem” no seletor do chat; dependemos de um Gem de sistema padrão para limpar a persona.

## 3. Solução Implementada / Estratégia

A solução final mantém dois fluxos:

### 3.1. Visão Geral

A) **`geminiweb gems list`**

1. `runGemsList` chama `tui.RunGemsTUI`.
2. `GemsModel` retorna `GemsTUIResult` com `GemID` quando `c` é pressionado.
3. `runGemsList` cria `ChatSession` com `gemID` e inicia `tui.RunChatWithSession`.

B) **Dentro do chat (`/gems` ou `Ctrl+G`)**

1. `Model.Update` entra em `selectingGem` e dispara `loadGemsForChat`.
2. `updateGemSelection` navega/filtra e, ao confirmar, aplica `session.SetGem(gemID)` e atualiza `activeGemName`.

```mermaid
graph TD
    subgraph "gems list"
        A[geminiweb gems list] --> B{tui.RunGemsTUI}
        B --> C[GemsModel]
        C --> D{GemsTUIResult}
        D --> E[runGemsList cria sessão]
        E --> F[tui.RunChatWithSession]
    end

    subgraph "chat"
        G[/gems ou Ctrl+G] --> H[Model.selectingGem]
        H --> I[loadGemsForChat]
        I --> J[updateGemSelection]
        J --> K[session.SetGem + activeGemName]
    end
```

### 3.2. Componentes‑chave

| Componente | Localização | Responsabilidades / Status |
| :--- | :--- | :--- |
| **`runGemsList`** | `internal/commands/gems.go` | Integra `GemsTUIResult` e inicia chat com `GemID` (concluído). |
| **`RunGemsTUI` / `GemsModel`** | `internal/tui/gems_model.go` | Lista, filtra e retorna Gem selecionado para chat (concluído). |
| **`Model.Update` / `updateGemSelection` / `renderGemSelector`** | `internal/tui/model.go` | Seletor inline no chat via `/gems`/`Ctrl+G` (concluído). |
| **`loadGemsForChat`** | `internal/tui/model.go` | Carrega e ordena gems para o seletor do chat (concluído). |
| **`createChatSession`** | `internal/commands/session.go` | Propaga `gemID` para `api.ChatSession` (concluído). |

### 3.3. Plano de ações / fases

As fases 1–3 estão implementadas nesta base. Mantemos os itens originais com status e adicionamos um backlog opcional.

#### Fase 1: Integração do comando `gems list` com chat (concluída)

| Task | Objetivo | Esforço | Critério de conclusão | Status |
| :--- | :--- | :--- | :--- | :--- |
| 1.1: **Refatorar `runGemsList`** | Usar o resultado `GemsTUIResult` para iniciar o chat. | M | `runGemsList` chama `tui.RunGemsTUI` e inicia `tui.RunChatWithSession` quando há seleção. | Concluído |
| 1.2: **Centralizar criação de cliente/sessão** | Evitar duplicação usando `createGemsClient` e `createChatSession`. | S | Fluxo de listagem e chat compartilham helpers de inicialização. | Concluído |
| 1.3: **Validar fluxo end‑to‑end** | Garantir que o `GemID` seja propagado ao payload de geração. | S | Teste manual: `gems list` → `c` abre chat com Gem correto. | Concluído |

#### Fase 2: Integração do comando `/gems` no TUI de chat (concluída)

| Task | Objetivo | Esforço | Critério de conclusão | Status |
| :--- | :--- | :--- | :--- | :--- |
| 2.1: **Ativar modo `selectingGem`** | Suportar `/gems` e atalho `Ctrl+G`. | S | Chat entra no seletor de Gem em overlay. | Concluído |
| 2.2: **Navegação e filtro** | Implementar `up/down`, `enter` e filtro por digitação. | M | Seleção atualiza `session.SetGem(gemID)` e `activeGemName`. | Concluído |
| 2.3: **Renderização do overlay** | Garantir lista responsiva e reutilização de estilos. | M | Overlay funcional em diferentes tamanhos de terminal. | Concluído |
| 2.4: **Header do chat** | Mostrar Gem ativo no cabeçalho. | S | `Model.View()` exibe o Gem quando ativo. | Concluído |

#### Fase 3: Melhorias de UX e busca (concluída)

| Task | Objetivo | Esforço | Critério de conclusão | Status |
| :--- | :--- | :--- | :--- | :--- |
| 3.1: **Busca ativa em `GemsModel`** | Filtrar em tempo real ao digitar. | S | Filtragem imediata na lista TUI. | Concluído |
| 3.2: **Refino de visualização** | Evitar quebras de layout/truncar descrições. | S | Lista agradável em terminais pequenos. | Concluído |

#### Fase 4: Melhorias futuras / backlog (opcional)

| Task | Objetivo | Esforço | Critério de conclusão | Status |
| :--- | :--- | :--- | :--- | :--- |
| 4.1: **Opção “Sem Gem” no seletor do chat** | Permitir limpar a persona ativa. | S | Seletor inclui item `<none>` que chama `session.SetGem("")` e limpa `activeGemName`. | Backlog |
| 4.2: **Auto‑refresh no chat iniciado via `gems list`** | Evitar expiração de cookies em chats longos. | S | Chat iniciado por listagem usa `autoRefresh=true` (novo cliente ou reconfiguração). | Backlog |
| 4.3: **Incluir gems ocultos no seletor do chat (config/flag)** | Paridade com `gems list --hidden`. | S | `loadGemsForChat` aceita `includeHidden=true` quando habilitado. | Backlog |
| 4.4: **Testes de filtragem/transição de Gem** | Reduzir regressões. | M | Testes em `internal/tui` cobrindo filtro e retorno do TUI. | Backlog |

### 3.4. Mudanças no Modelo de Dados

Não são necessárias alterações no modelo de dados persistente. A lógica usa modelos existentes:

  * `models.Gem` (ID, Name, Prompt, Description).
  * `models.GemJar` (cache de Gems no cliente).
  * `internal/api/session.go:ChatSession` (campo `gemID` já existe).
  * `internal/tui/gems_model.go:GemsTUIResult` (retorno do TUI).

### 3.5. Mudanças de API / Interfaces

Não são necessárias alterações nas interfaces públicas existentes (`GeminiClientInterface` ou `ChatSessionInterface`), pois `GemID` e `SetGem` já são suportados.

## 4. Considerações‑chave e Mitigação de Riscos

### 4.1. Riscos Técnicos

| Risco | Descrição | Mitigação |
| :--- | :--- | :--- |
| **Reutilização de TUI** | Reutilizar `GemsModel` diretamente no `ChatModel` pode complicar o ciclo de vida do Bubble Tea. | Manter seletor inline separado no chat (`selectingGem`/`updateGemSelection`). |
| **Consistência de estado** | Garantir que `session.SetGem()` se propague para `GenerateContent`. | `api/session.go` já lê `GetGemID()` e passa para `GenerateOptions`. |
| **Troca de programa TUI** | `gems list` encerra um programa Bubble Tea e inicia outro. | Reuso do mesmo cliente e criação explícita de sessão com `gemID`. Validar auto‑refresh no backlog (4.2). |

### 4.2. Dependências

  * Fases 1–3 concluídas e independentes.
  * Itens da Fase 4 são opcionais e independentes entre si.

### 4.3. Requisitos Não‑Funcionais (NFRs)

| NFR | Como o plano contribui |
| :--- | :--- |
| **Usabilidade (UX)** | Lista interativa com busca/seleção reduz atrito para descobrir e ativar Gems. |
| **Eficiência** | `c` inicia chat direto da listagem; `/gems` troca persona sem sair da sessão. |
| **Descoberta** | Descrições e tipo (system/custom) expostos na TUI facilitam explorar personas. |

## 5. Métricas de Sucesso / Validação

1. `geminiweb gems list` abre o TUI, permite navegar e `c` inicia chat com o Gem correto.
2. Em uma sessão de chat, `/gems` (ou `Ctrl+G`) abre seletor, e a seleção atualiza contexto e cabeçalho.
3. A filtragem no seletor é em tempo real e sem instabilidades.

## 6. Premissas

  * O `gemID` definido em sessão é incluído no payload de `/StreamGenerate` (`internal/api/generate.go:buildPayloadWithGem`).
  * O cliente é inicializado e fechado corretamente ao transicionar de listagem para chat.

## 7. Decisões e Pendências

| Questão | Decisão / Status |
| :--- | :--- |
| O filtro no seletor de Gems deve ser persistente? | Não; o filtro é ad hoc e reseta ao sair do seletor. |
| A TUI de Gems deve permitir criação/edição? | Não; create/update/delete ficam nos comandos CLI explícitos. |
| Deve haver opção "None" (sem persona)? | Ainda não existe na UI; proposta no backlog (4.1). |
