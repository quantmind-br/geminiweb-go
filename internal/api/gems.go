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
		name,            // 0: nome
		description,     // 1: descrição
		prompt,          // 2: system prompt
		nil,             // 3
		nil,             // 4
		nil,             // 5
		nil,             // 6
		nil,             // 7
		0,               // 8: flag
		nil,             // 9
		1,               // 10: flag
		nil,             // 11
		nil,             // 12
		nil,             // 13
		[]interface{}{}, // 14: array vazio
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
		name,            // 0
		description,     // 1
		prompt,          // 2
		nil,             // 3
		nil,             // 4
		nil,             // 5
		nil,             // 6
		nil,             // 7
		0,               // 8
		nil,             // 9
		1,               // 10
		nil,             // 11
		nil,             // 12
		nil,             // 13
		[]interface{}{}, // 14
		0,               // 15: flag extra (diferencia de create)
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
