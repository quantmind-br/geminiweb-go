Com base na análise do código fornecido, preparei um plano de refatoração focado em remover código morto (especificamente a duplicação na lógica de upload), otimizar o acesso à configuração (cacheamento) e limpar definições redundantes.

Aqui estão as modificações propostas:

### 1\. Remover Código Legado e Duplicado (`internal/api/upload.go`)

O arquivo `upload.go` contém duas estruturas quase idênticas: `FileUploader` e `ImageUploader`. A `ImageUploader` está marcada como depreciada. Vamos remover a `ImageUploader` e atualizar os métodos de conveniência no `GeminiClient` para usar apenas o `FileUploader`.

**Arquivo: `internal/api/upload.go`**
_Substitua o conteúdo do arquivo por:_

```go
package api

import (
 "bytes"
 "fmt"
 "io"
 "mime"
 "mime/multipart"
 "os"
 "path/filepath"
 "strings"

 fhttp "github.com/bogdanfinn/fhttp"

 apierrors "github.com/diogo/geminiweb/internal/errors"
 "github.com/diogo/geminiweb/internal/models"
)

const (
 MaxImageSize = 20 * 1024 * 1024 // 20MB
 MaxFileSize  = 50 * 1024 * 1024 // 50MB for text files
 // LargePromptThreshold is the size (in bytes) above which prompts should be uploaded as files
 LargePromptThreshold = 100 * 1024 // 100KB
)

// SupportedImageTypes returns the list of supported MIME types for image upload
func SupportedImageTypes() []string {
 return []string{
  "image/jpeg",
  "image/png",
  "image/gif",
  "image/webp",
 }
}

// SupportedTextTypes returns the list of supported MIME types for text file upload
func SupportedTextTypes() []string {
 return []string{
  "text/plain",
  "text/markdown",
  "text/x-markdown",
  "application/json",
  "text/csv",
  "text/html",
  "text/xml",
  "application/xml",
 }
}

// UploadedFile represents an uploaded file ready for use in prompts
type UploadedFile struct {
 ResourceID string
 FileName   string
 MIMEType   string
 Size       int64
}

// UploadedImage represents an uploaded image ready for use in prompts
// Type alias for backward compatibility
type UploadedImage = UploadedFile

// FileUploader handles file uploads to Gemini (images, text, etc.)
type FileUploader struct {
 client *GeminiClient
}

// NewFileUploader creates a new file uploader
func NewFileUploader(client *GeminiClient) *FileUploader {
 return &FileUploader{
  client: client,
 }
}

// UploadFile uploads any supported file from disk (images or text)
func (u *FileUploader) UploadFile(filePath string) (*UploadedFile, error) {
 // Get file info
 fileInfo, err := os.Stat(filePath)
 if err != nil {
  return nil, fmt.Errorf("failed to stat file: %w", err)
 }

 // Detect MIME type
 ext := filepath.Ext(filePath)
 mimeType := mime.TypeByExtension(ext)
 if mimeType == "" {
  mimeType = "application/octet-stream"
 }

 // Determine max size based on file type
 maxSize := int64(MaxFileSize)
 if u.isImageType(mimeType) {
  maxSize = MaxImageSize
 }

 if fileInfo.Size() > maxSize {
  return nil, fmt.Errorf("file size (%d bytes) exceeds maximum (%d bytes)", fileInfo.Size(), maxSize)
 }

 // Open file
 file, err := os.Open(filePath)
 if err != nil {
  return nil, fmt.Errorf("failed to open file: %w", err)
 }
 defer func() {
  if file != nil {
   _ = file.Close()
  }
 }()

 return u.uploadStream(file, filepath.Base(filePath), mimeType, fileInfo.Size())
}

// UploadText uploads text content as a file
func (u *FileUploader) UploadText(content string, fileName string) (*UploadedFile, error) {
 if fileName == "" {
  fileName = "prompt.txt"
 }

 // Ensure .txt extension for proper MIME detection
 if filepath.Ext(fileName) == "" {
  fileName += ".txt"
 }

 data := []byte(content)
 if int64(len(data)) > MaxFileSize {
  return nil, fmt.Errorf("content size (%d bytes) exceeds maximum (%d bytes)", len(data), MaxFileSize)
 }

 mimeType := "text/plain"
 ext := filepath.Ext(fileName)
 if detectedType := mime.TypeByExtension(ext); detectedType != "" {
  mimeType = detectedType
 }

 return u.uploadStream(bytes.NewReader(data), fileName, mimeType, int64(len(data)))
}

// UploadFromReader uploads data directly from an io.Reader
func (u *FileUploader) UploadFromReader(reader io.Reader, fileName, mimeType string) (*UploadedFile, error) {
 // We need to read content to determine size for multipart header usually,
 // or rely on the buffer. For simplicity/safety with current API impl:
 data, err := io.ReadAll(reader)
 if err != nil {
  return nil, fmt.Errorf("failed to read data: %w", err)
 }

 size := int64(len(data))
 maxSize := int64(MaxFileSize)
 if u.isImageType(mimeType) {
  maxSize = MaxImageSize
 }

 if size > maxSize {
  return nil, fmt.Errorf("data size exceeds maximum %d bytes", maxSize)
 }

 return u.uploadStream(bytes.NewReader(data), fileName, mimeType, size)
}

// uploadStream executes the actual upload using Google's content-push service
func (u *FileUploader) uploadStream(
 reader io.Reader,
 fileName string,
 mimeType string,
 size int64,
) (*UploadedFile, error) {
 // Create multipart body
 var body bytes.Buffer
 writer := multipart.NewWriter(&body)

 // Add file field
 part, err := writer.CreateFormFile("file", fileName)
 if err != nil {
  return nil, apierrors.NewUploadError(fileName, fmt.Sprintf("failed to create form file: %v", err))
 }

 if _, err := io.Copy(part, reader); err != nil {
  return nil, apierrors.NewUploadError(fileName, fmt.Sprintf("failed to write file data: %v", err))
 }

 _ = writer.Close()

 // Simple POST to upload endpoint
 req, err := fhttp.NewRequest(fhttp.MethodPost, models.EndpointUpload, &body)
 if err != nil {
  return nil, apierrors.NewUploadError(fileName, fmt.Sprintf("failed to create request: %v", err))
 }

 // Headers
 req.Header.Set("Content-Type", writer.FormDataContentType())
 for key, value := range models.UploadHeaders() {
  req.Header.Set(key, value)
 }

 resp, err := u.client.httpClient.Do(req)
 if err != nil {
  return nil, apierrors.NewUploadNetworkError(fileName, err)
 }
 defer resp.Body.Close()

 if resp.StatusCode != 200 && resp.StatusCode != 201 {
  bodyBytes, _ := io.ReadAll(resp.Body)
  return nil, apierrors.NewUploadErrorWithStatus(fileName, resp.StatusCode, string(bodyBytes))
 }

 respBody, err := io.ReadAll(resp.Body)
 if err != nil {
  return nil, apierrors.NewUploadError(fileName, fmt.Sprintf("failed to read response: %v", err))
 }

 resourceID := strings.TrimSpace(string(respBody))
 if resourceID == "" {
  return nil, apierrors.NewUploadError(fileName, "empty resource ID in upload response")
 }

 return &UploadedFile{
  ResourceID: resourceID,
  FileName:   fileName,
  MIMEType:   mimeType,
  Size:       size,
 }, nil
}

func (u *FileUploader) isImageType(mimeType string) bool {
 for _, supported := range SupportedImageTypes() {
  if strings.HasPrefix(mimeType, supported) {
   return true
  }
 }
 return false
}

// Convenience methods on GeminiClient to maintain API compatibility

// UploadImage is a convenience method on GeminiClient for uploading images
func (c *GeminiClient) UploadImage(filePath string) (*UploadedImage, error) {
 uploader := NewFileUploader(c)
 return uploader.UploadFile(filePath)
}

// UploadImageFromReader is a convenience method for uploading from a reader
func (c *GeminiClient) UploadImageFromReader(
 reader io.Reader,
 fileName string,
 mimeType string,
) (*UploadedImage, error) {
 uploader := NewFileUploader(c)
 return uploader.UploadFromReader(reader, fileName, mimeType)
}

// UploadFile is a convenience method on GeminiClient for uploading any file
func (c *GeminiClient) UploadFile(filePath string) (*UploadedFile, error) {
 uploader := NewFileUploader(c)
 return uploader.UploadFile(filePath)
}

// UploadText is a convenience method for uploading text content as a file
func (c *GeminiClient) UploadText(content string, fileName string) (*UploadedFile, error) {
 uploader := NewFileUploader(c)
 return uploader.UploadText(content, fileName)
}
```

