# Plano de Implementação: Gemini Gems

## Visão Geral

Este documento detalha o plano de implementação para adicionar suporte completo a **Gemini Gems** no projeto `geminiweb-go`. Gems são personas customizadas armazenadas nos servidores do Google, diferentemente das "Personas" locais já existentes no projeto.

### Diferença entre Personas (local) e Gems (servidor)

| Aspecto | Personas (atual) | Gems (a implementar) |
|---------|------------------|----------------------|
| Armazenamento | Local (`~/.geminiweb/personas.json`) | Servidor Google |
| System Prompt | Concatenado no prompt pelo cliente | Aplicado pelo servidor |
| Sincronização | Não sincroniza | Sincroniza com conta Google |
| CRUD | Via arquivos locais | Via API RPC |
| Compartilhamento | Manual (copiar arquivo) | Via conta Google |

---

## Arquitetura Proposta

```
internal/
├── api/
│   ├── gems.go              # NOVO: Operações CRUD de Gems
│   ├── gems_test.go         # NOVO: Testes para Gems
│   ├── batch.go             # NOVO: Sistema de batch RPC
│   ├── batch_test.go        # NOVO: Testes para batch RPC
│   ├── generate.go          # MODIFICAR: Adicionar suporte a gem_id no payload
│   ├── session.go           # MODIFICAR: Adicionar campo gem em ChatSession
│   └── client.go            # MODIFICAR: Adicionar campo gems e métodos
├── models/
│   ├── gems.go              # NOVO: Tipos Gem e GemJar
│   └── constants.go         # MODIFICAR: Adicionar RPC IDs
├── errors/
│   └── errors.go            # MODIFICAR: Adicionar GemError
└── commands/
    ├── gems.go              # NOVO: Comandos CLI para gems
    └── query.go             # MODIFICAR: Adicionar --gem flag
```

---

## Fase 1: Tipos e Constantes

### 1.1 Criar `internal/models/gems.go`

```go
package models

import "strings"

// Gem representa uma persona customizada armazenada no servidor Google
type Gem struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description,omitempty"`
    Prompt      string `json:"prompt,omitempty"`
    Predefined  bool   `json:"predefined"` // true = gem de sistema, false = custom
}

// GemJar é uma coleção de Gems indexada por ID
type GemJar map[string]*Gem

// Get retorna um Gem por ID ou nome
func (j GemJar) Get(id, name string) *Gem {
    if id != "" {
        if gem, ok := j[id]; ok {
            return gem
        }
    }
    if name != "" {
        for _, gem := range j {
            if gem.Name == name {
                return gem
            }
        }
    }
    return nil
}

// Filter filtra gems por critérios
func (j GemJar) Filter(predefined *bool, nameContains string) GemJar {
    result := make(GemJar)
    for id, gem := range j {
        if predefined != nil && gem.Predefined != *predefined {
            continue
        }
        if nameContains != "" && !strings.Contains(strings.ToLower(gem.Name), strings.ToLower(nameContains)) {
            continue
        }
        result[id] = gem
    }
    return result
}

// Custom retorna apenas gems customizados (não predefinidos)
func (j GemJar) Custom() GemJar {
    predefined := false
    return j.Filter(&predefined, "")
}

// System retorna apenas gems de sistema (predefinidos)
func (j GemJar) System() GemJar {
    predefined := true
    return j.Filter(&predefined, "")
}

// Values retorna todos os gems como slice
func (j GemJar) Values() []*Gem {
    gems := make([]*Gem, 0, len(j))
    for _, gem := range j {
        gems = append(gems, gem)
    }
    return gems
}

// Len retorna o número de gems
func (j GemJar) Len() int {
    return len(j)
}
```

### 1.2 Adicionar constantes em `internal/models/constants.go`

```go
// RPC IDs para operações de Gems (batch execute)
const (
    RPCListGems   = "CNgdBe"
    RPCCreateGem  = "oMH3Zd"
    RPCUpdateGem  = "kHv0Vd"
    RPCDeleteGem  = "UXcSJb"
)

// Parâmetros para ListGems
const (
    ListGemsNormal        = 3 // Gems normais (visíveis na UI)
    ListGemsIncludeHidden = 4 // Incluir gems ocultos de sistema
    ListGemsCustom        = 2 // Gems customizados do usuário
)
```

### 1.3 Testes para `internal/models/gems_test.go`

```go
package models

import "testing"

func TestGemJarGet(t *testing.T) {
    jar := make(GemJar)
    jar["abc123"] = &Gem{ID: "abc123", Name: "Test Gem", Predefined: false}
    jar["def456"] = &Gem{ID: "def456", Name: "System Gem", Predefined: true}

    tests := []struct {
        name     string
        id       string
        gemName  string
        wantID   string
        wantNil  bool
    }{
        {"by ID", "abc123", "", "abc123", false},
        {"by name", "", "Test Gem", "abc123", false},
        {"by name case sensitive", "", "test gem", "", true},
        {"not found", "xyz", "Unknown", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            gem := jar.Get(tt.id, tt.gemName)
            if tt.wantNil && gem != nil {
                t.Error("Expected nil, got gem")
            }
            if !tt.wantNil && (gem == nil || gem.ID != tt.wantID) {
                t.Errorf("Expected ID %s, got %v", tt.wantID, gem)
            }
        })
    }
}

