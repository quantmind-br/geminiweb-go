package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/diogo/geminiweb/internal/config"
)

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

func TestParseBatchResponseSingleRequest(t *testing.T) {
	requests := []RPCData{
		{RPCID: "CNgdBe", Payload: "[3]", Identifier: "gems"},
	}

	body := []byte(`)]}'
[["wrb.fr","CNgdBe","{\"data\":\"test\"}",null,null,null,"gems"]]`)

	responses, err := parseBatchResponse(body, requests)
	if err != nil {
		t.Fatalf("parseBatchResponse failed: %v", err)
	}

	if len(responses) != 1 {
		t.Fatalf("Expected 1 response, got %d", len(responses))
	}

	if responses[0].Identifier != "gems" {
		t.Errorf("Expected identifier 'gems', got %s", responses[0].Identifier)
	}
	if responses[0].Data != "{\"data\":\"test\"}" {
		t.Errorf("Expected data '{\"data\":\"test\"}', got %s", responses[0].Data)
	}
}

func TestParseBatchResponseInvalidJSON(t *testing.T) {
	requests := []RPCData{
		{RPCID: "CNgdBe", Payload: "[]", Identifier: "test"},
	}

	body := []byte(`)]}'
not valid json`)

	_, err := parseBatchResponse(body, requests)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestParseBatchResponseEmptyBody(t *testing.T) {
	requests := []RPCData{
		{RPCID: "CNgdBe", Payload: "[]", Identifier: "test"},
	}

	body := []byte(`)]}'
`)

	_, err := parseBatchResponse(body, requests)
	if err == nil {
		t.Error("Expected error for empty response")
	}
}

func TestParseBatchResponseNoMatch(t *testing.T) {
	requests := []RPCData{
		{RPCID: "CNgdBe", Payload: "[]", Identifier: "expected"},
	}

	// Response com identifier diferente
	body := []byte(`)]}'
[["wrb.fr","CNgdBe","data",null,null,null,"different"]]`)

	responses, err := parseBatchResponse(body, requests)
	if err != nil {
		t.Fatalf("parseBatchResponse failed: %v", err)
	}

	// Deve retornar resposta com identifier original mas sem data
	if len(responses) != 1 {
		t.Fatalf("Expected 1 response, got %d", len(responses))
	}
	if responses[0].Identifier != "expected" {
		t.Errorf("Expected identifier 'expected', got %s", responses[0].Identifier)
	}
	if responses[0].Data != "" {
		t.Errorf("Expected empty data, got %s", responses[0].Data)
	}
}

func TestBatchExecuteEmptyRequests(t *testing.T) {
	client, err := NewClient(&config.Cookies{
		Secure1PSID:   "test-psid",
		Secure1PSIDTS: "test-psidts",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	client.accessToken = "test-token"

	_, err = client.BatchExecute([]RPCData{})
	if err == nil {
		t.Error("Expected error for empty requests")
	}
	if !strings.Contains(err.Error(), "no requests provided") {
		t.Errorf("Expected 'no requests provided' error, got: %v", err)
	}
}

func TestBatchExecuteClosedClient(t *testing.T) {
	client, err := NewClient(&config.Cookies{
		Secure1PSID:   "test-psid",
		Secure1PSIDTS: "test-psidts",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	client.Close()

	_, err = client.BatchExecute([]RPCData{
		{RPCID: "test", Payload: "[]", Identifier: "test"},
	})
	if err == nil {
		t.Error("Expected error for closed client")
	}
	if !strings.Contains(err.Error(), "client is closed") {
		t.Errorf("Expected 'client is closed' error, got: %v", err)
	}
}

func TestBatchExecuteWithMockServer(t *testing.T) {
	// Mock server que retorna resposta válida
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verificar método
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		// Verificar content type
		contentType := r.Header.Get("Content-Type")
		if !strings.Contains(contentType, "application/x-www-form-urlencoded") {
			t.Errorf("Expected form content type, got %s", contentType)
		}

		response := `)]}'
[["wrb.fr","CNgdBe","[\"test_data\"]",null,null,null,"test_id"]]`
		w.WriteHeader(200)
		_, _ = w.Write([]byte(response))
	}))
	defer server.Close()

	// Nota: Este teste não funciona com o mock porque o cliente usa um endpoint fixo
	// Seria necessário injetar a URL ou usar um transport mock
	t.Skip("Requires endpoint injection or transport mock")
}

func TestBatchExecuteHTTPError(t *testing.T) {
	// Este teste verifica o comportamento quando o servidor retorna erro
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		_, _ = w.Write([]byte("Unauthorized"))
	}))
	defer server.Close()

	// Nota: Assim como o teste anterior, requer injeção de endpoint
	t.Skip("Requires endpoint injection or transport mock")
}

func TestParseBatchResponseWithPrefixVariants(t *testing.T) {
	requests := []RPCData{
		{RPCID: "CNgdBe", Payload: "[]", Identifier: "test"},
	}

	testCases := []struct {
		name string
		body []byte
	}{
		{
			name: "with )]}' prefix",
			body: []byte(")]}'\n[[\"wrb.fr\",\"CNgdBe\",\"data\",null,null,null,\"test\"]]"),
		},
		{
			name: "with )]} prefix",
			body: []byte(")]}\n[[\"wrb.fr\",\"CNgdBe\",\"data\",null,null,null,\"test\"]]"),
		},
		{
			name: "with empty lines",
			body: []byte("\n\n)]}'\n\n[[\"wrb.fr\",\"CNgdBe\",\"data\",null,null,null,\"test\"]]"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			responses, err := parseBatchResponse(tc.body, requests)
			if err != nil {
				t.Fatalf("parseBatchResponse failed: %v", err)
			}
			if len(responses) != 1 {
				t.Fatalf("Expected 1 response, got %d", len(responses))
			}
			if responses[0].Data != "data" {
				t.Errorf("Expected data 'data', got %s", responses[0].Data)
			}
		})
	}
}

func TestBatchResponseStruct(t *testing.T) {
	resp := BatchResponse{
		Identifier: "test-id",
		Data:       "{\"key\":\"value\"}",
		Error:      nil,
	}

	if resp.Identifier != "test-id" {
		t.Errorf("Expected Identifier 'test-id', got %s", resp.Identifier)
	}
	if resp.Data != "{\"key\":\"value\"}" {
		t.Errorf("Expected Data '{\"key\":\"value\"}', got %s", resp.Data)
	}
	if resp.Error != nil {
		t.Errorf("Expected nil Error, got %v", resp.Error)
	}
}