### 2\. Otimização de Performance: Cache de Configuração (`internal/config/config.go`)

Atualmente, `LoadConfig` lê o arquivo do disco toda vez. Em loops de TUI ou múltiplas chamadas, isso é ineficiente. Vamos adicionar um Singleton/Cache simples.

**Arquivo: `internal/config/config.go`**
_Modifique a função `LoadConfig` e adicione variáveis globais:_

```go
// Variáveis para cache
var (
 cachedConfig *Config
 configLoaded bool
)

// ForceReloadConfig forces reading configuration from disk
func ForceReloadConfig() (Config, error) {
 cfg := DefaultConfig()

 configPath, err := GetConfigPath()
 if err != nil {
  return cfg, err
 }

 data, err := os.ReadFile(configPath)
 if err != nil {
  if os.IsNotExist(err) {
   // Update cache even if using defaults
   cachedConfig = &cfg
   configLoaded = true
   return cfg, nil
  }
  return cfg, fmt.Errorf("failed to read config file: %w", err)
 }

 if err := json.Unmarshal(data, &cfg); err != nil {
  return DefaultConfig(), fmt.Errorf("failed to parse config file: %w", err)
 }

 cachedConfig = &cfg
 configLoaded = true
 return cfg, nil
}

// LoadConfig loads the configuration from disk (cached)
func LoadConfig() (Config, error) {
 if configLoaded && cachedConfig != nil {
  return *cachedConfig, nil
 }
 return ForceReloadConfig()
}

// SaveConfig saves the configuration to disk and updates cache
func SaveConfig(cfg Config) error {
 configDir, err := EnsureConfigDir()
 if err != nil {
  return err
 }

 configPath := filepath.Join(configDir, "config.json")

 data, err := json.MarshalIndent(cfg, "", "  ")
 if err != nil {
  return fmt.Errorf("failed to marshal config: %w", err)
 }

 if err := os.WriteFile(configPath, data, 0o644); err != nil {
  return fmt.Errorf("failed to write config file: %w", err)
 }

 // Update cache
 cachedConfig = &cfg
 configLoaded = true

 return nil
}
```

