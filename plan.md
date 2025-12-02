# Refactoring/Design Plan: Sistema de Tratamento de Erros Robusto (Go)

## 1. Executive Summary & Goals
O objetivo primário deste plano é refatorar o sistema de *error handling* existente em `internal/errors` e nos pacotes `internal/api` e `internal/commands` do projeto `geminiweb-go`. O foco é introduzir **tratamento de erros estruturado**, **código de erro canônico** (onde aplicável) e **melhor rastreabilidade (*observability*)** para facilitar a identificação, o diagnóstico e a manipulação programática de problemas nas requisições da API.

### Key Goals
1.  **Uniformidade e Rastreabilidade**: Garantir que todos os erros da camada `api` sejam retornados como tipos de erro específicos, contendo metadados essenciais (e.g., código HTTP, endpoint, código de erro interno, mensagem amigável), permitindo a rastreabilidade do ponto de falha.
2.  **Separação de Preocupações**: Isolar a lógica de apresentação/mensagens de erro do código do erro (e.g., status code) e da lógica de manipulação (*retry*, *fallback*).
3.  **Extensibilidade**: Projetar o sistema de erros para suportar a adição fácil de novos tipos de erro e metadados.

---

## 2. Current Situation Analysis
O projeto já possui um pacote `internal/errors` com tipos de erro customizados (`AuthError`, `APIError`, `UsageLimitError`, etc.), que é um bom ponto de partida.

### Key Pain Points & Limitations
* **APIError Genérico**: `APIError` em `internal/errors/errors.go` é usado para erros de status HTTP que não são 401. A mensagem de erro é genérica (`"generate content failed"` em `internal/api/generate.go`) e não inclui o corpo da resposta HTTP, que muitas vezes contém detalhes cruciais de erro do Gemini.
* **Sem Wrapper para Erros de Rede**: Erros de rede (timeouts, falhas de conexão) são retornados como `fmt.Errorf("request failed: %w", err)` em `internal/api/generate.go`, misturando-se a erros de lógica.
* **Tratamento de Erros Espalhado**: A lógica de mapeamento de código de erro (e.g., `handleErrorCode` em `internal/api/generate.go`) e a lógica de retry/fallback (`isAuthError` e o bloco de retry em `GenerateContent` no mesmo arquivo) estão acopladas à função `GenerateContent`.
* **Mapeamento Implícito de Erro**: O mapeamento de códigos de erro do Gemini (e.g., `ErrUsageLimitExceeded`) para os tipos de erro customizados (`NewUsageLimitError`) é feito em `internal/api/generate.go`, mas o próprio `APIError` não inclui o código de erro Gemini para fácil identificação.

---

## 3. Proposed Solution / Refactoring Strategy

O plano consiste em criar um **`GeminiError` estruturado** na camada `internal/errors` que encapsule todos os tipos de erro de rede, autenticação, parsing e API, adicionando um campo de código de erro canônico e metadados.

### 3.1. High-Level Design / Architectural Overview

A refatoração seguirá este fluxo de erro centralizado.


1.  **Definição de Erro Estruturado** (`internal/errors`): Introduzir `GeminiError` para ser o erro base, implementando a interface `error` e contendo metadados (código HTTP, código canônico interno, endpoint, etc.).
2.  **Captura e Criação de Erro** (`internal/api`): Todas as chamadas de API (`doGenerateContent`, `GetAccessToken`, `RotateCookies`, etc.) serão modificadas para capturar erros de rede, status HTTP, e erros de parsing JSON, e envolvê-los em um `GeminiError` ou um dos seus subtipos.
3.  **Lógica de Ação de Erro** (`internal/api/client.go` e `internal/api/generate.go`): O código de retry/fallback (e.g., `isAuthError` em `GenerateContent`) será refatorado para usar `errors.Is` ou `errors.As` contra os tipos canônicos de erro.

### 3.2. Key Components / Modules

| Componente | Modificação Proposta | Responsabilidade |
| :--- | :--- | :--- |
| `internal/errors/errors.go` | Refatorar `APIError` para ser `struct` mais detalhada. Introduzir `ErrorCodeInternal` (enum). | Definir a estrutura de erro canônico. |
| `internal/api/generate.go` | Centralizar a criação de `APIError` após falha de requisição. Refatorar `isAuthError` para usar `errors.Is`. | Criar erros estruturados a partir de respostas HTTP. |
| `internal/api/client.go` | Envolver erros de inicialização e `RefreshFromBrowser` em `GeminiError`s ou `AuthError`s claros. | Garantir que a camada superior manipule erros de autenticação consistentes. |
| `internal/commands/*` & `internal/tui/*` | Atualizar tratamento de erros para extrair informações úteis de `GeminiError` ou tipos específicos para exibição. | Apresentar mensagens de erro informativas para o usuário. |

