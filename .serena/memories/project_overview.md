# Project Overview

## Purpose

**geminiweb-go** is a high-performance CLI for interacting with Google Gemini via the web API (gemini.google.com). It uses cookie-based authentication (not API keys) and browser-like TLS fingerprinting (Chrome 120 profile) to communicate reliably with Gemini's web interface.

## Tech Stack

- **Language**: Go 1.23+
- **HTTP/TLS**: `bogdanfinn/tls-client` (Chrome fingerprinting), `bogdanfinn/fhttp`
- **CLI Framework**: `spf13/cobra`
- **TUI Framework**: `charmbracelet/bubbletea`, `charmbracelet/bubbles`, `charmbracelet/lipgloss`, `charmbracelet/glamour`
- **JSON Parsing**: `tidwall/gjson`

## Key Features

- Cookie-based authentication (requires `__Secure-1PSID` from browser)
- TLS fingerprinting with Chrome 120 profile to appear as real browser
- Auto cookie rotation in background (default 9 min interval)
- Interactive chat mode with TUI
- Conversation history persistence
- Multiple model support (Flash, Pro, 3.0 Pro)
- Persona management (custom GPT-like personas)

## Configuration

Configuration and cookies are stored in `~/.geminiweb/`.