func TestGemJarFilter(t *testing.T) {
    jar := make(GemJar)
    jar["1"] = &Gem{ID: "1", Name: "Code Helper", Predefined: true}
    jar["2"] = &Gem{ID: "2", Name: "My Coder", Predefined: false}
    jar["3"] = &Gem{ID: "3", Name: "Writer", Predefined: false}

    t.Run("filter by predefined true", func(t *testing.T) {
        predefined := true
        result := jar.Filter(&predefined, "")
        if len(result) != 1 {
            t.Errorf("Expected 1, got %d", len(result))
        }
    })

    t.Run("filter by name contains", func(t *testing.T) {
        result := jar.Filter(nil, "code")
        if len(result) != 2 {
            t.Errorf("Expected 2, got %d", len(result))
        }
    })

    t.Run("Custom helper", func(t *testing.T) {
        custom := jar.Custom()
        if len(custom) != 2 {
            t.Errorf("Expected 2 custom gems, got %d", len(custom))
        }
    })

    t.Run("System helper", func(t *testing.T) {
        system := jar.System()
        if len(system) != 1 {
            t.Errorf("Expected 1 system gem, got %d", len(system))
        }
    })
}
```

---

## Fase 2: Sistema de Batch RPC

O sistema batch é necessário pois todas as operações de Gems usam o endpoint `/batchexecute` com um protocolo RPC específico do Google.

### 2.1 Criar `internal/api/batch.go`

```go
package api

import (
    "encoding/json"
    "fmt"
    "net/url"
    "strings"

    http "github.com/bogdanfinn/fhttp"
    "github.com/tidwall/gjson"

    apierrors "github.com/diogo/geminiweb/internal/errors"
    "github.com/diogo/geminiweb/internal/models"
)

// RPCData representa uma chamada RPC individual para batch execute
type RPCData struct {
    RPCID      string // ID do método RPC (ex: "CNgdBe" para listar gems)
    Payload    string // JSON payload como string
    Identifier string // Identificador para match na resposta
}

// Serialize converte RPCData para o formato esperado pela API Google
// Formato: [rpcid, payload, null, identifier]
func (r *RPCData) Serialize() []interface{} {
    return []interface{}{r.RPCID, r.Payload, nil, r.Identifier}
}

// BatchResponse representa uma resposta individual do batch execute
type BatchResponse struct {
    Identifier string // Identifier que foi enviado na requisição
    Data       string // JSON string com os dados da resposta
    Error      error  // Erro se houver falha nesta operação específica
}

// BatchExecute executa múltiplas chamadas RPC em uma única requisição HTTP
// Este é o método central para todas as operações de Gems
func (c *GeminiClient) BatchExecute(requests []RPCData) ([]BatchResponse, error) {
    if c.IsClosed() {
        return nil, fmt.Errorf("client is closed")
    }

    if len(requests) == 0 {
        return nil, fmt.Errorf("no requests provided")
    }

    // Construir array de requisições serializadas
    // Formato final: [[rpc1], [rpc2], ...]
    var serialized []interface{}
    for _, req := range requests {
        serialized = append(serialized, req.Serialize())
    }

    payload, err := json.Marshal(serialized)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal batch payload: %w", err)
    }

    // Criar form data (igual ao generate)
    form := url.Values{}
    form.Set("at", c.GetAccessToken())
    form.Set("f.req", string(payload))

    req, err := http.NewRequest(
        http.MethodPost,
        models.EndpointBatchExec,
        strings.NewReader(form.Encode()),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    // Usar mesmos headers do generate
    for key, value := range models.DefaultHeaders() {
        req.Header.Set(key, value)
    }

    // Set cookies
    cookies := c.GetCookies()
    req.AddCookie(&http.Cookie{Name: "__Secure-1PSID", Value: cookies.Secure1PSID})
    if cookies.Secure1PSIDTS != "" {
        req.AddCookie(&http.Cookie{Name: "__Secure-1PSIDTS", Value: cookies.Secure1PSIDTS})
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, apierrors.NewNetworkErrorWithEndpoint("batch execute", models.EndpointBatchExec, err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return nil, apierrors.NewAPIError(resp.StatusCode, models.EndpointBatchExec, "batch execute failed")
    }

    // Ler body completo
    body := make([]byte, 0, 65536)
    buf := make([]byte, 4096)
    for {
        n, err := resp.Body.Read(buf)
        if n > 0 {
            body = append(body, buf[:n]...)
        }
        if err != nil {
            break
        }
    }

    return parseBatchResponse(body, requests)
}

// parseBatchResponse analisa a resposta do batch execute
// Formato da resposta:
// )]}'
// [["wrb.fr","RPCID","data_json",null,null,null,"identifier"],...]
func parseBatchResponse(body []byte, requests []RPCData) ([]BatchResponse, error) {
    lines := strings.Split(string(body), "\n")
    var jsonLine string

    // Pular linhas de lixo (como ")]}'" ou vazias) e encontrar JSON válido
    for _, line := range lines {
        line = strings.TrimSpace(line)
        if line == "" || line == ")]}" || line == ")]}'" {
            continue
        }
        if gjson.Valid(line) {
            jsonLine = line
            break
        }
    }

    if jsonLine == "" {
        return nil, apierrors.NewParseError("no valid JSON in batch response", "")
    }

    parsed := gjson.Parse(jsonLine)

    // Criar respostas iniciais
    responses := make([]BatchResponse, len(requests))
    for i, req := range requests {
        responses[i] = BatchResponse{Identifier: req.Identifier}
    }

    // Iterar sobre as partes da resposta e fazer match por identifier
    parsed.ForEach(func(_, part gjson.Result) bool {
        if !part.IsArray() {
            return true
        }

        arr := part.Array()
        if len(arr) < 3 {
            return true
        }

        // Extrair dados (posição 2 contém o JSON string)
        data := ""
        if arr[2].Type == gjson.String {
            data = arr[2].String()
        }

        // Encontrar identifier (procurar nas últimas posições)
        var identifier string
        for i := len(arr) - 1; i >= 3; i-- {
            if arr[i].Type == gjson.String && arr[i].String() != "" {
                candidateID := arr[i].String()
                // Verificar se é um identifier conhecido
                for _, req := range requests {
                    if candidateID == req.Identifier {
                        identifier = candidateID
                        break
                    }
                }
                if identifier != "" {
                    break
                }
            }
        }

        // Atualizar resposta correspondente
        if identifier != "" {
            for i, resp := range responses {
                if resp.Identifier == identifier {
                    responses[i].Data = data
                    break
                }
            }
        }

        return true
    })

    return responses, nil
}
```

### 2.2 Testes para `internal/api/batch_test.go`

```go
package api

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestBatchExecute(t *testing.T) {
    // Mock server que retorna resposta válida
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        response := `)]}'
[["wrb.fr","CNgdBe","[\"test_data\"]",null,null,null,"test_id"]]`
        w.Write([]byte(response))
    }))
    defer server.Close()

    // Este teste requer mock do client - implementar após criar helpers de teste
    t.Skip("Requires mock client implementation")
}

