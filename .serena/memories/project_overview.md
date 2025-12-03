# Project Overview

## Purpose

**geminiweb-go** is a high-performance CLI for interacting with Google Gemini via the web API (gemini.google.com). It uses cookie-based authentication (not API keys) and browser-like TLS fingerprinting (Chrome 133 profile) to communicate reliably with Gemini's web interface.

## Tech Stack

- **Language**: Go 1.23+
- **HTTP/TLS**: `bogdanfinn/tls-client` (Chrome 133 fingerprinting), `bogdanfinn/fhttp`
- **CLI Framework**: `spf13/cobra`
- **TUI Framework**: `charmbracelet/bubbletea`, `charmbracelet/bubbles`, `charmbracelet/lipgloss`, `charmbracelet/glamour`
- **JSON Parsing**: `tidwall/gjson`
- **Browser Cookies**: `browserutils/kooky` (cross-platform extraction with decryption)

## Key Features

- Cookie-based authentication (requires `__Secure-1PSID` from browser)
- TLS fingerprinting with Chrome 133 profile to appear as real browser
- Auto cookie rotation in background (default 9 min interval)
- Browser cookie auto-refresh on auth failure (rate-limited to 30s)
- Interactive chat mode with TUI and Glamour markdown rendering
- Conversation history persistence with auto-save
- Multiple model support (Flash 2.5, Pro 3.0)
- Local persona management (custom GPT-like personas)
- Gems support (Google's server-side personas)
- History selector for switching between conversations

## Configuration

Configuration and cookies are stored in `~/.geminiweb/`.