### 3.3. Detailed Action Plan / Phases

#### Phase 1: Structuring and Standardizing Errors (High Priority)

| Task | Rationale/Goal | Estimated Effort | Deliverable/Criteria for Completion |
| :--- | :--- | :--- | :--- |
| 1.1: Refactor `errors.go` structs | Criar um erro base único (`GeminiError`) que encapsula todos os outros, adicionando campos para `URL`, `Operation` e `InternalCode` (código Gemini). | M | Novo `GeminiError` base. `APIError`, `AuthError`, etc., implementam o método `Unwrap()` ou são renomeados para usar `GeminiError`. |
| 1.2: Centralize Gemini Error Codes | Mover o mapeamento de códigos de erro Gemini (`handleErrorCode` e `ErrorCode` enum) de `internal/api/generate.go` para `internal/errors/errors.go`. | S | `internal/errors/errors.go` contém a lógica de mapeamento e um novo `InternalErrorCode` enum. |
| 1.3: Update Error Creation in `generate.go` | Modificar `doGenerateContent` para envolver falhas de requisição e status codes não-200 em `GeminiError` ou `APIError` (que agora usa `GeminiError` como base/wrapper). | M | `doGenerateContent` retorna consistentemente `GeminiError` ou `APIError` ao falhar. |
| 1.4: Refactor `isAuthError` | Modificar `isAuthError` para usar `errors.Is(err, &errors.AuthError{})` ou `errors.Is(err, errors.ErrAuthFailed)` em vez de `apiErr.StatusCode == 401` ou *type assertion* direta, aproveitando o *error wrapping* (Go 1.13+). | S | Lógica de retry em `GenerateContent` é mais robusta e desacoplada do status code HTTP. |

#### Phase 2: Enhancing API Clients (Medium Priority)

| Task | Rationale/Goal | Estimated Effort | Deliverable/Criteria for Completion |
| :--- | :--- | :--- | :--- |
| 2.1: Enhance `APIError` creation with Body | Em `internal/api/generate.go`, ler o corpo da resposta em caso de status code diferente de 200, e incluir esse corpo (ou uma versão truncada/parseada) no `APIError` para diagnóstico. | M | `APIError` (ou `GeminiError`) inclui o corpo da resposta para logs detalhados. |
| 2.2: Update `token.go` and `rotate.go` | Modificar `GetAccessToken` e `RotateCookies` para retornar `errors.NewAuthError(...)` ou `errors.NewAPIError(...)` com metadados claros (URL, status code). | S | Erros de autenticação/rotação são uniformes com a nova estrutura. |
| 2.3: Handle Network/Timeout Errors | Em `generate.go` e outros clientes, envolver erros de rede de `c.httpClient.Do(req)` em um novo tipo de erro canônico (e.g., `NetworkError` que se aninha no `GeminiError`). | S | Erros de rede são distinguíveis de erros da API. |

#### Phase 3: Presentation and Integration (Low Priority)

| Task | Rationale/Goal | Estimated Effort | Deliverable/Criteria for Completion |
| :--- | :--- | :--- | :--- |
| 3.1: Update `commands/query.go` Error Display | Atualizar `runQuery` para extrair e exibir informações detalhadas do `GeminiError` (e.g., código de erro, endpoint, mensagem detalhada) quando for um erro estruturado. | S | Mensagem de erro mais informativa na CLI. |
| 3.2: Update `tui/model.go` Error Display | Atualizar a manipulação de `errMsg` para fornecer detalhes úteis no log de erro da TUI. | S | TUI exibe detalhes de erro de API se disponíveis. |
| 3.3: Logging Refinement (Optional) | Em `GenerateContent` e `RefreshFromBrowser`, usar o campo `Verbose` da configuração para logar o `GeminiError` completo (incluindo corpo da resposta) para fins de depuração. | S | Logs de depuração enriquecidos em modo *verbose*. |

### 3.4. Data Model Changes

Não se aplica, pois as mudanças são apenas na estrutura de erro e não em dados persistidos (exceto por erros que podem ser logados se a tarefa fosse sobre logging/observability, o que não é o caso aqui).

### 3.5. API Design / Interface Changes

