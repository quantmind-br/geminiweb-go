package api

import (
	"io"
	"net/url"

	fhttp "github.com/bogdanfinn/fhttp"
	"github.com/bogdanfinn/tls-client/bandwidth"
)

// MockResponseBody is a ReadCloser that simulates reading response data
type MockResponseBody struct {
	data []byte
	pos  int
}

// NewMockResponseBody creates a new MockResponseBody with the given data
func NewMockResponseBody(data []byte) *MockResponseBody {
	return &MockResponseBody{data: data, pos: 0}
}

// Read implements the io.Reader interface
func (m *MockResponseBody) Read(p []byte) (n int, err error) {
	if m.pos >= len(m.data) {
		return 0, io.EOF
	}
	n = copy(p, m.data[m.pos:])
	m.pos += n
	return n, nil
}

// Close implements the io.Closer interface
func (m *MockResponseBody) Close() error {
	return nil
}

// MockHttpClient is a mock implementation of tls_client.HttpClient for testing
type MockHttpClient struct {
	Response *fhttp.Response
	Err      error
}

// GetCookies implements the tls_client.HttpClient interface
func (m *MockHttpClient) GetCookies(u *url.URL) []*fhttp.Cookie {
	return nil
}

// SetCookies implements the tls_client.HttpClient interface
func (m *MockHttpClient) SetCookies(u *url.URL, cookies []*fhttp.Cookie) {}

// SetCookieJar implements the tls_client.HttpClient interface
func (m *MockHttpClient) SetCookieJar(jar fhttp.CookieJar) {}

// GetCookieJar implements the tls_client.HttpClient interface
func (m *MockHttpClient) GetCookieJar() fhttp.CookieJar {
	return nil
}

// SetProxy implements the tls_client.HttpClient interface
func (m *MockHttpClient) SetProxy(proxyUrl string) error {
	return nil
}

// GetProxy implements the tls_client.HttpClient interface
func (m *MockHttpClient) GetProxy() string {
	return ""
}

// SetFollowRedirect implements the tls_client.HttpClient interface
func (m *MockHttpClient) SetFollowRedirect(followRedirect bool) {}

// GetFollowRedirect implements the tls_client.HttpClient interface
func (m *MockHttpClient) GetFollowRedirect() bool {
	return false
}

// CloseIdleConnections implements the tls_client.HttpClient interface
func (m *MockHttpClient) CloseIdleConnections() {}

// Do implements the tls_client.HttpClient interface
func (m *MockHttpClient) Do(req *fhttp.Request) (*fhttp.Response, error) {
	return m.Response, m.Err
}

// Get implements the tls_client.HttpClient interface
func (m *MockHttpClient) Get(url string) (*fhttp.Response, error) {
	return m.Response, m.Err
}

// Head implements the tls_client.HttpClient interface
func (m *MockHttpClient) Head(url string) (*fhttp.Response, error) {
	return m.Response, m.Err
}

// Post implements the tls_client.HttpClient interface
func (m *MockHttpClient) Post(url, contentType string, body io.Reader) (*fhttp.Response, error) {
	return m.Response, m.Err
}

// GetBandwidthTracker implements the tls_client.HttpClient interface
func (m *MockHttpClient) GetBandwidthTracker() bandwidth.BandwidthTracker {
	return nil
}

// NewMockHttpClient creates a new MockHttpClient with a successful response
func NewMockHttpClient(body []byte, statusCode int) *MockHttpClient {
	return &MockHttpClient{
		Response: &fhttp.Response{
			StatusCode: statusCode,
			Body:       NewMockResponseBody(body),
			Header:     make(fhttp.Header),
		},
	}
}

// NewMockHttpClientWithError creates a new MockHttpClient that returns an error
func NewMockHttpClientWithError(err error) *MockHttpClient {
	return &MockHttpClient{
		Response: nil,
		Err:      err,
	}
}
