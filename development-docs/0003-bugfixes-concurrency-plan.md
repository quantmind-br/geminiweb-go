# PLAN.md - Correção de Bugs e Problemas de Concorrência

## Visão Geral

Este plano detalha a correção de **27 problemas identificados** na análise do codebase geminiweb-go, incluindo:
- 7 problemas críticos (race conditions, panics, resource leaks)
- 8 problemas de alta severidade
- 12 problemas de moderada/baixa severidade

**Estimativa de Complexidade:** Alta
**Ficheiros Afetados:** 12
**Testes Necessários:** Sim, incluindo testes de race condition

---

## Índice de Fases

1. [Fase 1: Tornar config.Cookies Thread-Safe](#fase-1-tornar-configcookies-thread-safe)
2. [Fase 2: Corrigir CookieRotator](#fase-2-corrigir-cookierotator)
3. [Fase 3: Corrigir ChatSession Race Condition](#fase-3-corrigir-chatsession-race-condition)
4. [Fase 4: Corrigir Erros Ignorados em batch.go](#fase-4-corrigir-erros-ignorados-em-batchgo)
5. [Fase 5: Corrigir Erros io.ReadAll em upload.go](#fase-5-corrigir-erros-ioreadall-em-uploadgo)
6. [Fase 6: Corrigir Cookie Store Leak em browser.go](#fase-6-corrigir-cookie-store-leak-em-browsergo)
7. [Fase 7: Melhorar Error Handling em gems.go](#fase-7-melhorar-error-handling-em-gemsgo)
8. [Fase 8: Corrigir Spinner Double-Close](#fase-8-corrigir-spinner-double-close)
9. [Fase 9: Refatorar Lock em RefreshFromBrowser](#fase-9-refatorar-lock-em-refreshfrombrowser)
10. [Fase 10: Correções Menores](#fase-10-correções-menores)

---

## Fase 1: Tornar config.Cookies Thread-Safe

### Contexto
O struct `config.Cookies` é partilhado entre:
- `GeminiClient` (leitura em requests)
- `CookieRotator` (escrita em background)
- `RefreshFromBrowser` (escrita durante refresh)

Sem sincronização, existe race condition na leitura/escrita de `Secure1PSIDTS`.

### Ficheiro Alvo
`internal/config/cookies.go`

### Alterações Detalhadas

#### 1.1 Adicionar mutex ao struct Cookies

**Antes (linha 9-13):**
```go
type Cookies struct {
	Secure1PSID   string `json:"__Secure-1PSID"`
	Secure1PSIDTS string `json:"__Secure-1PSIDTS,omitempty"`
}
```

**Depois:**
```go
type Cookies struct {
	mu            sync.RWMutex `json:"-"` // Não serializar o mutex
	Secure1PSID   string       `json:"__Secure-1PSID"`
	Secure1PSIDTS string       `json:"__Secure-1PSIDTS,omitempty"`
}
```

#### 1.2 Adicionar getters thread-safe

**Adicionar após linha 13:**
```go
// GetSecure1PSID retorna o cookie __Secure-1PSID de forma thread-safe
func (c *Cookies) GetSecure1PSID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Secure1PSID
}

// GetSecure1PSIDTS retorna o cookie __Secure-1PSIDTS de forma thread-safe
func (c *Cookies) GetSecure1PSIDTS() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Secure1PSIDTS
}

// Snapshot retorna ambos os cookies atomicamente (para serialização)
func (c *Cookies) Snapshot() (psid, psidts string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Secure1PSID, c.Secure1PSIDTS
}

// SetBoth atualiza ambos os cookies atomicamente
func (c *Cookies) SetBoth(psid, psidts string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Secure1PSID = psid
	c.Secure1PSIDTS = psidts
}
```

#### 1.3 Modificar Update1PSIDTS para usar lock

**Antes (linha 150-153):**
```go
func (c *Cookies) Update1PSIDTS(value string) {
	c.Secure1PSIDTS = value
}
```

**Depois:**
```go
func (c *Cookies) Update1PSIDTS(value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Secure1PSIDTS = value
}
```

#### 1.4 Modificar ToMap para ser thread-safe

**Antes (linha 139-148):**
```go
func (c *Cookies) ToMap() map[string]string {
	m := map[string]string{
		"__Secure-1PSID": c.Secure1PSID,
	}
	if c.Secure1PSIDTS != "" {
		m["__Secure-1PSIDTS"] = c.Secure1PSIDTS
	}
	return m
}
```

**Depois:**
```go
func (c *Cookies) ToMap() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	m := map[string]string{
		"__Secure-1PSID": c.Secure1PSID,
	}
	if c.Secure1PSIDTS != "" {
		m["__Secure-1PSIDTS"] = c.Secure1PSIDTS
	}
	return m
}
```

#### 1.5 Adicionar import de sync

**Adicionar ao bloco de imports:**
```go
import (
	"encoding/json"
	"fmt"
	"os"
	"sync" // ADICIONAR
)
```

### Ficheiros a Atualizar (uso de cookies)

Após esta alteração, os seguintes ficheiros devem usar os novos getters:

| Ficheiro | Linha | Alteração |
|----------|-------|-----------|
| `api/generate.go` | 115-118 | Usar `GetSecure1PSID()` e `GetSecure1PSIDTS()` |
| `api/batch.go` | 97-101 | Usar getters |
| `api/token.go` | 28-32 | Usar getters |
| `api/rotate.go` | 48-51 | Usar getters |
| `api/client.go` | 432-433 | Usar `SetBoth()` |

### Testes Necessários
- [ ] Teste de race condition: múltiplas goroutines a ler/escrever cookies
- [ ] Teste de serialização JSON (mutex não deve aparecer)
- [ ] Teste de `Snapshot()` atomicidade

---

## Fase 2: Corrigir CookieRotator

### Contexto
O `CookieRotator` tem três problemas:
1. Double-close do canal `stopCh` causa panic
2. Após `Stop()`, `Start()` não pode ser chamado (canal fechado)
3. Erros de rotação são silenciosamente ignorados

### Ficheiro Alvo
`internal/api/rotate.go`

### Alterações Detalhadas

#### 2.1 Adicionar callback de erro e criar novo canal em Start()

**Antes (linha 85-103):**
```go
type CookieRotator struct {
	client   tls_client.HttpClient
	cookies  *config.Cookies
	interval time.Duration
	stopCh   chan struct{}
	running  bool
	mu       sync.Mutex
}

func NewCookieRotator(client tls_client.HttpClient, cookies *config.Cookies, interval time.Duration) *CookieRotator {
	return &CookieRotator{
		client:   client,
		cookies:  cookies,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}
```

**Depois:**
```go
// RotatorErrorCallback é chamado quando ocorre um erro na rotação de cookies
type RotatorErrorCallback func(error)

// CookieRotator manages background cookie rotation
type CookieRotator struct {
	client    tls_client.HttpClient
	cookies   *config.Cookies
	interval  time.Duration
	stopCh    chan struct{}
	running   bool
	mu        sync.Mutex
	onError   RotatorErrorCallback // Callback opcional para erros
}

// RotatorOption configura o CookieRotator
type RotatorOption func(*CookieRotator)

// WithErrorCallback define um callback para erros de rotação
func WithErrorCallback(fn RotatorErrorCallback) RotatorOption {
	return func(r *CookieRotator) {
		r.onError = fn
	}
}

// NewCookieRotator creates a new cookie rotator
func NewCookieRotator(client tls_client.HttpClient, cookies *config.Cookies, interval time.Duration, opts ...RotatorOption) *CookieRotator {
	r := &CookieRotator{
		client:   client,
		cookies:  cookies,
		interval: interval,
		// stopCh será criado em Start()
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}
```

#### 2.2 Modificar Start() para criar novo canal

**Antes (linha 105-134):**
```go
func (r *CookieRotator) Start() {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return
	}
	r.running = true
	r.mu.Unlock()

	go func() {
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				newToken, err := RotateCookies(r.client, r.cookies)
				if err != nil {
					// Log error but continue
					continue
				}
				if newToken != "" {
					r.cookies.Update1PSIDTS(newToken)
				}
			case <-r.stopCh:
				return
			}
		}
	}()
}
```

**Depois:**
```go
// Start begins background cookie rotation
func (r *CookieRotator) Start() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return
	}

	// Criar novo canal em cada Start() para permitir restart
	r.stopCh = make(chan struct{})
	r.running = true

	go func() {
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				newToken, err := RotateCookies(r.client, r.cookies)
				if err != nil {
					// Reportar erro via callback se configurado
					if r.onError != nil {
						r.onError(fmt.Errorf("cookie rotation failed: %w", err))
					}
					continue
				}
				if newToken != "" {
					r.cookies.Update1PSIDTS(newToken)
				}
			case <-r.stopCh:
				return
			}
		}
	}()
}
```

#### 2.3 Adicionar import de fmt (se não existir)

Verificar se `fmt` está no bloco de imports.

#### 2.4 Atualizar chamada em client.go

**Ficheiro:** `internal/api/client.go`
**Linha:** 245

**Antes:**
```go
c.rotator = NewCookieRotator(c.httpClient, c.cookies, c.refreshInterval)
```

**Depois (opcional, se quiser logging):**
```go
c.rotator = NewCookieRotator(c.httpClient, c.cookies, c.refreshInterval,
	WithErrorCallback(func(err error) {
		// Por agora, apenas ignorar silenciosamente
		// Futuro: usar logger estruturado
		_ = err
	}),
)
```

### Testes Necessários
- [ ] Teste de Start/Stop/Start (restart funciona)
- [ ] Teste de Stop duplo (não causa panic)
- [ ] Teste de callback de erro é chamado
- [ ] Teste com `go test -race`

---

## Fase 3: Corrigir ChatSession Race Condition

### Contexto
`ChatSession` é usada concorrentemente pelo TUI (leitura de metadados) e pela goroutine de envio de mensagens (escrita de metadados).

### Ficheiro Alvo
`internal/api/session.go`

### Alterações Detalhadas

#### 3.1 Adicionar mutex ao struct

**Antes (linha 7-14):**
```go
type ChatSession struct {
	client     *GeminiClient
	model      models.Model
	metadata   []string
	lastOutput *models.ModelOutput
	gemID      string
}
```

**Depois:**
```go
type ChatSession struct {
	client     *GeminiClient
	mu         sync.RWMutex // Protege metadata, lastOutput, gemID, model
	model      models.Model
	metadata   []string
	lastOutput *models.ModelOutput
	gemID      string
}
```

#### 3.2 Adicionar import de sync

```go
import (
	"sync" // ADICIONAR

	"github.com/diogo/geminiweb/internal/models"
)
```

#### 3.3 Modificar SendMessage para usar locks

**Antes (linha 17-36):**
```go
func (s *ChatSession) SendMessage(prompt string, files []*UploadedFile) (*models.ModelOutput, error) {
	opts := &GenerateOptions{
		Model:    s.model,
		Metadata: s.metadata,
		GemID:    s.gemID,
		Files:    files,
	}

	output, err := s.client.GenerateContent(prompt, opts)
	if err != nil {
		return nil, err
	}

	// Update session state
	s.lastOutput = output
	s.updateMetadata(output)

	return output, nil
}
```

**Depois:**
```go
func (s *ChatSession) SendMessage(prompt string, files []*UploadedFile) (*models.ModelOutput, error) {
	// Ler estado atual com read lock
	s.mu.RLock()
	opts := &GenerateOptions{
		Model:    s.model,
		Metadata: copyMetadata(s.metadata), // Cópia para evitar race
		GemID:    s.gemID,
		Files:    files,
	}
	s.mu.RUnlock()

	// GenerateContent é thread-safe, não precisa de lock
	output, err := s.client.GenerateContent(prompt, opts)
	if err != nil {
		return nil, err
	}

	// Atualizar estado com write lock
	s.mu.Lock()
	s.lastOutput = output
	s.updateMetadataLocked(output)
	s.mu.Unlock()

	return output, nil
}

// copyMetadata cria uma cópia do slice de metadata
func copyMetadata(m []string) []string {
	if m == nil {
		return nil
	}
	result := make([]string, len(m))
	copy(result, m)
	return result
}
```

#### 3.4 Renomear updateMetadata para versão locked

**Antes (linha 38-51):**
```go
func (s *ChatSession) updateMetadata(output *models.ModelOutput) {
	if len(output.Metadata) > 0 {
		s.metadata = make([]string, len(output.Metadata))
		copy(s.metadata, output.Metadata)
	}

	if len(s.metadata) >= 3 {
		s.metadata[2] = output.RCID()
	} else if len(s.metadata) == 2 {
		s.metadata = append(s.metadata, output.RCID())
	}
}
```

**Depois:**
```go
// updateMetadataLocked atualiza metadata - DEVE ser chamado com s.mu.Lock() held
func (s *ChatSession) updateMetadataLocked(output *models.ModelOutput) {
	if len(output.Metadata) > 0 {
		s.metadata = make([]string, len(output.Metadata))
		copy(s.metadata, output.Metadata)
	}

	if len(s.metadata) >= 3 {
		s.metadata[2] = output.RCID()
	} else if len(s.metadata) == 2 {
		s.metadata = append(s.metadata, output.RCID())
	}
}
```

#### 3.5 Modificar SetMetadata

**Antes (linha 53-56):**
```go
func (s *ChatSession) SetMetadata(cid, rid, rcid string) {
	s.metadata = []string{cid, rid, rcid}
}
```

**Depois:**
```go
func (s *ChatSession) SetMetadata(cid, rid, rcid string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metadata = []string{cid, rid, rcid}
}
```

#### 3.6 Modificar GetMetadata para retornar cópia

**Antes (linha 58-61):**
```go
func (s *ChatSession) GetMetadata() []string {
	return s.metadata
}
```

**Depois:**
```go
func (s *ChatSession) GetMetadata() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return copyMetadata(s.metadata)
}
```

#### 3.7 Modificar CID, RID, RCID

**Antes (linhas 63-85):**
```go
func (s *ChatSession) CID() string {
	if len(s.metadata) > 0 {
		return s.metadata[0]
	}
	return ""
}

func (s *ChatSession) RID() string {
	if len(s.metadata) > 1 {
		return s.metadata[1]
	}
	return ""
}

func (s *ChatSession) RCID() string {
	if len(s.metadata) > 2 {
		return s.metadata[2]
	}
	return ""
}
```

**Depois:**
```go
func (s *ChatSession) CID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.metadata) > 0 {
		return s.metadata[0]
	}
	return ""
}

func (s *ChatSession) RID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.metadata) > 1 {
		return s.metadata[1]
	}
	return ""
}

func (s *ChatSession) RCID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.metadata) > 2 {
		return s.metadata[2]
	}
	return ""
}
```

#### 3.8 Modificar GetModel e SetModel

**Antes (linhas 87-95):**
```go
func (s *ChatSession) GetModel() models.Model {
	return s.model
}

func (s *ChatSession) SetModel(model models.Model) {
	s.model = model
}
```

**Depois:**
```go
func (s *ChatSession) GetModel() models.Model {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.model
}

func (s *ChatSession) SetModel(model models.Model) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.model = model
}
```

#### 3.9 Modificar LastOutput

**Antes (linhas 97-100):**
```go
func (s *ChatSession) LastOutput() *models.ModelOutput {
	return s.lastOutput
}
```

**Depois:**
```go
func (s *ChatSession) LastOutput() *models.ModelOutput {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastOutput
}
```

#### 3.10 Modificar ChooseCandidate

**Antes (linhas 102-114):**
```go
func (s *ChatSession) ChooseCandidate(index int) error {
	if s.lastOutput == nil {
		return nil
	}
	if index >= len(s.lastOutput.Candidates) {
		return nil
	}

	s.lastOutput.Chosen = index
	s.updateMetadata(s.lastOutput)
	return nil
}
```

**Depois:**
```go
func (s *ChatSession) ChooseCandidate(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lastOutput == nil {
		return nil
	}
	if index >= len(s.lastOutput.Candidates) {
		return nil
	}

	s.lastOutput.Chosen = index
	s.updateMetadataLocked(s.lastOutput)
	return nil
}
```

#### 3.11 Modificar SetGem e GetGemID

**Antes (linhas 116-124):**
```go
func (s *ChatSession) SetGem(gemID string) {
	s.gemID = gemID
}

func (s *ChatSession) GetGemID() string {
	return s.gemID
}
```

**Depois:**
```go
func (s *ChatSession) SetGem(gemID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gemID = gemID
}

func (s *ChatSession) GetGemID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.gemID
}
```

### Testes Necessários
- [ ] Teste de race: múltiplas goroutines a chamar SendMessage
- [ ] Teste de race: leitura de metadata enquanto SendMessage executa
- [ ] Verificar com `go test -race ./internal/api/...`

---

## Fase 4: Corrigir Erros Ignorados em batch.go

### Ficheiro Alvo
`internal/api/batch.go`

### Alteração

**Antes (linha 68):**
```go
u, _ := url.Parse(models.EndpointBatchExec)
```

**Depois:**
```go
u, err := url.Parse(models.EndpointBatchExec)
if err != nil {
	return nil, fmt.Errorf("failed to parse batch endpoint URL: %w", err)
}
```

### Testes Necessários
- [ ] Teste unitário existente deve continuar a passar
- [ ] Verificar que endpoint constante é válido

---

## Fase 5: Corrigir Erros io.ReadAll em upload.go

### Ficheiro Alvo
`internal/api/upload.go`

### Alterações

#### 5.1 Linha ~183 (em UploadImage ou similar)

**Antes:**
```go
bodyBytes, _ := io.ReadAll(resp.Body)
return nil, apierrors.NewUploadErrorWithStatus(fileName, resp.StatusCode, string(bodyBytes))
```

**Depois:**
```go
bodyBytes, readErr := io.ReadAll(resp.Body)
bodyStr := "(unable to read response body)"
if readErr == nil {
	bodyStr = string(bodyBytes)
}
return nil, apierrors.NewUploadErrorWithStatus(fileName, resp.StatusCode, bodyStr)
```

#### 5.2 Linha ~340 (em UploadFile ou similar)

Aplicar a mesma correção.

### Testes Necessários
- [ ] Teste de upload com resposta de erro
- [ ] Mock de body que falha na leitura

---

## Fase 6: Corrigir Cookie Store Leak em browser.go

### Ficheiro Alvo
`internal/browser/browser.go`

### Alteração na função extractFromBrowser

**Conceito:** Usar defer para garantir cleanup de todos os stores.

**Antes (aproximadamente linhas 120-153):**
```go
func extractFromBrowser(ctx context.Context, browserType SupportedBrowser) (*ExtractResult, error) {
	stores := kooky.FindAllCookieStores()

	var matchingStores []kooky.CookieStore
	for _, store := range stores {
		// ... filtrar stores ...
		if matches {
			matchingStores = append(matchingStores, store)
		} else {
			_ = store.Close()
		}
	}

	// ... usar matchingStores ...

	for _, store := range matchingStores {
		result, err := extractCookiesFromStore(ctx, store, browserName, store.Profile())
		_ = store.Close()
		if err == nil {
			// Close remaining stores
			for _, s := range matchingStores {
				_ = s.Close()  // BUG: double close!
			}
			return result, nil
		}
		lastErr = err
	}

	// BUG: Se chegarmos aqui, stores podem não estar fechados
	return nil, lastErr
}
```

**Depois:**
```go
func extractFromBrowser(ctx context.Context, browserType SupportedBrowser) (*ExtractResult, error) {
	stores := kooky.FindAllCookieStores()

	var matchingStores []kooky.CookieStore
	for _, store := range stores {
		// ... filtrar stores ...
		if matches {
			matchingStores = append(matchingStores, store)
		} else {
			_ = store.Close()
		}
	}

	// GARANTIR CLEANUP de todos os matching stores
	closedStores := make(map[int]bool)
	defer func() {
		for i, s := range matchingStores {
			if !closedStores[i] {
				_ = s.Close()
			}
		}
	}()

	var lastErr error
	for i, store := range matchingStores {
		result, err := extractCookiesFromStore(ctx, store, browserName, store.Profile())
		_ = store.Close()
		closedStores[i] = true

		if err == nil {
			return result, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("no cookies found in any %s profile", browserType)
}
```

### Testes Necessários
- [ ] Teste com múltiplos profiles
- [ ] Verificar que stores são fechados mesmo em erro

---

## Fase 7: Melhorar Error Handling em gems.go

### Ficheiro Alvo
`internal/api/gems.go`

### Alteração em FetchGems

**Antes (linhas 42-67):**
```go
func (c *GeminiClient) FetchGems(includeHidden bool) (*models.GemJar, error) {
	// ... setup ...

	responses, err := c.BatchExecute(requests)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch gems: %w", err)
	}

	jar := make(models.GemJar)

	for _, resp := range responses {
		if resp.Error != nil || resp.Data == "" {
			continue  // PROBLEMA: erro ignorado
		}

		predefined := resp.Identifier == "system"
		gems, err := parseGemsResponse(resp.Data, predefined)
		if err != nil {
			continue  // PROBLEMA: erro ignorado
		}

		for _, gem := range gems {
			jar[gem.ID] = gem
		}
	}

	// ... cache update ...
	return &jar, nil
}
```

**Depois:**
```go
func (c *GeminiClient) FetchGems(includeHidden bool) (*models.GemJar, error) {
	// ... setup ...

	responses, err := c.BatchExecute(requests)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch gems: %w", err)
	}

	jar := make(models.GemJar)
	var fetchErrors []string

	for _, resp := range responses {
		if resp.Error != nil {
			fetchErrors = append(fetchErrors, fmt.Sprintf("%s: %v", resp.Identifier, resp.Error))
			continue
		}
		if resp.Data == "" {
			fetchErrors = append(fetchErrors, fmt.Sprintf("%s: empty response", resp.Identifier))
			continue
		}

		predefined := resp.Identifier == "system"
		gems, err := parseGemsResponse(resp.Data, predefined)
		if err != nil {
			fetchErrors = append(fetchErrors, fmt.Sprintf("%s: parse error: %v", resp.Identifier, err))
			continue
		}

		for _, gem := range gems {
			jar[gem.ID] = gem
		}
	}

	// Se não obtivemos nenhum gem e houve erros, retornar erro
	if len(jar) == 0 && len(fetchErrors) > 0 {
		return nil, fmt.Errorf("failed to fetch gems: %s", strings.Join(fetchErrors, "; "))
	}

	// Atualizar cache
	c.mu.Lock()
	c.gems = &jar
	c.mu.Unlock()

	return &jar, nil
}
```

### Adicionar import de strings

```go
import (
	// ... outros imports ...
	"strings"
)
```

### Testes Necessários
- [ ] Teste com resposta parcial (um tipo funciona, outro falha)
- [ ] Teste com ambos a falhar
- [ ] Teste com ambos a funcionar

---

## Fase 8: Corrigir Spinner Double-Close

### Ficheiro Alvo
`internal/commands/query.go`

### Alterações

#### 8.1 Adicionar flag stopped ao struct

**Antes (aproximadamente linhas 67-72):**
```go
type spinner struct {
	message string
	stop    chan struct{}
	done    chan struct{}
	mu      sync.Mutex
	frame   int
}
```

**Depois:**
```go
type spinner struct {
	message string
	stop    chan struct{}
	done    chan struct{}
	mu      sync.Mutex
	frame   int
	stopped bool // Flag para evitar double-close
}
```

#### 8.2 Adicionar método interno stopOnce

**Adicionar antes de stopWithSuccess:**
```go
// stopOnce fecha o canal stop de forma segura (apenas uma vez)
func (s *spinner) stopOnce() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.stopped {
		close(s.stop)
		s.stopped = true
	}
}
```

#### 8.3 Modificar stopWithSuccess

**Antes:**
```go
func (s *spinner) stopWithSuccess(message string) {
	close(s.stop)
	<-s.done
	// ... resto ...
}
```

**Depois:**
```go
func (s *spinner) stopWithSuccess(message string) {
	s.stopOnce()
	<-s.done
	// ... resto ...
}
```

#### 8.4 Modificar stopWithError

**Antes:**
```go
func (s *spinner) stopWithError() {
	close(s.stop)
	<-s.done
}
```

**Depois:**
```go
func (s *spinner) stopWithError() {
	s.stopOnce()
	<-s.done
}
```

### Testes Necessários
- [ ] Teste de chamada dupla a stopWithError
- [ ] Teste de stopWithSuccess seguido de stopWithError

---

## Fase 9: Refatorar Lock em RefreshFromBrowser

### Ficheiro Alvo
`internal/api/client.go`

### Contexto
`RefreshFromBrowser` mantém o mutex durante operações de rede longas, o que pode bloquear outras goroutines.

### Alteração (linhas ~400-450)

**Antes:**
```go
func (c *GeminiClient) RefreshFromBrowser() (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.browserRefresh {
		return false, fmt.Errorf("browser refresh is not enabled")
	}

	if time.Since(c.lastBrowserRefresh) < c.browserRefreshMinWait {
		return false, fmt.Errorf("browser refresh attempted too recently")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// ... HTTP request sob lock ...

	result, err := browser.ExtractGeminiCookies(ctx, c.browserRefreshType)
	// ...
}
```

**Depois:**
```go
func (c *GeminiClient) RefreshFromBrowser() (bool, error) {
	// FASE 1: Verificar precondições com read lock
	c.mu.RLock()
	if !c.browserRefresh {
		c.mu.RUnlock()
		return false, fmt.Errorf("browser refresh is not enabled")
	}
	if time.Since(c.lastBrowserRefresh) < c.browserRefreshMinWait {
		waitTime := c.browserRefreshMinWait - time.Since(c.lastBrowserRefresh)
		c.mu.RUnlock()
		return false, fmt.Errorf("browser refresh attempted too recently, wait %v", waitTime)
	}
	browserType := c.browserRefreshType
	extractor := c.browserExtractor
	c.mu.RUnlock()

	// FASE 2: Operações de rede SEM lock
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var result *browser.ExtractResult
	var err error
	if extractor != nil {
		result, err = extractor.ExtractGeminiCookies(ctx, browserType)
	} else {
		result, err = browser.ExtractGeminiCookies(ctx, browserType)
	}

	if err != nil {
		// Atualizar timestamp mesmo em caso de erro para rate limiting
		c.mu.Lock()
		c.lastBrowserRefresh = time.Now()
		c.mu.Unlock()
		return false, fmt.Errorf("failed to extract cookies from browser: %w", err)
	}

	// FASE 3: Atualizar estado com write lock
	c.mu.Lock()
	defer c.mu.Unlock()

	// Re-verificar rate limit (double-check locking)
	if time.Since(c.lastBrowserRefresh) < c.browserRefreshMinWait {
		return false, fmt.Errorf("browser refresh completed by another goroutine")
	}

	// Atualizar cookies usando método thread-safe
	c.cookies.SetBoth(result.Cookies.Secure1PSID, result.Cookies.Secure1PSIDTS)
	c.lastBrowserRefresh = time.Now()

	// Guardar em disco
	if err := config.SaveCookies(c.cookies); err != nil {
		fmt.Printf("Warning: failed to save refreshed cookies to disk: %v\n", err)
	}

	// Obter novo access token
	// NOTA: Isto ainda faz HTTP request sob lock, mas é mais curto
	// Para otimização máxima, poderia ser movido para fora também
	token, err := GetAccessToken(c.httpClient, c.cookies)
	if err != nil {
		return false, fmt.Errorf("failed to get access token with new cookies: %w", err)
	}
	c.accessToken = token

	return true, nil
}
```

### Testes Necessários
- [ ] Teste de race com múltiplas goroutines a chamar RefreshFromBrowser
- [ ] Teste de rate limiting funciona corretamente
- [ ] Verificar com `go test -race`

---

## Fase 10: Correções Menores

### 10.1 filepath.Abs em download.go

**Ficheiro:** `internal/api/download.go`
**Linha:** ~112

**Antes:**
```go
absPath, _ := filepath.Abs(destPath)
return absPath, nil
```

**Depois:**
```go
absPath, err := filepath.Abs(destPath)
if err != nil {
	// Se não conseguir path absoluto, retornar o relativo
	return destPath, nil
}
return absPath, nil
```

### 10.2 Atualizar uso de cookies nos ficheiros API

Após Fase 1, atualizar:

**generate.go (linhas 114-118):**
```go
// Antes:
req.AddCookie(&http.Cookie{Name: "__Secure-1PSID", Value: cookies.Secure1PSID})
if cookies.Secure1PSIDTS != "" {
	req.AddCookie(&http.Cookie{Name: "__Secure-1PSIDTS", Value: cookies.Secure1PSIDTS})
}

// Depois:
psid, psidts := cookies.Snapshot()
req.AddCookie(&http.Cookie{Name: "__Secure-1PSID", Value: psid})
if psidts != "" {
	req.AddCookie(&http.Cookie{Name: "__Secure-1PSIDTS", Value: psidts})
}
```

Aplicar padrão similar em:
- `batch.go` (linhas 97-101)
- `token.go` (linhas 28-32)
- `rotate.go` (linhas 48-51)

---

## Ordem de Execução

A ordem de implementação deve respeitar dependências:

```
Fase 1 (cookies.go)
    │
    ├──► Fase 2 (rotate.go) ──► Fase 9 (client.go RefreshFromBrowser)
    │
    └──► Fase 10.2 (uso de cookies em generate.go, batch.go, etc.)

Fase 3 (session.go) - Independente

Fase 4 (batch.go url.Parse) - Independente

Fase 5 (upload.go io.ReadAll) - Independente

Fase 6 (browser.go store leak) - Independente

Fase 7 (gems.go error handling) - Independente

Fase 8 (query.go spinner) - Independente

Fase 10.1 (download.go filepath.Abs) - Independente
```

**Sequência Recomendada:**
1. Fase 1 (base para outras)
2. Fase 4, 5, 6, 8, 10.1 (independentes, podem ser paralelas)
3. Fase 2 (depende de Fase 1)
4. Fase 10.2 (depende de Fase 1)
5. Fase 3 (independente mas relacionada)
6. Fase 7 (independente)
7. Fase 9 (depende de Fase 1 e 2)

---

## Validação Final

### Testes de Race Condition
```bash
go test -race ./internal/api/...
go test -race ./internal/config/...
go test -race ./internal/browser/...
go test -race ./internal/commands/...
```

### Testes de Integração
```bash
make test
```

### Verificação de Build
```bash
make build
```

### Testes Manuais
- [ ] Chat TUI funciona sem erros
- [ ] Gems list/create/delete funciona
- [ ] Cookie rotation funciona em background
- [ ] Browser refresh funciona quando cookies expiram

---

## Riscos e Mitigações

| Risco | Probabilidade | Impacto | Mitigação |
|-------|--------------|---------|-----------|
| Deadlock introduzido | Média | Alto | Testar com -race, code review |
| Performance degradada | Baixa | Médio | Usar RLock onde possível |
| Quebra de API | Baixa | Alto | Manter assinaturas de métodos |
| Regressões | Média | Médio | Testes existentes + novos |

---

## Métricas de Sucesso

- [ ] Zero race conditions detectadas com `go test -race`
- [ ] Todos os testes existentes passam
- [ ] Novos testes de concorrência passam
- [ ] Nenhum panic em uso normal
- [ ] Build sem warnings

---

## Notas de Implementação

1. **Commits atómicos:** Cada fase deve ser um commit separado
2. **Testes primeiro:** Escrever testes de race antes de corrigir (TDD)
3. **Code review:** Cada fase deve ser revista antes de merge
4. **Rollback plan:** Manter branch de backup antes de cada fase

---

*Documento criado em: 2025-12-12*
*Última atualização: 2025-12-12*
