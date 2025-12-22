# Product Guide

## Initial Concept
`geminiweb-go` is a comprehensive Go implementation and CLI tool designed to interact with the private Google Gemini Web API. Unlike the official Google AI SDK, this project specializes in emulating browser-like behavior to unlock advanced features available on the Gemini web interface, such as Gems management, advanced file uploads, and specific model behaviors like "Thinking/Reasoning."

## Target Audience
The primary audience is **developers building automation** on top of Gemini Web. These users require a robust and reliable way to interface with Gemini's advanced web features programmatically, bypassing the limitations of the official public SDK.

## Core Value Proposition
The most critical feature is **reliable, undetectable browser emulation**. The tool ensures that requests mimic a real browser (Chrome/Firefox) using advanced TLS fingerprinting and HTTP/2 behavior to avoid detection, CAPTCHAs, and blocking by Google's security measures.

## User Experience (UX)
The project prioritizes **Interactive Richness**. While it provides a library for developers, the CLI offers a full-featured **Terminal User Interface (TUI)** built with the Charm stack (Bubble Tea). It is designed to feel like a responsive, modern desktop application running directly in the terminal, offering a seamless chat experience.

## Scope & Constraints
*   **Private API Focus**: The project strictly adheres to the **unofficial private web API** (`gemini.google.com`). It does not integrate with the official, paid Google Vertex AI or Gemini Developer API.
*   **Browser-Based Auth**: Authentication relies on extracting and rotating session cookies from local browser installations, reinforcing the "user emulation" approach.