func TestParseBatchResponse(t *testing.T) {
    requests := []RPCData{
        {RPCID: "CNgdBe", Payload: "[]", Identifier: "system"},
        {RPCID: "CNgdBe", Payload: "[]", Identifier: "custom"},
    }

    body := []byte(`)]}'
[["wrb.fr","CNgdBe","[\"system_data\"]",null,null,null,"system"],["wrb.fr","CNgdBe","[\"custom_data\"]",null,null,null,"custom"]]`)

    responses, err := parseBatchResponse(body, requests)
    if err != nil {
        t.Fatalf("parseBatchResponse failed: %v", err)
    }

    if len(responses) != 2 {
        t.Fatalf("Expected 2 responses, got %d", len(responses))
    }

    // Verificar que cada response tem o data correto
    for _, resp := range responses {
        if resp.Identifier == "system" && resp.Data != "[\"system_data\"]" {
            t.Errorf("System data mismatch: got %s", resp.Data)
        }
        if resp.Identifier == "custom" && resp.Data != "[\"custom_data\"]" {
            t.Errorf("Custom data mismatch: got %s", resp.Data)
        }
    }
}

func TestRPCDataSerialize(t *testing.T) {
    rpc := RPCData{
        RPCID:      "CNgdBe",
        Payload:    "[3]",
        Identifier: "test",
    }

    serialized := rpc.Serialize()

    if len(serialized) != 4 {
        t.Fatalf("Expected 4 elements, got %d", len(serialized))
    }

    if serialized[0] != "CNgdBe" {
        t.Errorf("Expected RPCID 'CNgdBe', got %v", serialized[0])
    }
    if serialized[1] != "[3]" {
        t.Errorf("Expected payload '[3]', got %v", serialized[1])
    }
    if serialized[2] != nil {
        t.Errorf("Expected nil at position 2, got %v", serialized[2])
    }
    if serialized[3] != "test" {
        t.Errorf("Expected identifier 'test', got %v", serialized[3])
    }
}
```

---

## Fase 3: Operações CRUD de Gems

### 3.1 Criar `internal/api/gems.go`

```go
package api

import (
    "encoding/json"
    "fmt"

    "github.com/tidwall/gjson"

    "github.com/diogo/geminiweb/internal/models"
)

// FetchGems carrega todos os gems do servidor Google
// includeHidden: se true, inclui gems de sistema ocultos (não visíveis na UI web)
func (c *GeminiClient) FetchGems(includeHidden bool) (*models.GemJar, error) {
    // Determinar parâmetro para gems de sistema
    systemParam := models.ListGemsNormal
    if includeHidden {
        systemParam = models.ListGemsIncludeHidden
    }

    // Duas requisições RPC em batch:
    // 1. Gems de sistema (predefinidos pelo Google)
    // 2. Gems customizados (criados pelo usuário)
    requests := []RPCData{
        {
            RPCID:      models.RPCListGems,
            Payload:    fmt.Sprintf("[%d]", systemParam),
            Identifier: "system",
        },
        {
            RPCID:      models.RPCListGems,
            Payload:    fmt.Sprintf("[%d]", models.ListGemsCustom),
            Identifier: "custom",
        },
    }

    responses, err := c.BatchExecute(requests)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch gems: %w", err)
    }

    jar := make(models.GemJar)

    for _, resp := range responses {
        if resp.Error != nil || resp.Data == "" {
            continue
        }

        predefined := resp.Identifier == "system"
        gems, err := parseGemsResponse(resp.Data, predefined)
        if err != nil {
            // Log mas não falha - pode ter dados parciais
            continue
        }

        for _, gem := range gems {
            jar[gem.ID] = gem
        }
    }

    // Atualizar cache no client
    c.mu.Lock()
    c.gems = &jar
    c.mu.Unlock()

    return &jar, nil
}

