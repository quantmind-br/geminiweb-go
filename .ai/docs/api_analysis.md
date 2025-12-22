# API Documentation

The `geminiweb-go` project provides a Go implementation and CLI for interacting with the private Google Gemini Web API. It specializes in emulating browser-like behavior to access features not available in the official Google AI SDK, such as Gems management, advanced file uploads, and specific model behaviors (Thinking/Reasoning).

## APIs Served by This Project

This project serves primarily as a **Go Package API** for programmatic integration and a **CLI Interface** for end-users. It does not expose a standalone REST or gRPC server.

### Go Package API (`internal/api`)

The following methods are exposed by the `GeminiClient` for integration into other Go services.

#### `GeminiClient.GenerateContent`
- **Description**: Sends a prompt to Gemini and returns the model's response.
- **Request Parameters**:
    - `prompt` (string): The text input for the model.
    - `opts` (`*GenerateOptions`): Configuration including `Model`, `Metadata` (for conversation context), `Files` (uploaded attachments), and `GemID`.
- **Response**: `*models.ModelOutput`
    - `Text`: The generated response.
    - `Metadata`: Context IDs (`[cid, rid, rcid]`) for subsequent requests.
- **Authentication**: Uses internal cookie management.
- **Example**:
  ```go
  client, _ := api.NewClient(cookies)
  resp, _ := client.GenerateContent("Explain Go interfaces", nil)
  fmt.Println(resp.Text)
  ```

#### `GeminiClient.UploadFile`
- **Description**: Uploads a local file or image to Google's content-push service to be used in a generation prompt.
- **Request**: `filePath` (string)
- **Response**: `*UploadedFile` containing a `ResourceID`.
- **Example**:
  ```go
  file, _ := client.UploadFile("diagram.png")
  resp, _ := client.GenerateContent("Analyze this", &api.GenerateOptions{Files: []*api.UploadedFile{file}})
  ```

#### `GeminiClient.FetchGems`
- **Description**: Retrieves all server-side personas (Gems) available to the user.
- **Request**: `includeHidden` (bool) - whether to include system-only Gems.
- **Response**: `*models.GemJar` (map of ID to Gem details).

---

### CLI Interface (User API)

The CLI provides a set of commands for direct interaction.

| Command | Purpose | Usage Example |
| :--- | :--- | :--- |
| `geminiweb [prompt]` | Single-shot query | `geminiweb "Hello world"` |
| `geminiweb chat` | Interactive TUI session | `geminiweb chat --model thinking` |
| `geminiweb gems list` | List available Gems | `geminiweb gems list` |
| `geminiweb import-cookies` | Import auth state | `geminiweb import-cookies cookies.json` |

---

### Authentication & Security

The service uses **Cookie-Based Authentication** to emulate a logged-in Google session.

1.  **Cookie Acquisition**: Cookies (`__Secure-1PSID` and `__Secure-1PSIDTS`) are retrieved from local browser stores or manual import.
2.  **Session Token (`SNlM0e`)**: Upon initialization, the client fetches the Gemini home page to extract the `SNlM0e` token, which is required as the `at` parameter in all POST requests.
3.  **TLS Fingerprinting**: The project uses `bogdanfinn/tls-client` to emulate a Chrome 133 TLS fingerprint, preventing detection as a bot.
4.  **Security Policy**: The `pkg/toolexec` package implements security policies to validate and potentially block dangerous tool executions when used with agentic workflows.

### Rate Limiting & Constraints

- **Session Expiration**: If the `SNlM0e` token or cookies expire, the client triggers an automatic refresh flow.
- **Rate Limit Handling**: The client detects Google's specific error codes (e.g., `1037` for usage limit) and returns structured errors.
- **Cookie Rotation**: Includes a dedicated `RotateCookies` call to refresh session duration without re-logging.

---

## External API Dependencies

The project relies on undocumented internal Google APIs.

### Services Consumed

#### 1. Gemini Web API (Streaming)
- **Service Name**: BardChatUi / Gemini Frontend
- **Base URL**: `https://gemini.google.com`
- **Endpoint**: `/_/BardChatUi/data/assistant.lamda.BardFrontendService/StreamGenerate`
- **Method**: `POST`
- **Payload**: Form-encoded `f.req` containing complex nested arrays representing the prompt and conversation state.
- **Authentication**: Cookies + `SNlM0e` token.
- **Error Handling**: Parses chunked JSON streams; handles fragmented messages and end-of-stream markers.

#### 2. Google Batch Execute (Gems & Management)
- **Service Name**: BatchExecute
- **Endpoint**: `https://gemini.google.com/_/BardChatUi/data/batchexecute`
- **Purpose**: Used for managing Gems (Personas).
- **RPC IDs**:
    - `CNgdBe`: List Gems
    - `oMH3Zd`: Create Gem
    - `UXcSJb`: Delete Gem
- **Authentication**: Same as main API.

#### 3. Google Content Push
- **Service Name**: File Upload Service
- **Endpoint**: `https://content-push.googleapis.com/upload`
- **Method**: `POST`
- **Purpose**: Uploading images and files for multimodal prompts.
- **Headers**: Requires `Push-ID` and specific content-type headers.

#### 4. Google Accounts Rotation
- **Service Name**: Cookie Rotation Service
- **Endpoint**: `https://accounts.google.com/RotateCookies`
- **Method**: `POST`
- **Purpose**: Extends session validity.

---

### Integration Patterns

- **Browser Emulation**: The client mimics a browser environment (User-Agent, headers, TLS fingerprint) to maintain access.
- **Automated Re-authentication**: On `401 Unauthorized` or specific Gemini auth errors, the client can automatically re-scan local browsers (Chrome, Firefox, etc.) for fresh cookies.
- **Streaming Response Parsing**: Implements a robust length-prefixed stream parser for the `StreamGenerate` endpoint to handle real-time output.

---

## Available Documentation

| Document | Path | Quality |
| :--- | :--- | :--- |
| **API Analysis Deep Dive** | `.ai/docs/api_analysis.md` | Excellent (detailed field mappings) |
| **Request Flow Analysis** | `.ai/docs/request_flow_analysis.md` | Good (covers sequence of calls) |
| **Tool Execution Protocol** | `pkg/toolexec/doc.go` | High (godoc style comments) |
| **Data Flow Analysis** | `.ai/docs/data_flow_analysis.md` | Good (architectural overview) |

Developers should refer to `internal/api/paths.go` for the exact indices used when parsing the nested array structures returned by the external Google APIs.