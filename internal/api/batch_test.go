package api

import (
	"bytes"
	"io"
	"net/url"
	"strings"
	"testing"

	http2 "github.com/bogdanfinn/fhttp"
	"github.com/bogdanfinn/tls-client/bandwidth"
	"github.com/diogo/geminiweb/internal/config"
)

// mockHTTPClient implements tls_client.HttpClient for testing
type mockHTTPClient struct {
	doFunc func(req *http2.Request) (*http2.Response, error)
}

func (m *mockHTTPClient) GetCookies(u *url.URL) []*http2.Cookie          { return nil }
func (m *mockHTTPClient) SetCookies(u *url.URL, cookies []*http2.Cookie) {}
func (m *mockHTTPClient) SetCookieJar(jar http2.CookieJar)               {}
func (m *mockHTTPClient) GetCookieJar() http2.CookieJar                  { return nil }
func (m *mockHTTPClient) SetProxy(proxyUrl string) error                 { return nil }
func (m *mockHTTPClient) GetProxy() string                               { return "" }
func (m *mockHTTPClient) SetFollowRedirect(followRedirect bool)          {}
func (m *mockHTTPClient) GetFollowRedirect() bool                        { return false }
func (m *mockHTTPClient) CloseIdleConnections()                          {}
func (m *mockHTTPClient) Get(url string) (*http2.Response, error)        { return nil, nil }
func (m *mockHTTPClient) Head(url string) (*http2.Response, error)       { return nil, nil }
func (m *mockHTTPClient) Post(url, contentType string, body io.Reader) (*http2.Response, error) {
	return nil, nil
}
func (m *mockHTTPClient) GetBandwidthTracker() bandwidth.BandwidthTracker { return nil }

func (m *mockHTTPClient) Do(req *http2.Request) (*http2.Response, error) {
	if m.doFunc != nil {
		return m.doFunc(req)
	}
	return nil, nil
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
	// Mock HTTP client que retorna resposta válida
	mockClient := &mockHTTPClient{
		doFunc: func(req *http2.Request) (*http2.Response, error) {
			// Verificar método
			if req.Method != http2.MethodPost {
				t.Errorf("Expected POST method, got %s", req.Method)
			}

			// Verificar content type
			contentType := req.Header.Get("Content-Type")
			if !strings.Contains(contentType, "application/x-www-form-urlencoded") {
				t.Errorf("Expected form content type, got %s", contentType)
			}

			response := `)]}'
[["wrb.fr","CNgdBe","[\"test_data\"]",null,null,null,"test_id"]]`

			return &http2.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(response)),
			}, nil
		},
	}

	client, err := NewClient(&config.Cookies{
		Secure1PSID:   "test-psid",
		Secure1PSIDTS: "test-psidts",
	}, WithHTTPClient(mockClient), WithAutoRefresh(false))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	client.accessToken = "test-token"

	requests := []RPCData{
		{RPCID: "CNgdBe", Payload: "[]", Identifier: "test_id"},
	}

	responses, err := client.BatchExecute(requests)
	if err != nil {
		t.Fatalf("BatchExecute failed: %v", err)
	}

	if len(responses) != 1 {
		t.Fatalf("Expected 1 response, got %d", len(responses))
	}

	if responses[0].Identifier != "test_id" {
		t.Errorf("Expected identifier 'test_id', got %s", responses[0].Identifier)
	}

	if responses[0].Data != "[\"test_data\"]" {
		t.Errorf("Expected data '[\"test_data\"]', got %s", responses[0].Data)
	}
}

func TestBatchExecuteHTTPError(t *testing.T) {
	testCases := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedErrMsg string
	}{
		{
			name:           "401 Unauthorized",
			statusCode:     401,
			responseBody:   "Unauthorized",
			expectedErrMsg: "401",
		},
		{
			name:           "500 Internal Server Error",
			statusCode:     500,
			responseBody:   "Internal Server Error",
			expectedErrMsg: "500",
		},
		{
			name:           "403 Forbidden",
			statusCode:     403,
			responseBody:   "Forbidden",
			expectedErrMsg: "403",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &mockHTTPClient{
				doFunc: func(req *http2.Request) (*http2.Response, error) {
					return &http2.Response{
						StatusCode: tc.statusCode,
						Body:       io.NopCloser(bytes.NewBufferString(tc.responseBody)),
					}, nil
				},
			}

			client, err := NewClient(&config.Cookies{
				Secure1PSID:   "test-psid",
				Secure1PSIDTS: "test-psidts",
			}, WithHTTPClient(mockClient), WithAutoRefresh(false))
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}
			client.accessToken = "test-token"

			requests := []RPCData{
				{RPCID: "CNgdBe", Payload: "[]", Identifier: "test"},
			}

			_, err = client.BatchExecute(requests)
			if err == nil {
				t.Error("Expected error for HTTP error response")
			}
			if !strings.Contains(err.Error(), tc.expectedErrMsg) {
				t.Errorf("Expected error to contain '%s', got: %v", tc.expectedErrMsg, err)
			}
		})
	}
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