// parseGemsResponse analisa a resposta JSON de listagem de gems
func parseGemsResponse(data string, predefined bool) ([]*models.Gem, error) {
    parsed := gjson.Parse(data)
    if !parsed.IsArray() {
        return nil, fmt.Errorf("invalid gems response: not an array")
    }

    // Gems estão na posição [2] do array de resposta
    gemsArray := parsed.Get("2")
    if !gemsArray.Exists() || !gemsArray.IsArray() {
        // Pode não ter gems - não é erro
        return nil, nil
    }

    var gems []*models.Gem
    gemsArray.ForEach(func(_, gemData gjson.Result) bool {
        gem := parseGemData(gemData, predefined)
        if gem != nil {
            gems = append(gems, gem)
        }
        return true
    })

    return gems, nil
}

// parseGemData extrai dados de um gem individual da resposta
// Estrutura do gem no array:
// [0] = ID (string)
// [1][0] = Nome (string)
// [1][1] = Descrição (string)
// [2][0] = Prompt (string, pode não existir)
func parseGemData(data gjson.Result, predefined bool) *models.Gem {
    id := data.Get("0").String()
    if id == "" {
        return nil
    }

    name := data.Get("1.0").String()
    description := data.Get("1.1").String()

    // Prompt pode não existir (posição [2] pode ser null)
    prompt := ""
    promptData := data.Get("2.0")
    if promptData.Exists() {
        prompt = promptData.String()
    }

    return &models.Gem{
        ID:          id,
        Name:        name,
        Description: description,
        Prompt:      prompt,
        Predefined:  predefined,
    }
}

// CreateGem cria um novo gem customizado no servidor
func (c *GeminiClient) CreateGem(name, prompt, description string) (*models.Gem, error) {
    // Payload estruturado com padding específico exigido pela API
    // Formato: [[name, description, prompt, null x5, 0, null, 1, null x3, []]]
    inner := []interface{}{
        name,             // 0: nome
        description,      // 1: descrição
        prompt,           // 2: system prompt
        nil,              // 3
        nil,              // 4
        nil,              // 5
        nil,              // 6
        nil,              // 7
        0,                // 8: flag
        nil,              // 9
        1,                // 10: flag
        nil,              // 11
        nil,              // 12
        nil,              // 13
        []interface{}{},  // 14: array vazio
    }

    payload, err := json.Marshal([]interface{}{inner})
    if err != nil {
        return nil, fmt.Errorf("failed to marshal create payload: %w", err)
    }

    requests := []RPCData{
        {
            RPCID:      models.RPCCreateGem,
            Payload:    string(payload),
            Identifier: "create",
        },
    }

    responses, err := c.BatchExecute(requests)
    if err != nil {
        return nil, fmt.Errorf("failed to create gem: %w", err)
    }

    if len(responses) == 0 || responses[0].Error != nil {
        return nil, fmt.Errorf("failed to create gem: no valid response")
    }

    // Extrair ID do gem criado (posição [0] do response data)
    respData := gjson.Parse(responses[0].Data)
    gemID := respData.Get("0").String()
    if gemID == "" {
        return nil, fmt.Errorf("failed to create gem: no ID in response")
    }

    gem := &models.Gem{
        ID:          gemID,
        Name:        name,
        Description: description,
        Prompt:      prompt,
        Predefined:  false,
    }

    // Atualizar cache
    c.mu.Lock()
    if c.gems != nil {
        (*c.gems)[gemID] = gem
    }
    c.mu.Unlock()

    return gem, nil
}

// UpdateGem atualiza um gem existente
// IMPORTANTE: Deve fornecer todos os campos, mesmo que só queira atualizar um
func (c *GeminiClient) UpdateGem(gemID, name, prompt, description string) (*models.Gem, error) {
    // Payload similar ao create, mas com gem_id na frente e um campo extra
    // Formato: [gem_id, [name, description, prompt, null x5, 0, null, 1, null x3, [], 0]]
    inner := []interface{}{
        name,             // 0
        description,      // 1
        prompt,           // 2
        nil,              // 3
        nil,              // 4
        nil,              // 5
        nil,              // 6
        nil,              // 7
        0,                // 8
        nil,              // 9
        1,                // 10
        nil,              // 11
        nil,              // 12
        nil,              // 13
        []interface{}{},  // 14
        0,                // 15: flag extra (diferencia de create)
    }

    payload, err := json.Marshal([]interface{}{gemID, inner})
    if err != nil {
        return nil, fmt.Errorf("failed to marshal update payload: %w", err)
    }

    requests := []RPCData{
        {
            RPCID:      models.RPCUpdateGem,
            Payload:    string(payload),
            Identifier: "update",
        },
    }

    _, err = c.BatchExecute(requests)
    if err != nil {
        return nil, fmt.Errorf("failed to update gem: %w", err)
    }

    gem := &models.Gem{
        ID:          gemID,
        Name:        name,
        Description: description,
        Prompt:      prompt,
        Predefined:  false,
    }

    // Atualizar cache
    c.mu.Lock()
    if c.gems != nil {
        (*c.gems)[gemID] = gem
    }
    c.mu.Unlock()

    return gem, nil
}

