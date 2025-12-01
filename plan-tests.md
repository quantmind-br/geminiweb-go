╭────────────────────────────────────────────────────────────────────────────╮
│                                                                            │
│    Refactoring/Design Plan: Aumento da Cobertura de Testes (Mínimo         │
│   80%)                                                                     │
│                                                                            │
│   ## 1. Executive Summary & Goals                                          │
│                                                                            │
│   O objetivo principal deste plano é aumentar a cobertura de testes de     │
│   unidade para um mínimo de 80% na aplicação  geminiweb-go . A             │
│   cobertura atual, embora existente, possui lacunas evidentes,             │
│   especialmente nas funções com lógica de negócios e I/O que requerem      │
│   mocking e testes de concorrência.                                        │
│                                                                            │
│   ### Key Goals                                                            │
│                                                                            │
│   1. Atingir e Manter Cobertura: Alcançar uma cobertura de testes          │
│   sustentável de, no mínimo, 80% do código-fonte Go.                       │
│   2. Melhorar a Robustez: Aumentar a confiança no sistema através da       │
│   cobertura de cenários de erro, concorrência e integração (simulada).     │
│   3. Padronização de Mocking: Implementar um padrão claro de mocking       │
│   para dependências externas (ex:  tls_client.HttpClient ,  browser.       │
│   ExtractGeminiCookies ) para isolar as unidades de código.                │
│                                                                            │
│   --------                                                                 │
│                                                                            │
│   ## 2. Current Situation Analysis                                         │
│                                                                            │
│   O projeto  geminiweb-go  é uma aplicação CLI em Go com uma               │
│   arquitetura modular ( internal/api ,  internal/commands ,                │
│   internal/config , etc.).                                                 │
│                                                                            │
│   ### Key Pain Points for Test Coverage:                                   │
│                                                                            │
│   * Cobertura Parcial: Embora existam arquivos  _test.go  para quase       │
│   todos os módulos, a cobertura é inconsistente.                           │
│     * Exemplo:  internal/api/client.go  possui testes ( client_test.go     │
│     ), mas a cobertura para funções complexas como  RefreshFromBrowser     │
│     pode ser baixa devido à dificuldade de mocking de I/O de navegador.    │
│   * Foco Insuficiente em Erros e Borda: Muitas funções de I/O e            │
│   utilitárias (ex:  internal/api/rotate.go ,  internal/commands/* ,        │
│   internal/config/* ) podem ter cenários de erro e condições de limite     │
│   não testadas.                                                            │
│   * Falta de Testes para Lógica CLI/TUI: O pacote  internal/commands/      │
│   e  internal/tui/  (com exceção de alguns testes de TUI) têm lógica       │
│   de execução principal que precisa de mais testes, especialmente          │
│   runQuery  e  runChat .                                                   │
│                                                                            │
│   --------                                                                 │
│                                                                            │
│   ## 3. Proposed Solution / Refactoring Strategy                           │
│                                                                            │
│   A estratégia se concentrará na injeção de dependência para facilitar     │
│   o mocking e na criação de testes de unidade focados em: 1) Cobertura     │
│   de Lógica Crítica, 2) Cenários de Erro, e 3) Condições de                │
│   Concorrência.                                                            │
│                                                                            │
│   ### 3.1. High-Level Design / Architectural Overview                      │
│                                                                            │
│   O foco principal será na camada  internal/api  e  internal/commands      │
│   .                                                                        │
│                                                                            │
│   1. Implementar Mocks de Interface: Criar interfaces Go para serviços     │
│   críticos que interagem com o sistema de arquivos, rede ou libs de        │
│   terceiros, como:                                                         │
│     *  tls_client.HttpClient  (já tem um mock no  token_test.go , mas      │
│     precisa ser padronizado e reutilizado).                                │
│     *  browser.CookieExtractor  (para abstrair a biblioteca  kooky ).      │
│     *  config.CookieLoaderSaver  (para I/O de arquivo).                    │
│   2. Refatoração Mínima Necessária: Modificar o                            │
│   internal/api/GeminiClient  e funções relevantes em                       │
│   internal/commands  para aceitar essas interfaces via injeção.            │
│                                                                            │
│   ### 3.2. Key Components / Modules for Test Enhancement                   │
│                                                                            │
│    Componente/Pacote          | Descrição da Necessidade                   │
│   ----------------------------+-------------------------------------       │
│     internal/api/client.go    | Aumentar a cobertura de  Init() ,          │
│                               |  Close() ,  RefreshFromBrowser()  e        │
│                               | métodos de acesso simultâneo.              │
│     internal/api/generate.go  | Cobrir todos os cenários de erro de        │
│                               |  GenerateContent  e                        │
│                               |  doGenerateContent , especialmente         │
│                               | a lógica de retry com                      │
│                               |  RefreshFromBrowser  e todos os            │
│                               | casos de  handleErrorCode .                │
│     internal/api/rotate.go    | Testar  RotateCookies  (e seu rate         │
│                               | limiting) com mocking de                   │
│                               |  tls_client.HttpClient . Testar            │
│                               |  CookieRotator.Start/Stop .                │
│     internal/api/token.go     | Garantir 100% de cobertura para            │
│                               |  GetAccessToken  (sucesso, 401,            │
│                               | erro de regex).                            │
│     internal/commands/*       | Adicionar testes de integração             │
│                               | (simulada) para  runQuery ,                │
│                               |  runChat ,  runAutoLogin  para             │
│                               | garantir que o fluxo de                    │
│                               | argumentos/flags/I/O funcione.             │
│     internal/config/*         | Testar exaustivamente cenários de          │
│                               | I/O de arquivo para                        │
│                               |  Load/SaveConfig  e                        │
│                               |  Load/SaveCookies  (ex: permissões,        │
│                               | I/O erros, JSON malformado).               │
│     internal/history/*        | Aumentar a cobertura de  store.go ,        │
│                               | especialmente a lógica de I/O,             │
│                               | ordenação e manipulação de                 │
│                               | metadados.                                 │
│                                                                            │
│   ### 3.3. Detailed Action Plan / Phases                                   │
│                                                                            │
│   A execução deve ser iterativa. A prioridade é cobrir a lógica            │
│   central de API e os caminhos de erro.                                    │
│                                                                            │
│   #### Phase 1: Foundation & Core API (High Priority)                      │
│                                                                            │
│   Objective(s): Padronizar o mocking de HTTP e aumentar a cobertura do     │
│   internal/api/ .                                                          │
│                                                                            │
│    Task               | Rationale/Goal      | | Deliverable/Criter…        │
│   --------------------+---------------------+-+---------------------       │
│    1.1: Padronizar    | Mover e formalizar  | |  MockHttpClient  e         │
│     MockHttpClient    |  MockHttpClient     | |  NewMockResponseBod        │
│                       | (atualmente em      | | y  reutilizáveis           │
│                       |  token_test.go )    | | definidos.                 │
│                       | para um arquivo de  | |                            │
│                       | teste compartilhado | |                            │
│                       | (e.g.,              | |                            │
│                       |  internal/api/mock_ | |                            │
│                       | test.go ) para      | |                            │
│                       | reutilização.       | |                            │
│    1.2: Refatorar     | Refatorar           | | Interface                  │
│     GeminiClient      |  GeminiClient  para | |  BrowserCookieExtra        │
│    para aceitar       | usar uma interface  | | ctor  criada e             │
│    interfaces         | para o extrator de  | | injetada em                │
│                       | cookies do          | |  GeminiClient .            │
│                       | navegador,          | |                            │
│                       | permitindo mocking  | |                            │
│                       | em                  | |                            │
│                       |  RefreshFromBrowser | |                            │
│                       |  .                  | |                            │
│    1.3: Aumentar      | Cobrir              | | Cobertura de               │
│    Cobertura de       |  RefreshFromBrowser | |  client.go  > 90%.         │
│     client.go         |                     | |                            │
│                       | (sucesso/falha/rate | |                            │
│                       | limit/falha de      | |                            │
│                       | token após          | |                            │
│                       | refresh),  Init     | |                            │
│                       | (erro de token), e  | |                            │
│                       | concorrência em     | |                            │
│    1.4: Aumentar      | Garantir 100% de    | | Cobertura de               │
│    Cobertura de       | cobertura para      | |  token.go  = 100%.         │
│     token.go          |  GetAccessToken ,   | |                            │
│                       | testando todos os   | |                            │
│                       | caminhos de erro    | |                            │
│                       | (HTTP status, erro  | |                            │
│                       | de regex).          | |                            │
│    1.5: Aumentar      | Testar todos os     | | Cobertura de               │
│    Cobertura de       | cenários de         | |  rotate.go  > 90%.         │
│     rotate.go         |  RotateCookies      | |                            │
│                       | (401, rate limit,   | |                            │
│                       | status 500, sem     | |                            │
│                       | cookie de resposta) | |                            │
│                       | usando              | |                            │
│                       |  MockHttpClient .   | |                            │
│    1.6: Aumentar      | Cobrir todos os     | | Cobertura de               │
│    Cobertura de       | casos de            | |  generate.go  >            │
│     generate.go       |  handleErrorCode  e | | 85%.                       │
│                       | a lógica de retry   | |                            │
│                       | em                  | |                            │
│                       |  GenerateContent    | |                            │
│                       | (chamar             | |                            │
│                       |  isAuthError  e     | |                            │
│                       |  RefreshFromBrowser | |                            │
│                                                                            │
│   #### Phase 2: CLI Commands & Utilities (Medium Priority)                 │
│                                                                            │
│   Objective(s): Cobrir a lógica de entrada/saída (I/O) da CLI e as         │
│   funções auxiliares.                                                      │
│                                                                            │
│    Task               | Rationale/Goal      | | Deliverable/Criter…        │
│   --------------------+---------------------+-+---------------------       │
│    2.1: Testes de I/O | Testar a lógica de  | | Cobertura de               │
│    em  runQuery       |  runQuery  em       | |  runQuery  (exceto         │
│                       |  internal/commands/ | | TUI) > 80%.                │
│                       | query.go  para      | |                            │
│                       | todos os caminhos   | |                            │
│                       | de entrada          | |                            │
│                       | (argumento,  -f ,   | |                            │
│                       | stdin) e saída ( -  | |                            │
│                       | o , clipboard).     | |                            │
│                       | Requer abstração do | |                            │
│                       |  GeminiClient  para | |                            │
│                       | simular a geração   | |                            │
│    2.2: Testar a      | Testar              | | Cobertura de               │
│    Resolução de       | exaustivamente      | |  root.go  (funções         │
│    Flags/Config       |  getModel  e        | | auxiliares) > 90%.         │
│                       |  getBrowserRefresh  | |                            │
│                       | ( internal/commands | |                            │
│                       | /root.go ) com      | |                            │
│                       | várias combinações  | |                            │
│                       | de flags e          | |                            │
│                       | configuração        | |                            │
│                       | (incluindo caminhos | |                            │
│                       | de erro de          | |                            │
│                       |  browser.ParseBrows | |                            │
│                       | er ).               | |                            │
│                       |                     | |                            │
│    2.3: Testes de I/O | Adicionar testes de | | Cobertura de               │
│    de Configuração    | unidade para        | |  config/*  > 85%.          │
│                       |  internal/config/co | |                            │
│                       | nfig.go  e          | |                            │
│                       |  cookies.go  para   | |                            │
│                       | falhas de I/O de    | |                            │
│                       | arquivo (ex:        | |                            │
│                       |  os.WriteFile       | |                            │
│                       | falha, JSON         | |                            │
│                       | malformado) e       | |                            │
│                       | caminhos de         | |                            │
│                       | validação de        | |                            │
│                       |  ValidateCookies .  | |                            │
│    2.4: Testes de     | Cobrir a lógica de  | | Cobertura de               │
│    Lógica de História | ordenação, loading  | |  history/store.go          │
│                       | e saving de         | | > 85%.                     │
│                       |  internal/history/s | |                            │
│                       | tore.go ,           | |                            │
│                       | garantindo o teste  | |                            │
│                       | de  UpdateTitle  e  | |                            │
│                       |  ClearAll .         | |                            │
│                                                                            │
│   #### Phase 3: TUI & Code Structure (Low Priority)                        │
│                                                                            │
│   Objective(s): Adicionar cobertura aos componentes TUI e garantir que     │
│   a base do código esteja totalmente testada.                              │
│                                                                            │
│    Task               | Rationale/Goal      | | Deliverable/Criter…        │
│   --------------------+---------------------+-+---------------------       │
│    3.1: Testes de     | Aumentar a          | | Cobertura de               │
│    Modelo TUI         | cobertura de        | |  internal/tui/model        │
│    ( model.go )       |  Model.Update  em   | | .go  (lógica de            │
│                       |  internal/tui/model | | estado) > 70%.             │
│                       | .go  para cenários  | |                            │
│                       | chave (Enter para   | |                            │
│                       | enviar, Escape,     | |                            │
│                       |  responseMsg ,      | |                            │
│                       |  errMsg ). Requer   | |                            │
│                       | mocking de          | |                            │
│                       |  ChatSessionInterfa | |                            │
│                       | ce .                | |                            │
│    3.2: Testes de     | Adicionar testes    | |  updateViewport            │
│    Renderização de    | para                | | coberto para todos         │
│    Mensagens          |  updateViewport  em | | os tipos de                │
│                       |  internal/tui/model | | mensagem.                  │
│                       | .go  para garantir  | |                            │
│                       | que as mensagens    | |                            │
│                       | (incluindo          | |                            │
│                       |  thoughts ) sejam   | |                            │
│                       | renderizadas        | |                            │
│                       | corretamente (sem   | |                            │
│                       | falha de panic).    | |                            │
│    3.3: Testes de     | Garantir a          | | Cobertura de               │
│    Comandos           | cobertura de        | |  internal/commands         │
│    Utilitários        | comandos como       | | (excluindo TUI) >          │
│                       |  internal/commands/ | | 80%.                       │
│                       | config.go ,         | |                            │
│                       |  history.go ,       | |                            │
│                       |  persona.go  (pelo  | |                            │
│                       | menos os caminhos   | |                            │
│                       | de sucesso e I/O de | |                            │
│                       | erro).              | |                            │
│    3.4: Relatório     | Executar o          | | Relatório                  │
│    Final de Cobertura |  make test-         | |  coverage.html             │
│                       | coverage  e         | | mostrando cobertura        │
│                       | garantir que o      | | > 80%.                     │
│                       | relatório final     | |                            │
│                                                                            │
│   --------                                                                 │
│                                                                            │
│   ## 4. Key Considerations & Risk Mitigation                               │
│                                                                            │
│   ### 4.1. Technical Risks & Challenges                                    │
│                                                                            │
│   * Risco: Mocking do TLS Client: O  tls-client  é uma dependência de      │
│   terceiros que simula um navegador. Usar a interface  tls_client.         │
│   HttpClient  é essencial. O risco é ter testes que passam, mas que        │
│   falham na concorrência da vida real.                                     │
│     * Mitigação: Padronizar e refinar o  MockHttpClient  e focar os        │
│     testes de concorrência ( client_test.go ) na lógica do mutex (         │
│     sync.RWMutex ).                                                        │
│   * Risco: Teste de I/O de Arquivo: Testar falhas de I/O de arquivo        │
│   (ex: permissão negada,  os.IsNotExist ) é complexo sem usar mocks de     │
│   sistema de arquivos (filesystem mocking).                                │
│     * Mitigação: Usar diretórios temporários ( t.TempDir() ) e simular     │
│     erros de forma direta onde possível (ex: tentar ler um arquivo         │
│     inexistente ou gravar em um caminho inválido simulado). Testar         │
│     permissões com  os.WriteFile  com  0o600  e verificar.                 │
│   * Risco: Dependência da Configuração Global: Funções como  config.       │
│   LoadConfig()  e  config.GetCookiesPath()  dependem de  $HOME  e têm      │
│   impacto global.                                                          │
│     * Mitigação: Usar  t.TempDir()  e sobrescrever a variável de           │
│     ambiente  $HOME  no início dos testes relevantes (como feito em        │
│     internal/commands/history_test.go ) para isolar o ambiente.            │
│                                                                            │
│                                                                            │
│   ### 4.2. Dependencies                                                    │
│                                                                            │
│   * Interna: A Fase 2 depende da conclusão do mocking em  internal/api     │
│   (Fase 1).                                                                │
│   * Ferramentas: O uso de  go test -coverprofile  e  go tool cover -       │
│   html  (via  make test-coverage ) é crucial para a validação do           │
│   objetivo.                                                                │
│   * Conhecimento: A equipe deve estar familiarizada com as interfaces      │
│   Go e o padrão de Injeção de Dependência para aplicar a refatoração       │
│   do  GeminiClient  (Task 1.2).                                            │
│                                                                            │
│   ### 4.3. Non-Functional Requirements (NFRs) Addressed                    │
│                                                                            │
│   * Manutenibilidade: O aumento na cobertura e a introdução de             │
│   interfaces para mocking tornam o código mais fácil de entender e         │
│   manter, pois as dependências são explícitas e os testes documentam o     │
│   comportamento esperado.                                                  │
│   * Confiabilidade: A cobertura de 80%+ garante que a maior parte da       │
│   lógica crítica de inicialização (token, cookies) e de geração de         │
│   conteúdo seja robusta a falhas esperadas (erros de API, autenticação     │
│   expirada, falhas de rede).                                               │
│   * Testabilidade: A refatoração implícita (ID) aumenta diretamente a      │
│   Testabilidade do sistema.                                                │
│                                                                            │
│   --------                                                                 │
│                                                                            │
│   ## 5. Success Metrics / Validation Criteria                              │
│                                                                            │
│    Métrica                | Critério de Sucesso                            │
│   ------------------------+-----------------------------------------       │
│    Cobertura de Código    | Relatório de  go test -coverprofile            │
│                           | mostrando uma cobertura total de, no           │
│                           | mínimo, 80% (exibido em                        │
│                           |  coverage.html ).                              │
│    Testes de Concorrência | Todos os testes em  client_test.go             │
│                           | (especialmente                                 │
│                           |  TestGeminiClient_ConcurrentAccess  e          │
│                           |  TestGeminiClient_ConcurrencyWithInit )        │
│                           | devem passar consistentemente sob o            │
│                           |  go test -race .                               │
│    Caminhos de Erro       | Testes explícitos para todas as funções        │
│                           | de I/O ( LoadConfig ,  RotateCookies ,         │
│                           |  GenerateContent ) devem falhar                │
│                           | graciosamente com o erro correto ao            │
│                           | simular condições de erro (401, falta          │
│                           | de token, falha de I/O).                       │
│    CI/CD                  | O target  make test  deve ser integrado        │
│                           | ao pipeline de CI e deve passar                │
│                           | consistentemente, com um gate opcional         │
│                           | para aplicar a cobertura de 80%.               │
│                                                                            │
│   --------                                                                 │
│                                                                            │
│   ## 6. Assumptions Made                                                   │
│                                                                            │
│   1. Ferramentas de Teste: O ambiente de desenvolvimento tem acesso a      │
│   go test ,  go tool cover  e  t.TempDir()  (garantido pelo Go 1.23+).     │
│   2. Ambiente Isolado: Os testes conseguem isolar a I/O de disco e de      │
│   rede externa através de mocking de interfaces e manipulação de           │
│   variáveis de ambiente ( $HOME ).                                         │
│   3. Lógica TUI/CLI: A maior parte da lógica de negócios e I/O que         │
│   precisa de cobertura está em  internal/api ,  internal/config ,          │
│   internal/commands/query.go  e  internal/history . A lógica puramente     │
│   de visualização (como em  internal/tui/styles.go ) pode ser excluída     │
│   do cálculo de cobertura, se necessário, para atingir o target de 80%     │
│   de forma pragmática.                                                     │
│                                                                            │
│   --------                                                                 │
│                                                                            │
│   ## 7. Open Questions / Areas for Further Investigation                   │
│                                                                            │
│   1. Target de Cobertura por Pacote: Devemos definir um target de          │
│   cobertura mais alto (ex: 90%+) para pacotes críticos ( internal/api      │
│   ) para compensar pacotes menos críticos ou mais difíceis de testar       │
│   (ex:  internal/tui )?                                                    │
│   2. Mocking do  kooky : É necessário criar uma interface customizada      │
│   para abstrair o  browserutils/kooky  e injetá-la em                      │
│   internal/browser/browser.go  ou o teste pode ser feito indiretamente     │
│   através de uma refatoração menor no  internal/api/client.go  para        │
│   expor o extrator? (A Task 1.2 sugere a primeira opção, mas a segunda     │
│   pode ser mais leve).                                                     │
│   3. Integração TUI: É viável testar a lógica do TUI de forma              │
│   automatizada sem mockar completamente  bubbletea ? (A Task 3.1 foca      │
│   na lógica do  Model.Update , que é o suficiente para o requisito de      │
│   80%).                                                                    │
╰────────────────────────────────────────────────────────────────────────────╯