* **`internal/api/GeminiClientInterface`**: Não há alterações de assinatura nas funções públicas (`GenerateContent` ainda retorna `(*models.ModelOutput, error)`). A mudança está na qualidade e estrutura do `error` retornado.
* **`internal/errors/errors.go`**:
    * **Novos Métodos em `GeminiError`**: Implementação de métodos como `IsAuth()`, `IsRateLimit()`, `IsNetwork()` para facilitar a manipulação.

---

## 4. Key Considerations & Risk Mitigation

### 4.1. Technical Risks & Challenges

| Risco | Descrição | Estratégia de Mitigação |
| :--- | :--- | :--- |
| **Parsing de Corpo de Erro** | O corpo de resposta de erro da API Gemini pode ser não-JSON, HTML ou um JSON aninhado de difícil extração, mudando frequentemente. | Apenas logar o corpo do erro (ou um snippet) no `GeminiError` e usar a informação de status code + código de erro Gemini (extraído de GJSON se possível) para a lógica. Não depender do corpo da mensagem para a lógica de retry. |
| **Aumento de Complexidade** | A introdução de um erro base pode complicar as verificações de erro existentes. | Usar as primitivas `errors.Is` e `errors.As` do Go para manter a verificação de tipo limpa. (Ex: `if errors.Is(err, errors.ErrAuthFailed)`). |
| **Backward Compatibility** | Código que usa `err.Error() == "expected string"` pode quebrar se a mensagem de erro mudar. | Focar na refatoração interna (API/erros) e manter a compatibilidade externa (comandos/TUI) usando *type assertions* ou `errors.Is` para verificar o erro subjacente. |

### 4.2. Dependencies
* **Go 1.13+ (`errors` package)**: O plano depende fortemente das primitivas de *error wrapping* (`errors.Is`, `errors.As`). A Go.mod aponta para Go 1.23.10, o que atende a esse requisito.
* **Dependência Interna**: As tarefas das Fases 2 e 3 são totalmente dependentes da conclusão da Fase 1.

### 4.3. Non-Functional Requirements (NFRs) Addressed

| NFR | Contribuição do Design |
| :--- | :--- |
| **Rastreabilidade (Observability)** | `GeminiError` inclui `Endpoint`, `StatusCode`, e `InternalCode`, fornecendo todos os metadados para rastrear a origem da falha na API. |
| **Confiabilidade (Reliability)** | O refatoramento de `isAuthError` para usar `errors.Is` torna a lógica de *retry* mais robusta e menos propensa a quebras devido a mudanças de mensagem. |
| **Manutenibilidade** | Centraliza a definição de erro e a lógica de mapeamento em `internal/errors/errors.go` e a lógica de manipulação (*retry*) na função `GenerateContent`, melhorando a separação de preocupações. |

---

## 5. Success Metrics / Validation Criteria
* Todos os erros da camada `internal/api` são do tipo `GeminiError` ou de um tipo que envolve `GeminiError`.
* A lógica de *retry* em `GenerateContent` utiliza `errors.Is` ou `errors.As` e não depende da checagem de `StatusCode` diretamente.
* Em caso de falha de requisição, o erro retornado inclui: `Endpoint`, `StatusCode` (se aplicável), e a causa raiz (wrapped error).
* A CLI e a TUI exibem mensagens de erro mais específicas e úteis ao usuário.

---

## 6. Assumptions Made
* **Go Version:** Assumimos Go 1.23+ é o ambiente de destino, permitindo o uso total do *error wrapping* (`%w` e `errors.Is/As`).
* **Error Body:** Assumimos que a leitura do corpo da resposta de erro HTTP não causará problemas de performance significativos, pois isso só ocorrerá em caso de falha (status code não-200).

## 7. Open Questions / Areas for Further Investigation
* **Corpo de Erro Gemini:** Qual é o formato exato (JSON, HTML, string) dos diferentes tipos de respostas de erro da API Gemini? A validação disso é crucial para a Tarefa 2.1 (Enhance `APIError` creation with Body).
* **Códigos de Erro de Rede:** Devemos definir códigos de erro canônicos internos para falhas de rede comuns (e.g., `ErrNetworkTimeout`, `ErrConnectionRefused`) ou apenas confiar no *error wrapping* de `net/http` e `tls-client`?

**Key discussion points for the team before finalizing or starting implementation:**
1.  Definir o conjunto mínimo de códigos de erro internos (além dos do Gemini) que a lógica do cliente deve reconhecer (e.g., `ErrClientClosed`, `ErrInvalidCookie`).
2.  Discutir se a inclusão do corpo de erro HTTP deve ser limitada a `APIError`s com status 4xx e 5xx, ou se deve ser logada para todos os erros não-200.