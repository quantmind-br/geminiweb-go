# geminiweb (Go)

A high-performance CLI for interacting with Google Gemini via the web API. Built in Go with browser-like TLS fingerprinting for reliable authentication.

## Features

- **Interactive Chat** - Full-featured TUI with markdown rendering
- **Single Queries** - Quick questions from command line, files, or stdin
- **Multiple Models** - Support for Gemini 2.5 Flash, 2.5 Pro, and 3.0 Pro
- **Cookie Authentication** - Uses browser cookies for authentication
- **Auto Cookie Extraction** - Extract cookies directly from browsers (Chrome, Firefox, Edge, etc.)
- **Auto Cookie Refresh** - Background token rotation and automatic browser refresh on auth failure
- **TLS Fingerprinting** - Chrome-like TLS profile to avoid detection

## Installation

### Build from source

```bash
# Clone the repository
cd geminiweb-go

# Build
make build

# Install to GOPATH/bin
make install
```

### Requirements

- Go 1.23+
- CGO enabled (for TLS client)

## Usage

### Setup

**Option 1: Auto-extract from browser (recommended)**

```bash
# Auto-detect browser and extract cookies
geminiweb auto-login

# Or specify browser
geminiweb auto-login -b firefox
geminiweb auto-login -b chrome
```

**Option 2: Manual import**

1. Export cookies from your browser after logging into [gemini.google.com](https://gemini.google.com)
2. Import cookies:

```bash
geminiweb import-cookies ~/cookies.json
```

> **Note:** For auto-login, close the browser first to avoid database lock errors.

### Interactive Chat

```bash
geminiweb chat
```

### Using Gems (Server-side Personas)

Gems are custom personas stored on Google's servers that sync across devices.

```bash
# Browse and manage gems interactively
geminiweb gems list

# Start chat with a specific gem
geminiweb chat --gem "Code Helper"
geminiweb chat -g code

# In the gems list, press 'c' to start chat with the selected gem

# During chat, type /gems to switch gems without leaving
```

Keyboard shortcuts in gems list:
- `c` - Start chat with selected gem
- `y` - Copy gem ID to clipboard
- `/` - Search gems
- `Enter` - View gem details

### Single Query

```bash
# Direct prompt
geminiweb "What is Go?"

# From file
geminiweb -f prompt.md

# From stdin
cat prompt.md | geminiweb

# Save to file
geminiweb "Hello" -o response.md
```

### Configuration

```bash
geminiweb config
```

Available settings:
- **default_model**: gemini-2.5-flash, gemini-2.5-pro, gemini-3.0-pro
- **auto_close**: Auto-close connections after inactivity
- **verbose**: Enable debug logging

### Model Selection

```bash
# Use specific model for a query
geminiweb -m gemini-3.0-pro "Explain quantum computing"

# In chat mode
geminiweb chat -m gemini-2.5-pro
```

### Auto Cookie Refresh

Enable automatic cookie refresh from browser when authentication fails:

```bash
# Auto-detect browser for refresh
geminiweb --browser-refresh=auto "Hello"
geminiweb --browser-refresh=auto chat

# Use specific browser
geminiweb --browser-refresh=firefox "Hello"
geminiweb --browser-refresh=chrome chat
```

Supported browsers: `chrome`, `chromium`, `firefox`, `edge`, `opera`, `auto`

## Cookie Format

The cookies file supports two formats:

**Browser export format (list):**
```json
[
  {"name": "__Secure-1PSID", "value": "..."},
  {"name": "__Secure-1PSIDTS", "value": "..."}
]
```

**Simple format (dict):**
```json
{
  "__Secure-1PSID": "...",
  "__Secure-1PSIDTS": "..."
}
```

Required: `__Secure-1PSID`
Optional: `__Secure-1PSIDTS`

## Project Structure

```
geminiweb-go/
├── cmd/geminiweb/       # Entry point
├── internal/
│   ├── api/             # API client (TLS, token, generation, browser refresh)
│   ├── browser/         # Browser cookie extraction (Chrome, Firefox, Edge, etc.)
│   ├── commands/        # CLI commands (Cobra)
│   ├── config/          # Configuration and cookies
│   ├── errors/          # Custom error types
│   ├── history/         # Conversation history persistence
│   ├── models/          # Data types and constants
│   └── tui/             # Terminal UI (Bubble Tea)
├── Makefile
└── go.mod
```

## Development

```bash
# Download dependencies
make deps

# Build for development (faster)
make build-dev

# Run with arguments
make run ARGS="chat"

# Run tests
make test

# Format code
make fmt
```

## License

MIT
