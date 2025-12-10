# Diagnóstico: Mudança na API do Gemini Web (Dezembro 2024)

## Resumo Executivo

Em dezembro de 2024, o Google implementou mudanças significativas na API web do Gemini que quebraram a compatibilidade com o geminiweb. A principal mudança foi a introdução de um **token de verificação anti-bot** no payload das requisições.

**Status**: API incompatível - requer implementação de novo mecanismo de autenticação.

---

## 1. Sintomas Observados

### Erro Original
```
parse error: no valid JSON found in response
```

### Erro Após Correções Parciais
```
API error: unknown error
Error Code: 2 (unknown error)
Endpoint: https://gemini.google.com/_/BardChatUi/data/assistant.lamda.BardFrontendService/StreamGenerate
```

---

## 2. Análise Comparativa

### 2.1 Formato da Resposta (Corrigido)

A API agora retorna respostas em formato **streaming** com múltiplos chunks JSON:

```
)]}'                           <- Prefixo XSSI protection
160                            <- Tamanho do chunk
[["wrb.fr",null,"..."]]        <- Chunk 1 (metadata, sem candidatos)
861
[["wrb.fr",null,"..."]]        <- Chunk 2 (pode ter candidatos)
...
31
[["e",125,null,null,1066124]]  <- Marcador de fim do stream
```

**Correção aplicada**: Adicionada detecção do marcador de fim `[["e",` em `internal/api/generate.go:155-162`.

### 2.2 Headers da Requisição

#### Header Faltante (Adicionado)
```
x-goog-ext-73010989-jspb: [0]
```

**Correção aplicada**: Adicionado em `internal/models/constants.go:101`.

#### Header do Modelo (Já existente)
```
x-goog-ext-525001261-jspb: [1,null,null,null,"e6fa609c3fa255c0",null,null,0,[4],null,null,2]
```

### 2.3 Estrutura do Payload (Mudança Crítica)

#### Formato Antigo (geminiweb)
```json
[null, "[[\"prompt\"], null, [\"cid\", \"rid\", \"rcid\"]]"]
```

Estrutura interna:
- Elemento 0: `["prompt"]` - apenas o prompt
- Elemento 1: `null`
- Elemento 2: `["cid", "rid", "rcid"]` - metadata do chat

#### Formato Novo (API Atual)
```json
[null, "[[\"prompt\",0,null,null,null,null,0],[\"pt\"],[\"\",\"\",\"\",...],\"!TOKEN\",\"HASH\",...]]"]
```

Estrutura interna (101 elementos):
| Posição | Conteúdo | Descrição |
|---------|----------|-----------|
| 0 | `["prompt", 0, null, null, null, null, 0]` | Prompt com flags |
| 1 | `["pt"]` | Código do idioma |
| 2 | `["", "", "", null, null, null, null, null, null, ""]` | Metadata expandida |
| 3 | `"!Li2lLXXNAAb9MTdP3TFC..."` | **Token de verificação** (~1463 chars) |
| 4 | `"a136525ab0fe58168c4fadc30f0d2c38"` | Hash/checksum |
| 5-100 | Diversos valores, nulls e flags | Configurações adicionais |

---

## 3. Token de Verificação (Elemento 3)

### Características
- **Prefixo**: Sempre começa com `!`
- **Tamanho**: Aproximadamente 1463 caracteres
- **Formato**: Base64-like com caracteres `A-Za-z0-9_-`
- **Exemplo**: `!Li2lLXXNAAb9MTdP3TFCZA9-gAVeSCw7ADQBEArZ1Ctg6p8viN-M7T6m8NGHT-hvOCTFUaMCImEAId_cntrBQW5rOa055eJ5WsNCp4snAgAAAa5SAAAAHWgBB34A...`

### Origem Provável
Este token é gerado pelo **BotGuard** (sistema anti-bot do Google) que:
1. Executa JavaScript no navegador
2. Coleta fingerprints do ambiente
3. Gera um token assinado que prova que a requisição vem de um navegador real

### Tokens Encontrados na Página de Init (Não são o token de verificação)
```
"SNlM0e": "ANHAVo3O0eddPrPNhJ8e3q-_v6BF:1765201991852"  <- Access token (já usado)
"thykhd": "AFWLbD3DGAUkN9oTU8CYsErX-aOj3D7F0diMFWQ1Z..."  <- Token auxiliar
"PI9WOb": "CAMS9gIV9gXM2swNqASxpNAFgYn4DtP1BesCt967F..."  <- Outro token
"cfb2h": "boq_assistant-bard-web-server_20251203.10_p1"    <- Versão do servidor
```

