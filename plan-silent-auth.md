╭────────────────────────────────────────────────────────────────────────────╮
│                                                                            │
│    Refactoring/Design Plan: Autenticação de Inicialização Silenciosa       │
│   do GeminiClient                                                          │
│                                                                            │
│   ## 1. Executive Summary & Goals                                          │
│                                                                            │
│   O objetivo principal deste plano é refatorar o ciclo de vida de          │
│   inicialização do  GeminiClient  para incorporar uma tentativa            │
│   silenciosa e automática de auto-login via navegador como um              │
│   mecanismo                                                                │
│   de fallback. Isso garante que o cliente tente se autenticar no           │
│   estágio mais inicial, antes de qualquer requisição de token ou API,      │
│   melhorando significativamente a experiência do usuário quando os         │
│   cookies salvos expiram ou não existem.                                   │
│                                                                            │
│   ### Key Goals:                                                           │
│                                                                            │
│   1. Centralizar a Lógica de Autenticação: Mover a responsabilidade de     │
│   obter cookies (carregamento de arquivo ou auto-login via navegador)      │
│   para dentro do  GeminiClient.Init() , simplificando os comandos CLI      │
│   ( chat.go ,  query.go ).                                                 │
│   2. Habilitar Auto-login Silencioso: Se o carregamento de  cookies.       │
│   json  falhar, o  Init()  deve tentar a extração automática de            │
│   cookies do navegador como um passo de pré-autenticação, usando o         │
│   browser-refresh  configurado (ou  auto  por padrão).                     │
│   3. Priorizar Configuração: Manter a prioridade de carregar cookies       │
│   existentes antes de tentar o auto-login, e garantir que o auto-login     │
│   de retry em  generate.go  mantenha seu rate-limiting.                    │
│                                                                            │
│   --------                                                                 │
│                                                                            │
│   ## 2. Current Situation Analysis                                         │
│                                                                            │
│   O mecanismo de autenticação atual é distribuído e reativo:               │
│                                                                            │
│   *  commands/chat.go  /  query.go : Dependem do sucesso do  config.       │
│   LoadCookies()  antes de instanciar o  api.NewClient() . Se o arquivo     │
│   de cookies não for encontrado, o aplicativo falha antes mesmo de         │
│   criar o cliente.                                                         │
│   *  api.NewClient : Requer cookies válidos para inicialização.            │
│   *  api.GenerateContent : Contém a única lógica de fallback de auto-      │
│   login ( client.RefreshFromBrowser() ) que é reativa, ou seja, só é       │
│   acionada após a falha de autenticação (401) de uma requisição de         │
│   geração de conteúdo.                                                     │
│                                                                            │
│   O principal ponto de dor é que o auto-login não é uma etapa proativa     │
│   de inicialização. Se os cookies expirarem ou forem excluídos, o          │
│   usuário é forçado a executar  geminiweb auto-login  manualmente,         │
│   mesmo que o  --browser-refresh  esteja ativado.                          │
│                                                                            │
│   --------                                                                 │
│                                                                            │
│   ## 3. Proposed Solution / Refactoring Strategy                           │
│                                                                            │
│   A estratégia é remover o  config.LoadCookies()  dos arquivos de          │
│   comando ( chat.go ,  query.go ) e injetar o carregamento e a lógica      │
│   de fallback do auto-login no  GeminiClient.Init() .                      │
│                                                                            │
│   ### 3.1. High-Level Design / Architectural Overview                      │
│                                                                            │
│   $$mermaid                                                                │
│   graph TD                                                                 │
│   A[Start Command/Query] --> B(api.NewClient(nil, opts));                  │
│   B --> C(client.Init());                                                  │
│   C --> D{config.LoadCookies()};                                           │
│   D -- Success --> E[Update Client Cookies];                               │
│   D -- Fail / Not Found --> F{Attempt Silent AutoLogin};                   │
│   F -- Success --> G[Save & Update Client Cookies];                        │
│   F -- Fail / Error --> H[Exit with Auth Error];                           │
│   E --> I(api.GetAccessToken());                                           │
│   G --> I;                                                                 │
│   I -- Success --> J[Start Cookie Rotator];                                │
│   I -- Fail --> H;                                                         │
│   J --> K[Ready for API Calls];                                            │
│   H --> L[Exit CLI];                                                       │
│   $$                                                                       │
│                                                                            │
│   ### 3.2. Key Components / Modules                                        │
│                                                                            │
│    Componente          | Ação Proposta       | Racional                    │
│   ---------------------+---------------------+----------------------       │
│     internal/commands/ | Remover             | Simplificar a lógica        │
│    *                   |  config.LoadCookies | de inicialização dos        │
│                        | ()  e passar  nil   | comandos e                  │
│                        | para                | centralizar a               │
│                        |  api.NewClient() .  | autenticação.               │
│     internal/api/clien | Novo método         | Isolamento da lógica        │
│    t.go                |  attemptInitialAuth | de fallback e               │
│                        | () error  que       | autenticação.               │
│                        | encapsula           |                             │
│                        |  LoadCookies()  e   |                             │
│                        |  RefreshFromBrowser |                             │
│                        | () .                |                             │
│     internal/api/clien | Modificar para      | Permitir a criação          │
│    t.go:NewClient      | aceitar             | do cliente sem              │
│                        |  *config.Cookies    | autenticação                │
│                        | opcional ( nil ).   | imediata, delegando         │
│                        | Remover a validação | a responsabilidade          │
│                        | de cookies se       | para  Init() .              │
│                        |  nil .              |                             │
│     internal/api/clien | Chamar              | Fazer de  Init()  a         │
│    t.go:Init           |  c.attemptInitialAu | função primária de          │
│                        | th() . Se for bem-  | autenticação/inicial        │
│                        | sucedido,           | ização.                     │
│                        | prosseguir para     |                             │
│                        |  GetAccessToken()   |                             │
│                        | e                   |                             │
│                        |  NewCookieRotator() |                             │
│     internal/api/clien | Manter o rate       | Distinguir o fluxo          │
│    t.go:RefreshFromBro | limiting e a        | de inicialização            │
│    wser                | checagem de         | proativa (novo              │
│                        |  browserRefresh     | método) do retry            │
│                        | para preservar o    | reativo (método             │
│                        | comportamento de    | existente).                 │
│                        | retry reativo.      |                             │
│                                                                            │
│   ### 3.3. Detailed Action Plan / Phases                                   │
│                                                                            │
│   #### Phase 1: Modificação do GeminiClient e Autenticação de              │
│   Inicialização                                                            │
│                                                                            │
│   * Objective(s): Introduzir a lógica de fallback de auto-login no         │
│   processo de inicialização do cliente.                                    │
│   * Priority: High                                                         │
│                                                                            │
│    Task               | Rationale/Goal      | | Deliverable/Criter…        │
│   --------------------+---------------------+-+---------------------       │
│    1.1: Refatorar     | Permitir a criação  | |  NewClient  aceita         │
│     api.NewClient     | com cookies  nil .  | |  *config.Cookies           │
│                       |                     | | nulo e não chama           │
│                       |                     | |  config.ValidateCoo        │
│                       |                     | | kies  se nulo.             │
│    1.2: Novo método   | Criar uma função    | | Novo método chama          │
│     (c *GeminiClient) | para extrair        | |  browser.ExtractGem        │
│     initialBrowserRef | cookies do          | | iniCookies  com            │
│    resh()             | navegador de forma  | |  c.browserRefreshTy        │
│                       | não-rate-limited.   | | pe  (ou  auto  se          │
│                       |                     | | desabilitado),             │
│                       |                     | | salva cookies, e           │
│                       |                     | | atualiza                   │
│                       |                     | |  c.cookies  e              │
│    1.3: Novo método   | Tentar carregar     | | O método retorna           │
│     (c *GeminiClient) | cookies do disco,   | |  nil  se                   │
│     attemptInitialAut | com fallback para   | |  c.cookies  for            │
│    h()                |  initialBrowserRefr | | válido e                   │
│                       | esh()  em caso de   | |  c.accessToken  for        │
│                       | falha.              | | definido.                  │
│    1.4: Refatorar     | Delegar a           | |  Init()  chama             │
│     GeminiClient.Init | autenticação        | |  c.attemptInitialAu        │
│    ()                 | inicial ao novo     | | th()  e                    │
│                       | método.             | |  GetAccessToken()          │
│                       |                     | | é removido de              │
│                       |                     | |  Init()  para ser          │
│                       |                     | | movido para dentro         │
│                       |                     | | de                         │
│                       |                     | |  attemptInitialAuth        │
│                       |                     | | ()  (ou                    │
│                       |                     | | imediatamente após,        │
│                       |                     | | se a lógica for            │
│    1.5: Adaptação de  | Garantir que a nova | | Testes de                  │
│     internal/api/clie | arquitetura de      | |  api/client.go             │
│    nt_test.go         | autenticação de     | | modificados/adicion        │
│                       | inicialização       | | ados para cobrir o         │
│                       | funcione.           | | fluxo de                   │
│                       |                     | |  LoadCookies  fail         │
│                       |                     | | ->                         │
│                       |                     | |  initialBrowserRefr        │
│                       |                     | | esh  success.              │
│                                                                            │
│   #### Phase 2: Integração nos Comandos CLI                                │
│                                                                            │
│   * Objective(s): Simplificar os comandos  chat  e  query .                │
│   * Priority: Medium                                                       │
│                                                                            │
│    Task               | Rationale/Goal      | | Deliverable/Criter…        │
│   --------------------+---------------------+-+---------------------       │
│    2.1: Modificar     | Remover a           | |  runChat()  chama          │
│     commands/chat.go: | responsabilidade de | |  api.NewClient(nil,        │
│    runChat()          | carregamento de     | |  clientOpts...)  e         │
│                       | cookies.            | | lida apenas com o          │
│                       |                     | | erro final de              │
│                       |                     | |  client.Init() .           │
│    2.2: Modificar     | Remover a           | |  runQuery()  chama         │
│     commands/query.go | responsabilidade de | |  api.NewClient(nil,        │
│    :runQuery()        | carregamento de     | |  clientOpts...)  e         │
│                       | cookies.            | | lida apenas com o          │
│                       |                     | | erro final de              │
│                       |                     | |  client.Init() .           │
│    2.3: Atualizar     | Garantir que os     | | Testes de comandos         │
│     commands/*_test.g | testes de comandos  | | passam com a nova          │
│    o                  | simulem a falha de  | | arquitetura.               │
│                       | cookies             | |                            │
│                       | corretamente.       | |                            │
│                                                                            │
│   ### 3.4. Data Model Changes (if applicable)                              │
│                                                                            │
│   Nenhum.                                                                  │
│                                                                            │
│   ### 3.5. API Design / Interface Changes (if applicable)                  │
│                                                                            │
│   *  internal/api/client.go :                                              │
│     * Modified:  func NewClient(cookies *config.Cookies, opts ...          │
│     ClientOption) (*GeminiClient, error) :  cookies  pode ser  nil .       │
│     * Modified:  func (c *GeminiClient) Init() error : Agora encapsula     │
│     a tentativa de carregamento/extração de cookies e a obtenção do        │
│     access token.                                                          │
│     * New Private Method:  func (c *GeminiClient)                          │
│     initialBrowserRefresh() error : Realiza a extração de cookies sem      │
│     rate-limiting.                                                         │
│                                                                            │
│                                                                            │
│   --------                                                                 │
│                                                                            │
│   ## 4. Key Considerations & Risk Mitigation                               │
│                                                                            │
│   ### 4.1. Technical Risks & Challenges                                    │
│                                                                            │
│    Risco                          | Mitigação                              │
│   --------------------------------+---------------------------------       │
│    Rate Limit vs. Inicialização   | Garantir que o novo método             │
│    (Task 1.4)                     |  initialBrowserRefresh()               │
│                                   | ignore apenas o rate limit, mas        │
│                                   | mantenha o  browserRefreshType         │
│                                   | e a lógica de                          │
│                                   |  browserExtractor . O                  │
│                                   |  RefreshFromBrowser  existente         │
│                                   | em  generate.go  deve manter o         │
│    Atraso na Inicialização        | A extração de cookies via              │
│                                   | navegador pode ser lenta (até          │
│                                   | 30s se o timeout for atingido).        │
│                                   | O spinner em                           │
│                                   |  commands/query.go  e                  │
│                                   |  commands/chat.go  deve                │
│                                   | comunicar claramente que a             │
│                                   | conexão/autenticação está em           │
│                                   | andamento. O timeout de 30s (em        │
│                                   |  RefreshFromBrowser ) é                │
│                                   | aceitável para um passo de             │
│    Concorrência                   | As chamadas de  Init()  são            │
│                                   | protegidas por  c.mu.Lock() ,          │
│                                   | garantindo que a tentativa de          │
│                                   | autenticação e a obtenção do           │
│                                   | token sejam thread-safe.               │
│                                                                            │
│   ### 4.2. Dependencies                                                    │
│                                                                            │
│   * Internal:  internal/api/client.go  depende de                          │
│   internal/config/cookies.go  e  internal/browser/browser.go .             │
│   * Sequência: Phase 1 deve ser concluída e testada antes da Phase 2.      │
│                                                                            │
│   ### 4.3. Non-Functional Requirements (NFRs) Addressed                    │
│                                                                            │
│   * Usability: A principal melhoria é permitir que o usuário               │
│   simplesmente execute  geminiweb chat  ou  geminiweb "query"  mesmo       │
│   que seus cookies tenham expirado, desde que ele esteja logado no         │
│   navegador, eliminando a etapa manual de  auto-login .                    │
│   * Maintainability: Centraliza a lógica de autenticação                   │
│   inicial/fallback no componente  GeminiClient , simplificando a base      │
│   de código dos comandos CLI.                                              │
│   * Reliability: Aumenta a confiabilidade da inicialização do cliente,     │
│   garantindo que ele fará a melhor tentativa possível para se              │
│   autenticar.                                                              │
│                                                                            │
│   --------                                                                 │
│                                                                            │
│   ## 5. Success Metrics / Validation Criteria                              │
│                                                                            │
│   * Test Case 1 (Fresh Install): Em um sistema sem arquivo  cookies.       │
│   json ,  geminiweb chat  deve ser bem-sucedido (desde que o usuário       │
│   esteja logado no Gemini em um navegador suportado).                      │
│   * Test Case 2 (Expired Cookies): Em um sistema com um arquivo            │
│   cookies.json  que contenha cookies obviamente inválidos (ex: valor       │
│   "expired" ), o  client.Init()  deve falhar o  GetAccessToken()           │
│   inicial, acionar o auto-login (novo método) e, se a extração do          │
│   navegador for bem-sucedida, o  client.Init()  deve ser bem-sucedido.     │
│   * Test Case 3 (Existing Valid Cookies): Em um sistema com cookies        │
│   válidos, o auto-login do navegador não deve ser chamado (economia de     │
│   tempo).                                                                  │
│   * Test Case 4 (Retry Integrity): O mecanismo de retry em                 │
│   internal/api/generate.go  deve continuar funcionando, chamando           │
│   RefreshFromBrowser()  com seu rate-limiting ativo.                       │
│                                                                            │
│   ## 6. Assumptions Made                                                   │
│                                                                            │
│   1. A função  browser.ExtractGeminiCookies  é robusta e tem a             │
│   capacidade de obter cookies de sessão quando um usuário está             │
│   autenticado em um navegador suportado.                                   │
│   2. O  browserRefreshType  (seja do flag ou  auto ) será usado para a     │
│   tentativa de auto-login proativa.                                        │
│                                                                            │
│   ## 7. Open Questions / Areas for Further Investigation                   │
│                                                                            │
│   N/A. O plano de ação detalhado (Task 1.3, 1.4) aborda a distinção        │
│   entre os fluxos de auto-login inicial e de retry.                        │
╰────────────────────────────────────────────────────────────────────────────╯