// DeleteGem remove um gem customizado do servidor
func (c *GeminiClient) DeleteGem(gemID string) error {
    payload, err := json.Marshal([]interface{}{gemID})
    if err != nil {
        return fmt.Errorf("failed to marshal delete payload: %w", err)
    }

    requests := []RPCData{
        {
            RPCID:      models.RPCDeleteGem,
            Payload:    string(payload),
            Identifier: "delete",
        },
    }

    _, err = c.BatchExecute(requests)
    if err != nil {
        return fmt.Errorf("failed to delete gem: %w", err)
    }

    // Remover do cache
    c.mu.Lock()
    if c.gems != nil {
        delete(*c.gems, gemID)
    }
    c.mu.Unlock()

    return nil
}

// Gems retorna o cache de gems (nil se FetchGems nunca foi chamado)
func (c *GeminiClient) Gems() *models.GemJar {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.gems
}

// GetGem retorna um gem por ID ou nome do cache
func (c *GeminiClient) GetGem(id, name string) *models.Gem {
    c.mu.RLock()
    defer c.mu.RUnlock()

    if c.gems == nil {
        return nil
    }
    return c.gems.Get(id, name)
}
```

### 3.2 Modificar `internal/api/client.go`

Adicionar campo `gems` à struct `GeminiClient`:

```go
// GeminiClient - adicionar campo
type GeminiClient struct {
    // ... campos existentes ...
    gems *models.GemJar // Cache de gems carregados
}
```

---

## Fase 4: Integração com Generate e Chat

### 4.1 Modificar `internal/api/generate.go`

**4.1.1 Adicionar campo `GemID` ao `GenerateOptions`:**

```go
// GenerateOptions contains options for content generation
type GenerateOptions struct {
    Model    models.Model
    Metadata []string         // [cid, rid, rcid] for chat context
    Images   []*UploadedImage // Images to include in the prompt
    GemID    string           // ID do gem a usar (NOVO)
}
```

**4.1.2 Modificar `buildPayloadWithImages`:**

```go
// buildPayloadWithImages creates the f.req payload including file references and gem
func buildPayloadWithImages(prompt string, metadata []string, images []*UploadedImage, gemID string) (string, error) {
    var inner []interface{}

    if len(images) > 0 {
        var fileParts []interface{}
        for _, img := range images {
            fileParts = append(fileParts, []interface{}{
                []interface{}{img.ResourceID},
                img.FileName,
            })
        }

        inner = []interface{}{
            []interface{}{prompt, 0, nil, fileParts},
            nil,
            metadata,
        }
    } else {
        inner = []interface{}{
            []interface{}{prompt},
            nil,
            metadata,
        }
    }

    // NOVO: Adicionar gem_id se fornecido
    // Formato: 16 nulls seguidos do gem_id (posição 19 total)
    if gemID != "" {
        for i := 0; i < 16; i++ {
            inner = append(inner, nil)
        }
        inner = append(inner, gemID)
    }

    innerJSON, err := json.Marshal(inner)
    if err != nil {
        return "", err
    }

    outer := []interface{}{nil, string(innerJSON)}
    outerJSON, err := json.Marshal(outer)
    if err != nil {
        return "", err
    }

    return string(outerJSON), nil
}
```

**4.1.3 Atualizar `doGenerateContent`:**

```go
func (c *GeminiClient) doGenerateContent(prompt string, opts *GenerateOptions) (*models.ModelOutput, error) {
    // ... código existente ...

    gemID := ""
    if opts != nil {
        // ... código existente para model, metadata, images ...
        gemID = opts.GemID  // NOVO
    }

    // MODIFICAR: passar gemID
    payload, err := buildPayloadWithImages(prompt, metadata, images, gemID)
    // ... resto igual ...
}
```

### 4.2 Modificar `internal/api/session.go`

```go
// ChatSession maintains conversation context across messages
type ChatSession struct {
    client     *GeminiClient
    model      models.Model
    metadata   []string
    lastOutput *models.ModelOutput
    gemID      string // NOVO: ID do gem associado à sessão
}

// SendMessage sends a message in the chat session
func (s *ChatSession) SendMessage(prompt string) (*models.ModelOutput, error) {
    opts := &GenerateOptions{
        Model:    s.model,
        Metadata: s.metadata,
        GemID:    s.gemID,  // NOVO
    }
    // ... resto igual ...
}

// SetGem define o gem para a sessão
func (s *ChatSession) SetGem(gemID string) {
    s.gemID = gemID
}

// GetGemID retorna o gem ID da sessão
func (s *ChatSession) GetGemID() string {
    return s.gemID
}
```

### 4.3 Adicionar opções de chat em `internal/api/client.go`

```go
// ChatOption configura uma ChatSession
type ChatOption func(*ChatSession)

// WithChatModel define o modelo para a sessão
func WithChatModel(model models.Model) ChatOption {
    return func(s *ChatSession) {
        s.model = model
    }
}

// WithGem define o gem para a sessão (usando objeto Gem)
func WithGem(gem *models.Gem) ChatOption {
    return func(s *ChatSession) {
        if gem != nil {
            s.gemID = gem.ID
        }
    }
}

// WithGemID define o gem para a sessão (usando ID direto)
func WithGemID(gemID string) ChatOption {
    return func(s *ChatSession) {
        s.gemID = gemID
    }
}