Nenhum destes corresponde ao formato `!...` do token de verificação, indicando que ele é gerado dinamicamente pelo JavaScript.

---

## 4. Código de Erro 2

Quando o token de verificação está ausente ou inválido, a API retorna:

```json
[["wrb.fr",null,null,null,null,[2]],["di",13654],["af.httprm",13654,"8805643464341139065",3]]
```

- Posição `[0][5][0]` = `2` (código de erro)
- Este código não está documentado nos códigos conhecidos do geminiweb

### Códigos de Erro Conhecidos
| Código | Significado |
|--------|-------------|
| 2 | Verificação anti-bot falhou (novo) |
| 3 | Prompt muito longo |
| 1037 | Limite de uso excedido |
| 1050 | Modelo inconsistente |
| 1052 | Header do modelo inválido |
| 1060 | IP bloqueado |

---

## 5. Correções Implementadas

### 5.1 Detecção de Fim de Stream
**Arquivo**: `internal/api/generate.go`

```go
// Read response body
// The Gemini API uses a streaming format with chunks: {size}\n{json}\n
// The stream ends with a special marker: [["e",status,null,null,bytes]]
body := make([]byte, 0, 65536)
buf := make([]byte, 4096)
streamEndMarker := []byte(`[["e",`)
for {
    n, err := resp.Body.Read(buf)
    if n > 0 {
        body = append(body, buf[:n]...)
        // Check if we've received the stream end marker
        if bytes.Contains(body, streamEndMarker) {
            break
        }
    }
    if err != nil {
        break
    }
}
```

### 5.2 Header Adicionado
**Arquivo**: `internal/models/constants.go`

```go
func DefaultHeaders() map[string]string {
    return map[string]string{
        // ... outros headers ...
        "x-goog-ext-73010989-jspb": "[0]", // Required safety/feature flag header
    }
}
```

### 5.3 Parsing de Resposta Multi-chunk
**Arquivo**: `internal/api/generate.go` (função `parseResponse`)

O parsing foi ajustado para iterar por todos os chunks JSON válidos até encontrar um com candidatos contendo texto.

---

## 6. Soluções Possíveis

### 6.1 API Oficial do Google (Recomendada)
Migrar para a API oficial do Gemini que usa chaves de API em vez de cookies:
- https://ai.google.dev/gemini-api/docs
- Requer criar projeto no Google Cloud
- Sem limitações de anti-bot

### 6.2 Automação de Navegador
Usar Playwright, Puppeteer ou similar para:
1. Carregar a página do Gemini
2. Executar o JavaScript que gera o token
3. Extrair o token do contexto JavaScript
4. Usar o token nas requisições

**Desvantagem**: Overhead significativo, mais lento, mais complexo.

### 6.3 Monitorar Projetos da Comunidade
- [HanaokaYuzu/Gemini-API](https://github.com/HanaokaYuzu/Gemini-API) - Wrapper Python ativamente mantido
- [dsdanielpark/Gemini-API](https://github.com/dsdanielpark/Gemini-API) - Projeto original

Estes projetos podem encontrar soluções para o token de verificação.

### 6.4 Reverse Engineering do BotGuard
Analisar o JavaScript do BotGuard para entender como o token é gerado.
**Nota**: Isso pode violar os termos de serviço do Google.

---

## 7. Arquivos de Referência

### Requisições Capturadas
- `requests/geminiwebrequest.txt` - Primeira captura de requisição real
- `requests/geminirequest2.txt` - Segunda captura com resposta completa

### Página de Init Capturada
- `/tmp/gemini_init_page.html` - HTML da página de inicialização (temporário)

### Resposta de Debug
- `/tmp/geminiweb_debug_response.txt` - Resposta da API com erro (temporário)

---

## 8. Conclusão

A API web do Gemini agora requer um token de verificação gerado pelo BotGuard que não pode ser obtido sem executar JavaScript no navegador. Isso efetivamente bloqueia clientes HTTP simples como o geminiweb.

**Recomendação**: Considerar migração para a API oficial do Google Gemini ou implementar automação de navegador para extrair o token de verificação.

---

*Documento criado em: 2024-12-08*
*Baseado na análise de requisições capturadas do navegador Chrome*
