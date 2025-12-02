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
	// Formato final: [[[rpc1], [rpc2], ...]] - nota: 3 níveis de colchetes
	var serialized []interface{}
	for _, req := range requests {
		serialized = append(serialized, req.Serialize())
	}

	// Wrap in outer array: [[...]] -> [[[...]]]
	outerPayload := []interface{}{serialized}

	payload, err := json.Marshal(outerPayload)
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
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if resp.StatusCode != 200 {
		// Read response body for error diagnostics
		errorBody := make([]byte, 0, 4096)
		buf := make([]byte, 1024)
		for {
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				errorBody = append(errorBody, buf[:n]...)
				if len(errorBody) >= 4096 {
					break
				}
			}
			if readErr != nil {
				break
			}
		}
		return nil, apierrors.NewAPIErrorWithBody(resp.StatusCode, models.EndpointBatchExec, "batch execute failed", string(errorBody))
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