// StartChatWithOptions cria uma nova sessão de chat com opções
func (c *GeminiClient) StartChatWithOptions(opts ...ChatOption) *ChatSession {
    session := &ChatSession{
        client: c,
        model:  c.GetModel(),
    }
    for _, opt := range opts {
        opt(session)
    }
    return session
}
```

---

## Fase 5: Tipos de Erro

### 5.1 Adicionar em `internal/errors/errors.go`

```go
// GemError representa erros específicos de operações com gems
type GemError struct {
    *GeminiError
    GemID   string
    GemName string
}

// NewGemError cria um novo erro de gem genérico
func NewGemError(gemID, gemName, message string) *GemError {
    return &GemError{
        GeminiError: &GeminiError{
            Operation: "gem operation",
            Endpoint:  "https://gemini.google.com/_/BardChatUi/data/batchexecute",
            Message:   message,
        },
        GemID:   gemID,
        GemName: gemName,
    }
}

// NewGemNotFoundError cria erro para gem não encontrado
func NewGemNotFoundError(idOrName string) *GemError {
    return &GemError{
        GeminiError: &GeminiError{
            Operation: "get gem",
            Message:   fmt.Sprintf("gem '%s' not found", idOrName),
        },
    }
}

// NewGemReadOnlyError cria erro para tentativa de modificar gem de sistema
func NewGemReadOnlyError(gemName string) *GemError {
    return &GemError{
        GeminiError: &GeminiError{
            Operation: "modify gem",
            Message:   fmt.Sprintf("cannot modify system gem '%s'", gemName),
        },
        GemName: gemName,
    }
}

func (e *GemError) Error() string {
    if e.GemName != "" {
        return fmt.Sprintf("gem error (%s): %s", e.GemName, e.Message)
    }
    if e.GemID != "" {
        return fmt.Sprintf("gem error (ID: %s): %s", e.GemID, e.Message)
    }
    return fmt.Sprintf("gem error: %s", e.Message)
}

// IsGemError verifica se o erro é um GemError
func IsGemError(err error) bool {
    var gemErr *GemError
    return errors.As(err, &gemErr)
}
```

---

## Fase 6: Interface de Linha de Comando

### 6.1 Criar `internal/commands/gems.go`

```go
package commands

import (
    "fmt"
    "os"
    "text/tabwriter"

    "github.com/spf13/cobra"

    "github.com/diogo/geminiweb/internal/api"
    "github.com/diogo/geminiweb/internal/browser"
    "github.com/diogo/geminiweb/internal/config"
    "github.com/diogo/geminiweb/internal/models"
)

var gemsCmd = &cobra.Command{
    Use:   "gems",
    Short: "Manage Gemini Gems (server-side personas)",
    Long: `Gems are custom personas stored on Google's servers.
Unlike local personas, gems sync across devices with your Google account.

Use 'geminiweb gems list' to see available gems.
Use 'geminiweb gems create' to create a new gem.`,
}

var gemsListCmd = &cobra.Command{
    Use:   "list",
    Short: "List all gems",
    RunE:  runGemsList,
}

var gemsCreateCmd = &cobra.Command{
    Use:   "create <name>",
    Short: "Create a new gem",
    Args:  cobra.ExactArgs(1),
    RunE:  runGemsCreate,
}

var gemsUpdateCmd = &cobra.Command{
    Use:   "update <id-or-name>",
    Short: "Update an existing gem",
    Args:  cobra.ExactArgs(1),
    RunE:  runGemsUpdate,
}

var gemsDeleteCmd = &cobra.Command{
    Use:   "delete <id-or-name>",
    Short: "Delete a gem",
    Args:  cobra.ExactArgs(1),
    RunE:  runGemsDelete,
}

var gemsShowCmd = &cobra.Command{
    Use:   "show <id-or-name>",
    Short: "Show gem details",
    Args:  cobra.ExactArgs(1),
    RunE:  runGemsShow,
}

// Flags
var (
    gemsIncludeHidden bool
    gemPrompt         string
    gemDescription    string
    gemPromptFile     string
)

func init() {
    rootCmd.AddCommand(gemsCmd)

    gemsCmd.AddCommand(gemsListCmd)
    gemsCmd.AddCommand(gemsCreateCmd)
    gemsCmd.AddCommand(gemsUpdateCmd)
    gemsCmd.AddCommand(gemsDeleteCmd)
    gemsCmd.AddCommand(gemsShowCmd)

    // Flags
    gemsListCmd.Flags().BoolVar(&gemsIncludeHidden, "hidden", false, "Include hidden system gems")

    gemsCreateCmd.Flags().StringVarP(&gemPrompt, "prompt", "p", "", "System prompt for the gem")
    gemsCreateCmd.Flags().StringVarP(&gemDescription, "description", "d", "", "Description")
    gemsCreateCmd.Flags().StringVarP(&gemPromptFile, "file", "f", "", "Read prompt from file")

    gemsUpdateCmd.Flags().StringVarP(&gemPrompt, "prompt", "p", "", "New system prompt")
    gemsUpdateCmd.Flags().StringVarP(&gemDescription, "description", "d", "", "New description")
    gemsUpdateCmd.Flags().StringVarP(&gemPromptFile, "file", "f", "", "Read prompt from file")
}