### 3\. Limpeza de Código: Redundância em Modelos (`internal/models/constants.go`)

O pacote `models` está importando `apierrors` apenas para fazer _alias_ de constantes de erro. Isso cria acoplamento desnecessário e potencial importação cíclica futura. Devemos remover esses aliases e fazer com que o código consumidor use `apierrors` diretamente (o que já acontece em `internal/errors/errors.go`).

**Arquivo: `internal/models/constants.go`**
_Remova a importação de `apierrors` e o bloco de constantes/tipos no final:_

```go
package models

// Remova: import apierrors "github.com/diogo/geminiweb/internal/errors"

// ... (Mantenha constantes de Endpoints, RPC e Modelos) ...

// Remova todo este bloco final:
/*
// ErrorCode represents known API error codes
// Deprecated: Use errors.ErrorCode instead. These are kept for backward compatibility.
type ErrorCode = apierrors.ErrorCode

// Error code constants - aliased from errors package for backward compatibility
const (
 ErrUsageLimitExceeded = apierrors.ErrCodeUsageLimitExceeded
 ErrModelInconsistent  = apierrors.ErrCodeModelInconsistent
 ErrModelHeaderInvalid = apierrors.ErrCodeModelHeaderInvalid
 ErrIPBlocked          = apierrors.ErrCodeIPBlocked
)
*/
```

_Nota: Se houver código que dependa desses aliases (como `internal/api/generate.go`), você deve alterar a importação nesses arquivos para usar `apierrors.ErrCode...` diretamente, o que já parece ser o caso no código fornecido._

### 4\. Otimização de Leitura HTTP (`internal/api/batch.go` e `generate.go`)

A leitura manual de buffers (`resp.Body.Read` em loop) é verbosa e menos otimizada que `io.ReadAll` (que ajusta buffers automaticamente).

**Arquivo: `internal/api/batch.go`**
_Modifique a leitura do corpo no final de `BatchExecute`:_

```go
 // ... (código anterior de verificação de status)

 // Substituir o loop manual de leitura por io.ReadAll
 body, err := io.ReadAll(resp.Body)
 if err != nil {
  return nil, apierrors.NewNetworkErrorWithEndpoint("read batch response", models.EndpointBatchExec, err)
 }

 return parseBatchResponse(body, requests)
}
```

Fazer o mesmo em `internal/api/generate.go` no método `doGenerateContent`:

```go
 // ...
 // Read response body
 body, err := io.ReadAll(resp.Body)
 if err != nil {
  return nil, apierrors.NewNetworkErrorWithEndpoint("read generate response", models.EndpointGenerate, err)
 }

 return parseResponse(body, model.Name)
}
```

### Resumo das Melhorias

1. **Dead Code:** `ImageUploader` removido. O código agora é mais limpo e fácil de manter com apenas uma estrutura `FileUploader` fazendo todo o trabalho.
2. **Performance:** `LoadConfig` agora usa cache em memória, evitando I/O de disco repetitivo em comandos interativos e TUI.
3. **Organização:** `models` deixa de depender de `errors` apenas para _aliasing_, o que melhora a árvore de dependências.
4. **Simplicidade:** Substituição de loops manuais de leitura de buffer por `io.ReadAll` da biblioteca padrão, que é otimizado e menos propenso a erros.
