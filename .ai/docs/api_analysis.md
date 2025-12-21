# API Documentation

The `geminiweb-go` project is a command-line interface and Go library designed to interact with the private Google Gemini Web API. It emulates browser behavior to provide features not available in the public Google AI SDK, such as Gems management, advanced file uploads, and specific model behaviors.

## APIs Served by This Project

This project primarily serves as a **CLI tool** and a **Go Package API**. It does not expose a REST/HTTP server by default.

### CLI Interface (User-Facing API)

The CLI acts as the primary interface for users and scripts to interact with the Gemini service.

| Command | Description | Key Flags |
| :--- | :--- | :--- |
| `geminiweb [prompt]` | Send a single query and get a response. | `--model`, `--image`, `--file`, `--gem`, `--output` |
| `geminiweb chat` | Start an interactive TUI chat session. | `--new`, `--persona`, `--gem` |
| `geminiweb gems` | Manage server-side personas (Gems). | `list`, `create`, `delete` |
| `geminiweb history` | Manage local conversation history. | `list`, `export`, `search` |
| `geminiweb config` | Configure application settings. | `--model`, `--theme` |
| `geminiweb import-cookies` | Manually import cookies from a JSON file. | N/A |

### Go Package API (`internal/api`)

For developers integrating `geminiweb` into other Go projects, the `internal/api` package provides the following programmatic interface:

#### `GeminiClient.GenerateContent(prompt string, opts *GenerateOptions)`
- **Description**: The core method for generating responses from Gemini.
- **Request**:
  - `prompt`: The text string to send.
  - `opts`: Struct containing `Model`, `Metadata` (for context), `Files` (uploaded attachments), and `GemID`.
- **Response**: `*models.ModelOutput` containing the response text, candidate IDs, and metadata.
- **Error Handling**: Automatically attempts browser-based cookie refresh if an authentication error occurs.

#### `FileUploader.UploadFile(filePath string)`
- **Description**: Uploads a file (image or text) to Google's content-push service.
- **Request**: Path to a local file.
- **Response**: `*UploadedFile` containing a `ResourceID` used in generation requests.

---

## Authentication & Security

The project uses a sophisticated authentication flow to maintain access to the Gemini Web API.

### Authentication Flow
1.  **Cookie Acquisition**: Cookies (`__Secure-1PSID` and `__Secure-1PSIDTS`) are extracted automatically from local browser stores (Chrome, Firefox, etc.) or imported via `cookies.json`.
2.  **Token Extraction**: The client performs a GET request to `https://gemini.google.com/app` to extract the `SNlM0e` (session-specific) token from the HTML response using regex.
3.  **Request Signing**: Every POST request to the API includes the cookies in the header and the `SNlM0e` token as the `at` form parameter.

### Browser Fingerprinting
To avoid anti-bot detection, the project uses `bogdanfinn/tls-client` to:
- Emulate specific browser TLS fingerprints (Chrome 133).
- Maintain consistent User-Agent and related headers (`sec-ch-ua`, etc.).
- Handle redirects and cookies similarly to a real browser.

---

## External API Dependencies

The project consumes several undocumented or private Google APIs.

### Services Consumed

| Service Name | Purpose | Base URL |
| :--- | :--- | :--- |
| **Gemini Web UI** | Main interaction point | `https://gemini.google.com` |
| **Google Content Push** | File/Image uploads | `https://content-push.googleapis.com` |
| **Google Accounts** | Session maintenance | `https://accounts.google.com` |

### Endpoints Used

#### 1. Content Generation (Streaming)
- **Path**: `/_/BardChatUi/data/assistant.lamda.BardFrontendService/StreamGenerate`
- **Method**: `POST`
- **Format**: Form-encoded `f.req` (complex nested JSON arrays).
- **Response**: Chunked streaming JSON. Each chunk is prefixed with its length. The stream ends with a specific `[["e", ...]]` marker.

#### 2. Batch Execute (Gems & Management)
- **Path**: `/_/BardChatUi/data/batchexecute`
- **Method**: `POST`
- **Purpose**: Used for listing, creating, and deleting Gems (server-side personas).
- **RPC IDs**:
    - `Y6pS7b`: List Gems
    - `pLAt4c`: Create/Update Gem
    - `Xm7p3c`: Delete Gem

#### 3. File Upload
- **Path**: `/upload` (on `content-push.googleapis.com`)
- **Method**: `POST`
- **Headers**: Requires `Push-ID: 1033` and `Content-Type: multipart/form-data`.
- **Response**: Plain text Resource ID (e.g., `/contrib_service/ttl_1d/...`).

#### 4. Cookie Rotation
- **Path**: `/RotateCookies` (on `accounts.google.com`)
- **Method**: `POST`
- **Purpose**: Refreshes session cookies to prevent expiration during long sessions.
- **Constraint**: Rate-limited to once per minute.

---

## Error Handling & Resilience

### Error Patterns
- **Authentication Failure (401/403)**: The project intercepts these and triggers `RefreshFromBrowser()`, which re-scans local browsers for fresh cookies without user intervention.
- **Rate Limiting (1037/1060)**: Specific Gemini error codes are parsed. The CLI provides user-friendly guidance (e.g., "Usage limit exceeded" or "IP blocked").
- **Anti-Bot (Error 2)**: Detects when Google requests a manual verification (CAPTCHA) and provides instructions to the user.

### Resilience Mechanisms
- **Automatic Client Re-initialization**: If the client is idle and the session expires, it re-fetches the `SNlM0e` token before the next request.
- **Multipart Upload Robustness**: Handles both binary images and text files, automatically converting text snippets into "uploaded files" for better context handling by Gemini.
- **Stream Detection**: The client robustly parses the fragmented JSON response chunks and stops immediately upon receiving the end-of-stream marker to avoid hanging.

---

## Available Documentation

- **OpenAPI/Swagger**: None (Private API).
- **Internal Specs**:
    - `.ai/docs/api_analysis.md`: Detailed breakdown of response paths and GJSON selectors.
    - `internal/api/paths.go`: Centralized list of indices for parsing the nested array responses.
- **Documentation Quality**: High. The internal Go code is well-commented, and the `.ai/docs` directory contains extensive analysis of the reverse-engineered API structure.