func runGemsList(cmd *cobra.Command, args []string) error {
    client, err := createGemsClient()
    if err != nil {
        return err
    }
    defer client.Close()

    gems, err := client.FetchGems(gemsIncludeHidden)
    if err != nil {
        return fmt.Errorf("failed to fetch gems: %w", err)
    }

    if gems.Len() == 0 {
        fmt.Println("No gems found.")
        return nil
    }

    w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
    fmt.Fprintln(w, "ID\tNAME\tTYPE\tDESCRIPTION")
    fmt.Fprintln(w, "--\t----\t----\t-----------")

    for _, gem := range gems.Values() {
        gemType := "custom"
        if gem.Predefined {
            gemType = "system"
        }
        desc := gem.Description
        if len(desc) > 50 {
            desc = desc[:47] + "..."
        }
        fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", gem.ID, gem.Name, gemType, desc)
    }
    w.Flush()

    return nil
}

func runGemsCreate(cmd *cobra.Command, args []string) error {
    name := args[0]

    prompt := gemPrompt
    if gemPromptFile != "" {
        data, err := os.ReadFile(gemPromptFile)
        if err != nil {
            return fmt.Errorf("failed to read prompt file: %w", err)
        }
        prompt = string(data)
    }

    if prompt == "" {
        return fmt.Errorf("prompt is required (use -p or -f)")
    }

    client, err := createGemsClient()
    if err != nil {
        return err
    }
    defer client.Close()

    gem, err := client.CreateGem(name, prompt, gemDescription)
    if err != nil {
        return fmt.Errorf("failed to create gem: %w", err)
    }

    fmt.Printf("Created gem '%s' with ID: %s\n", gem.Name, gem.ID)
    return nil
}

func runGemsUpdate(cmd *cobra.Command, args []string) error {
    idOrName := args[0]

    client, err := createGemsClient()
    if err != nil {
        return err
    }
    defer client.Close()

    gems, err := client.FetchGems(false)
    if err != nil {
        return fmt.Errorf("failed to fetch gems: %w", err)
    }

    gem := gems.Get(idOrName, idOrName)
    if gem == nil {
        return fmt.Errorf("gem '%s' not found", idOrName)
    }

    if gem.Predefined {
        return fmt.Errorf("cannot update system gems")
    }

    // Usar valores existentes se não fornecidos novos
    newPrompt := gem.Prompt
    newDesc := gem.Description
    newName := gem.Name

    if gemPromptFile != "" {
        data, err := os.ReadFile(gemPromptFile)
        if err != nil {
            return fmt.Errorf("failed to read prompt file: %w", err)
        }
        newPrompt = string(data)
    } else if gemPrompt != "" {
        newPrompt = gemPrompt
    }

    if gemDescription != "" {
        newDesc = gemDescription
    }

    updated, err := client.UpdateGem(gem.ID, newName, newPrompt, newDesc)
    if err != nil {
        return fmt.Errorf("failed to update gem: %w", err)
    }

    fmt.Printf("Updated gem '%s'\n", updated.Name)
    return nil
}

func runGemsDelete(cmd *cobra.Command, args []string) error {
    idOrName := args[0]

    client, err := createGemsClient()
    if err != nil {
        return err
    }
    defer client.Close()

    gems, err := client.FetchGems(false)
    if err != nil {
        return fmt.Errorf("failed to fetch gems: %w", err)
    }

    gem := gems.Get(idOrName, idOrName)
    if gem == nil {
        return fmt.Errorf("gem '%s' not found", idOrName)
    }

    if gem.Predefined {
        return fmt.Errorf("cannot delete system gems")
    }

    if err := client.DeleteGem(gem.ID); err != nil {
        return fmt.Errorf("failed to delete gem: %w", err)
    }

    fmt.Printf("Deleted gem '%s'\n", gem.Name)
    return nil
}

func runGemsShow(cmd *cobra.Command, args []string) error {
    idOrName := args[0]

    client, err := createGemsClient()
    if err != nil {
        return err
    }
    defer client.Close()

    gems, err := client.FetchGems(true)
    if err != nil {
        return fmt.Errorf("failed to fetch gems: %w", err)
    }

    gem := gems.Get(idOrName, idOrName)
    if gem == nil {
        return fmt.Errorf("gem '%s' not found", idOrName)
    }

    fmt.Printf("ID:          %s\n", gem.ID)
    fmt.Printf("Name:        %s\n", gem.Name)
    fmt.Printf("Description: %s\n", gem.Description)
    gemType := "custom"
    if gem.Predefined {
        gemType = "system"
    }
    fmt.Printf("Type:        %s\n", gemType)
    fmt.Printf("\nPrompt:\n%s\n", gem.Prompt)

    return nil
}

func createGemsClient() (*api.GeminiClient, error) {
    clientOpts := []api.ClientOption{
        api.WithAutoRefresh(false),
    }

    if browserType, enabled := getBrowserRefresh(); enabled {
        clientOpts = append(clientOpts, api.WithBrowserRefresh(browserType))
    }

    client, err := api.NewClient(nil, clientOpts...)
    if err != nil {
        return nil, fmt.Errorf("failed to create client: %w", err)
    }

    if err := client.Init(); err != nil {
        return nil, fmt.Errorf("failed to initialize client: %w", err)
    }

    return client, nil
}
```

### 6.2 Adicionar `--gem` flag ao comando query em `internal/commands/root.go`

```go
var gemFlag string

