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
