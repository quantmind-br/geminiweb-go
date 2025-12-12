# GeminiWeb-Go

## Project Overview

**GeminiWeb-Go** is a powerful CLI application for interacting with Google Gemini's web API using cookie-based authentication. The application provides both single-query execution and interactive chat capabilities with advanced features like file uploads, conversation history management, and server-side persona (Gems) support.

**Purpose and Main Functionality:**
- Command-line interface for Google Gemini AI interactions
- Browser-based authentication without API keys
- Interactive chat sessions with conversation context
- File upload support for images and documents
- Conversation history management and export
- Server-side persona (Gems) management

**Key Features and Capabilities:**
- Multi-browser cookie extraction (Chrome, Firefox, Edge, Chromium, Opera)
- Chrome 133 TLS fingerprinting for anti-bot evasion
- Interactive TUI with Bubble Tea framework
- Markdown rendering with Glamour
- JSON-based configuration and history persistence
- Automatic cookie rotation and token refresh
- File upload support (images up to 20MB, text files up to 50MB)
- Conversation search, filtering, and export capabilities

**Intended Use Cases:**
- Developers integrating Gemini into workflows
- Power users preferring CLI over web interface
- Automated scripting and batch processing
- Conversation analysis and management
- Cross-platform Gemini access without browser dependencies

## Table of Contents

- [Architecture](#architecture)
- [C4 Model Architecture](#c4-model-architecture)
- [Repository Structure](#repository-structure)
- [Dependencies and Integration](#dependencies-and-integration)
- [API Documentation](#api-documentation)
- [Development Notes](#development-notes)
- [Known Issues and Limitations](#known-issues-and-limitations)
- [Additional Documentation](#additional-documentation)

## Architecture

### High-Level Architecture Overview

GeminiWeb-Go follows a hexagonal architecture pattern with clear separation of concerns. The core API client serves as the domain component, surrounded by adapters for browser integration, CLI commands, and user interfaces.

### Technology Stack and Frameworks

- **Language**: Go 1.23.10
- **CLI Framework**: Cobra for command structure and routing
- **TUI Framework**: Bubble Tea with Bubbles components
- **HTTP Client**: bogdanfinn/tls-client with Chrome 133 fingerprinting
- **Browser Integration**: browserutils/kooky for cross-platform cookie extraction
- **Markdown Rendering**: Charm Glamour with custom themes
- **Data Processing**: tidwall/gjson for JSON parsing

### Component Relationships

```mermaid
graph TB
    subgraph "CLI Layer"
        CLI[Cobra CLI Router]
        Root[Root Command]
        Chat[Chat Command]
        Config[Config Command]
        History[History Command]
        Gems[Gems Command]
    end
    
    subgraph "Core Services"
        API[GeminiClient API]
        Browser[Cookie Extractor]
        HistoryStore[History Store]
        Render[Render Service]
    end
    
    subgraph "Data Layer"
        ConfigFiles[JSON Config]
        CookieFiles[Cookie Storage]
        HistoryFiles[Conversation History]
    end
    
    subgraph "External Services"
        GeminiAPI[Google Gemini Web API]
        Browsers[Local Browsers]
    end
    
    CLI -.→ API
    Root -.→ API
    Chat -.→ API
    Config -.→ ConfigFiles
    History -.→ HistoryStore
    Gems -.→ API
    
    API -.→ Browser
    API -.→ GeminiAPI
    Browser -.→ Browsers
    HistoryStore -.→ HistoryFiles
    Render -.→ ConfigFiles
    
    API -.→ CookieFiles
    Browser -.→ CookieFiles
    
    classDef cliLayer fill:#e1f5fe
    classDef coreLayer fill:#f3e5f5
    classDef dataLayer fill:#e8f5e8
    classDef externalLayer fill:#fff3e0
    
    class CLI,Root,Chat,Config,History,Gems cliLayer
    class API,Browser,HistoryStore,Render coreLayer
    class ConfigFiles,CookieFiles,HistoryFiles dataLayer
    class GeminiAPI,Browsers externalLayer
```

### Key Design Patterns

- **Dependency Injection**: Functional options pattern for client configuration
- **Repository Pattern**: JSON-based storage for configuration and history
- **Command Pattern**: Cobra CLI commands encapsulating specific operations
- **Observer Pattern**: Bubble Tea's message-driven TUI architecture
- **Strategy Pattern**: Multiple browser extraction strategies with fallback
- **Factory Pattern**: Client and model factories with configurable options

## C4 Model Architecture

### Context Diagram

</arg_value>
</tool_call>