func init() {
    // ... flags existentes ...
    rootCmd.Flags().StringVar(&gemFlag, "gem", "", "Use a gem (by ID or name)")
}
```

### 6.3 Modificar `internal/commands/query.go`

```go
func runQuery(prompt string, rawOutput bool) error {
    // ... código existente até criar client ...

    // NOVO: Se gem especificado, buscar e validar
    var gemID string
    if gemFlag != "" {
        gems, err := client.FetchGems(false)
        if err != nil {
            return fmt.Errorf("failed to fetch gems: %w", err)
        }
        gem := gems.Get(gemFlag, gemFlag)
        if gem == nil {
            return fmt.Errorf("gem '%s' not found", gemFlag)
        }
        gemID = gem.ID

        if !rawOutput {
            fmt.Fprintf(os.Stderr, "Using gem: %s\n", gem.Name)
        }
    }

    // ... código de upload de imagem ...

    opts := &api.GenerateOptions{
        Images: images,
        GemID:  gemID, // NOVO
    }

    output, err := client.GenerateContent(actualPrompt, opts)
    // ... resto igual ...
}
```

---

## Fase 7: Atualizar Interface do Cliente

### 7.1 Modificar `GeminiClientInterface` em `internal/api/client.go`

```go
type GeminiClientInterface interface {
    // ... métodos existentes ...

    // Gems
    FetchGems(includeHidden bool) (*models.GemJar, error)
    CreateGem(name, prompt, description string) (*models.Gem, error)
    UpdateGem(gemID, name, prompt, description string) (*models.Gem, error)
    DeleteGem(gemID string) error
    Gems() *models.GemJar
    GetGem(id, name string) *models.Gem

    // Batch RPC
    BatchExecute(requests []RPCData) ([]BatchResponse, error)
}
```

---

## Checklist de Implementação

### Fase 1: Tipos e Constantes
- [ ] Criar `internal/models/gems.go` com tipos `Gem` e `GemJar`
- [ ] Adicionar RPC IDs em `internal/models/constants.go`
- [ ] Criar testes em `internal/models/gems_test.go`

### Fase 2: Sistema Batch RPC
- [ ] Criar `internal/api/batch.go` com `RPCData`, `BatchResponse`, `BatchExecute`
- [ ] Criar testes em `internal/api/batch_test.go`

### Fase 3: CRUD de Gems
- [ ] Implementar `FetchGems` em `internal/api/gems.go`
- [ ] Implementar `CreateGem`
- [ ] Implementar `UpdateGem`
- [ ] Implementar `DeleteGem`
- [ ] Adicionar campo `gems *models.GemJar` ao `GeminiClient`
- [ ] Criar testes em `internal/api/gems_test.go`

### Fase 4: Integração Generate/Chat
- [ ] Adicionar `GemID` ao `GenerateOptions`
- [ ] Modificar `buildPayloadWithImages` para incluir gem (16 nulls + gem_id)
- [ ] Adicionar `gemID` ao `ChatSession`
- [ ] Implementar `ChatOption` functions (`WithGem`, `WithGemID`)
- [ ] Criar testes de integração

### Fase 5: Tipos de Erro
- [ ] Adicionar `GemError` em `internal/errors/errors.go`
- [ ] Adicionar `NewGemNotFoundError`, `NewGemReadOnlyError`
- [ ] Adicionar `IsGemError` helper
- [ ] Criar testes

### Fase 6: CLI
- [ ] Criar `internal/commands/gems.go`
- [ ] Implementar subcomandos: list, create, update, delete, show
- [ ] Adicionar `--gem` flag ao comando query
- [ ] Registrar comando em `root.go`

### Fase 7: Interface
- [ ] Atualizar `GeminiClientInterface` com métodos de gems

### Fase 8: Documentação e Limpeza
- [ ] Atualizar README com comandos de gems
- [ ] Documentar diferença entre Personas (local) e Gems (servidor)
- [ ] Adicionar exemplos de uso
- [ ] Rodar `go test ./...` e garantir 80%+ de cobertura

---

## Notas Técnicas Importantes

### Estrutura do Payload com Gem

Quando um gem é usado em `generateContent`, o payload inner deve ter esta estrutura:

```json
[
  [prompt],           // ou [prompt, 0, null, files] se houver arquivos
  null,               // reservado
  metadata,           // [cid, rid, rcid] ou null
  null, null, null, null, null, null, null, null, null, null, null, null, null,  // 16 nulls (índices 3-18)
  "gem_id"            // ID do gem na posição 19
]
```

### Parsing de Resposta do BatchExecute

A resposta vem no formato especial do Google:
```
)]}'
[["wrb.fr","RPCID","data_json",null,null,null,"identifier"],...]
```

- Primeira linha é prefixo de segurança (`)]}'`)
- Segunda linha é JSON válido
- `data_json` (posição 2) contém os dados como string JSON que precisa ser parseada novamente
- `identifier` permite fazer match com a requisição original

### Estrutura de Resposta de Gems

```json
// Resposta de listagem (após parsear data_json)
[
  null,
  null,
  [  // Array de gems na posição [2]
    ["gem_id", ["Nome", "Descrição"], ["Prompt"]],
    ["gem_id_2", ["Nome 2", "Desc 2"], null],  // Prompt pode ser null
    ...
  ]
]
```

### Limitações Conhecidas

1. **Gems de sistema são read-only**: Não é possível criar/editar/deletar gems predefinidos pelo Google
2. **Sem validação de nome duplicado**: A API não valida se já existe gem com mesmo nome
3. **Update requer todos os campos**: Mesmo para atualizar só o prompt, deve enviar name e description
4. **Cache manual**: O cache de gems não é atualizado automaticamente - chamar `FetchGems` para atualizar
