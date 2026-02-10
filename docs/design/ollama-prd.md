# Ollama Agent Support — Product Requirements Document

## Overview

Add an Ollama agent to roborev that connects to local or remote Ollama servers for code reviews. Ollama provides access to open-source models (Llama, Qwen, Mistral, etc.) running locally, giving users a free, private, and customizable alternative to cloud-based agents. The implementation will follow existing agent patterns while being clean, spare, and fully featured.

## Goals and Non-Goals

**Goals:**
- Support HTTP communication with Ollama servers (default: `http://localhost:11434`)
- Allow users to specify any Ollama model via config (e.g., `qwen2.5:latest`, `llama3:70b`)
- Stream response output for real-time progress visibility
- Handle common failure modes gracefully (server unreachable, model not found, timeout)
- Support both read-only and agentic review modes
- Match the agent interface fully (reasoning levels, model selection, agentic mode)

**Non-Goals:**
- Installing or managing Ollama server (user's responsibility)
- Pull/download models automatically (user pre-installs models via `ollama pull`)
- Implementing custom tool-calling beyond what Ollama + model provide natively
- Web UI for Ollama configuration (use TOML config)
- Retry logic or request queuing (keep it simple)

## Technical Decisions

**HTTP Client:**
- Use Go stdlib `net/http` (no external deps)
- Default base URL: `http://localhost:11434` (Ollama's default)
- Configurable via:
  1. `OLLAMA_HOST` environment variable (Ollama standard)
  2. Config TOML: `ollama_base_url = "http://remote-server:11434"`
  3. Explicit override in code (for testing)

**API Integration:**
- Use Ollama's `/api/generate` endpoint for streaming completions
- Request format: `{"model": "...", "prompt": "...", "stream": true}`
- Response format: NDJSON stream with `{"response": "...", "done": false}`
- Final message: `{"response": "...", "done": true}`

**Model Selection:**
- Model format: Ollama model names (e.g., `qwen2.5-coder:latest`, `llama3:70b`)
- Default model: `qwen2.5-coder:latest` (good for code, supports basic tool syntax)
- Configurable via standard config resolution (per-repo, global, CLI)

**Reasoning Level Support:**
- Ollama has no native reasoning level API
- Map reasoning levels to temperature/top_p parameters:
  - **thorough**: `temperature=0.3, top_p=0.9` (more focused)
  - **standard**: `temperature=0.7, top_p=0.95` (default)
  - **fast**: `temperature=1.0, top_p=1.0` (more creative, faster)

**Agentic Mode:**
- Ollama itself doesn't enforce read-only vs agentic modes
- The agent will include tool descriptions in the prompt when agentic mode is enabled (similar to how some models understand JSON function calling)
- Note in docs: "Agentic mode effectiveness depends on the loaded model. Models like Qwen 2.5+ support tool syntax; others may ignore it."

**Error Handling:**
- Check server health via `/api/tags` (lighter than `/api/version`)
- Classify errors:
  - **Connection refused**: "Ollama server not reachable at {url}. Is Ollama running? (Start with: `ollama serve`)"
  - **404 on /api/generate**: "Model {name} not found. Pull it with: `ollama pull {name}`"
  - **Timeout**: "Ollama request exceeded timeout. Try a smaller model or increase timeout."
  - **Invalid response**: "Unexpected response from Ollama: {details}"

**Dependencies:**
- Zero new external dependencies
- Uses stdlib only: `net/http`, `encoding/json`, `bufio`, `context`

## Design and Operation

### User Perspective

**Configuration (global `~/.roborev/config.toml`):**
```toml
default_agent = "ollama"
default_model = "qwen2.5-coder:latest"
ollama_base_url = "http://localhost:11434"  # optional, defaults to OLLAMA_HOST or http://localhost:11434
```

**Configuration (per-repo `.roborev.toml`):**
```toml
agent = "ollama"
model = "llama3:70b"  # override for this repo
```

**CLI usage:**
```bash
# Use Ollama with default model
roborev review HEAD --agent ollama

# Use specific model
roborev review HEAD --agent ollama --model qwen2.5-coder:latest

# Agentic mode (if allow_unsafe_agents is enabled)
roborev fix 123 --agent ollama --agentic
```

**What happens:**
1. User commits code, hook triggers review
2. Daemon worker picks up job, resolves agent to "ollama"
3. OllamaAgent connects to configured base URL
4. Sends prompt to `/api/generate` with streaming enabled
5. Parses NDJSON response, streams progress to job output
6. Returns completed review text when `done: true` received

### System Perspective

**Data Flow:**
```
Worker → OllamaAgent.Review()
  → HTTP POST to {baseURL}/api/generate
  → Stream NDJSON lines
  → Parse each line, accumulate response text
  → Stream progress to io.Writer (if provided)
  → Return full response on completion
```

**State Transitions:**
- No persistent state in the agent (stateless HTTP calls)
- Each `Review()` call is independent

**Concurrency:**
- Agent is thread-safe (no shared mutable state)
- Multiple workers can use Ollama concurrently (Ollama handles queueing)

**Error Handling Edge Cases:**
1. **Empty model name**: Return error "model name cannot be empty"
2. **Server returns empty response**: Return "No review output generated"
3. **Partial stream failure**: Return accumulated text + error context
4. **Context cancellation**: Stop streaming, return "review cancelled by user"
5. **Invalid JSON in stream**: Skip line, continue parsing (log warning)

### Failure Modes

| Failure | Detection | Handling |
|---------|-----------|----------|
| Ollama not running | Connection refused on first request | Clear error message with `ollama serve` hint |
| Model not installed | 404 from /api/generate | Error with `ollama pull {model}` command |
| Network timeout | Context deadline exceeded | Error mentioning timeout, suggest retry or smaller model |
| Malformed response | JSON parse error | Log line, skip, continue with remaining stream |
| Empty response | done=true but no text accumulated | Return "No review output generated" |
| Server error (500) | HTTP status code | Include server error message in failure output |

## Implementation Stages

### Stage 1: Core Agent Implementation (minimal, working)
**Deliverable:** A basic Ollama agent that can connect to a local server and complete a simple review.

- Create `internal/agent/ollama.go` with `OllamaAgent` struct
- Implement `Agent` interface methods: `Name()`, `Review()`, `WithModel()`, `WithReasoning()`, `WithAgentic()`
- Add HTTP client with `/api/generate` POST request
- Parse NDJSON streaming response
- Basic error handling (connection refused, 404, timeout)
- Register agent in `init()`
- Add basic unit tests with mocked HTTP responses

### Stage 2: Configuration & Model Selection
**Deliverable:** Users can configure Ollama base URL and select models via config.

- Add `OllamaBaseURL` field to `Config` struct
- Add base URL resolution logic (env var `OLLAMA_HOST` > config > default)
- Update model resolution to support Ollama model names
- Add validation for Ollama model name format
- Test config loading with Ollama settings
- Update docs with configuration examples

### Stage 3: Advanced Features & Polish
**Deliverable:** Full feature parity with other agents, production-ready.

- Implement reasoning level mapping (temperature/top_p parameters)
- Add agentic mode prompt modifications (tool descriptions)
- Implement streaming progress output via `io.Writer`
- Add health check helper for better error messages
- Comprehensive error classification and user-friendly messages
- Add integration tests with actual Ollama server (optional, skip in CI)
- Update `README.md` and `AGENTS.md` with Ollama documentation

### Stage 4: CLI & User Experience
**Deliverable:** Smooth user experience with helpful diagnostics.

- Add `roborev check` command to verify Ollama availability
- Include Ollama in agent fallback list in `GetAvailable()`
- Add `--ollama-url` CLI flag for ad-hoc base URL override
- Improve error messages with actionable next steps
- Add example `.roborev.toml` with Ollama config
- Create troubleshooting guide in docs

## Open Questions

- Should Ollama be in the fallback list for `GetAvailable()`, or only used when explicitly configured?
  - **Recommendation**: Include in fallback list but with low priority (after opencode). Makes it discoverable.

- Do you want health check caching (like PR #149 had), or keep it simpler with no caching?
  - **Recommendation**: No caching in v1. Keep it simple. Add caching later if performance becomes an issue.

- Any specific models you want as defaults besides `qwen2.5-coder:latest`?
  - **Recommendation**: `qwen2.5-coder:latest` is good. It's optimized for code, reasonably sized, and supports tool syntax.

## References

- PR #149: Previous Ollama implementation attempt (closed in favor of opencode integration)
- Ollama API docs: https://github.com/ollama/ollama/blob/main/docs/api.md
- Existing agents: `internal/agent/claude.go`, `internal/agent/codex.go`, `internal/agent/opencode.go`
