# Ollama Agent Support — Implementation Tasks

## Stage 1: Core Agent Implementation

### Task 1.1: Create `internal/agent/ollama.go` with agent structure
**Files:**
- Create `internal/agent/ollama.go`

**Code:**
- Define `OllamaAgent` struct with fields:
  - `BaseURL string` — Ollama server URL (default: "http://localhost:11434")
  - `Model string` — Model name (e.g., "qwen2.5-coder:latest")
  - `Reasoning ReasoningLevel` — Reasoning level
  - `Agentic bool` — Whether agentic mode is enabled
  - `HTTPClient *http.Client` — HTTP client for requests (configurable for testing)
- Add constructor `NewOllamaAgent(baseURL string) *OllamaAgent` that initializes with defaults
- Add method `Name() string` returning `"ollama"`

### Task 1.2: Implement agent interface methods (non-Review)
**Files:**
- `internal/agent/ollama.go`

**Code:**
- Implement `WithModel(model string) Agent` — returns new agent with model set
- Implement `WithReasoning(level ReasoningLevel) Agent` — returns new agent with reasoning level set
- Implement `WithAgentic(agentic bool) Agent` — returns new agent with agentic flag set
- All methods preserve other fields when creating copies

### Task 1.3: Implement HTTP request structure and `/api/generate` call
**Files:**
- `internal/agent/ollama.go`

**Code:**
- Define `ollamaGenerateRequest` struct with JSON tags:
  - `Model string` (json:"model")
  - `Prompt string` (json:"prompt")
  - `Stream bool` (json:"stream")
  - `Options map[string]interface{}` (json:"options,omitempty") — for temperature, top_p
- Define `ollamaGenerateResponse` struct with JSON tags:
  - `Response string` (json:"response")
  - `Done bool` (json:"done")
  - `Error string` (json:"error,omitempty")
- Add helper method `(a *OllamaAgent) buildRequest(prompt string) ollamaGenerateRequest` that:
  - Sets model (defaults to "qwen2.5-coder:latest" if empty)
  - Sets prompt
  - Sets stream to true
  - Maps reasoning level to temperature/top_p in Options
- Add helper method `(a *OllamaAgent) getHTTPClient() *http.Client` that:
  - Returns `a.HTTPClient` if set (for testing)
  - Otherwise returns `&http.Client{Timeout: 5 * time.Minute}`

### Task 1.4: Implement `Review()` method with streaming response parser
**Files:**
- `internal/agent/ollama.go`

**Code:**
- Implement `Review(ctx context.Context, repoPath, commitSHA, prompt string, output io.Writer) (string, error)`:
  1. Build request using `buildRequest(prompt)`
  2. Marshal request to JSON
  3. Create POST request to `{baseURL}/api/generate`
  4. Set `Content-Type: application/json` header
  5. Execute request with context
  6. Check response status code:
     - 404 → return error "model not found" with helpful message
     - 500 → return error with server error details
     - Other non-200 → return generic error
  7. Parse NDJSON stream:
     - Use `bufio.Scanner` to read line-by-line
     - Unmarshal each line into `ollamaGenerateResponse`
     - Accumulate `response` text
     - If `output` writer provided, write each chunk
     - Stop when `done: true` received
  8. Return accumulated response text
- Add helper method `(a *OllamaAgent) parseStream(reader io.Reader, output io.Writer) (string, error)`

### Task 1.5: Implement error handling and classification
**Files:**
- `internal/agent/ollama.go`

**Code:**
- Add helper method `(a *OllamaAgent) classifyError(err error, statusCode int, model string) error`:
  - If `errors.Is(err, context.DeadlineExceeded)`: return "Ollama request timed out..."
  - If connection refused (check error string): return "Ollama server not reachable at {url}. Is Ollama running? (Start with: `ollama serve`)"
  - If statusCode == 404: return "Model {model} not found. Pull it with: `ollama pull {model}`"
  - If statusCode >= 500: return "Ollama server error: {details}"
  - Otherwise: return original error with context
- Use this helper in `Review()` method error handling

### Task 1.6: Register agent in `init()`
**Files:**
- `internal/agent/ollama.go`

**Code:**
- Add `init()` function that calls `Register(NewOllamaAgent(""))`

### Task 1.7: Create unit tests with mocked HTTP server
**Files:**
- Create `internal/agent/ollama_test.go`

**Tests:**
- `TestOllamaAgent_Name`: verify Name() returns "ollama"
- `TestOllamaAgent_WithModel`: verify model is set correctly
- `TestOllamaAgent_WithReasoning`: verify reasoning level is preserved
- `TestOllamaAgent_WithAgentic`: verify agentic flag is set
- `TestOllamaAgent_Review_Success`: mock successful streaming response, verify accumulated output
- `TestOllamaAgent_Review_ModelNotFound`: mock 404 response, verify error message includes `ollama pull`
- `TestOllamaAgent_Review_ServerUnreachable`: mock connection refused, verify error message includes `ollama serve`
- `TestOllamaAgent_Review_Timeout`: use short context timeout, verify timeout error
- `TestOllamaAgent_Review_EmptyResponse`: mock response with done=true but no text, verify "No review output generated"
- `TestOllamaAgent_Review_MalformedJSON`: mock stream with invalid JSON line, verify it's skipped gracefully

**Dependencies:**
- Use `httptest.NewServer` for mocking HTTP responses

---

## Stage 2: Configuration & Model Selection

### Task 2.1: Add Ollama config fields to `Config` struct
**Files:**
- `internal/config/config.go`

**Code:**
- Add field `OllamaBaseURL string` with toml tag `toml:"ollama_base_url"` to `Config` struct (around line 80)
- Add field `OllamaCmd string` with toml tag `toml:"ollama_cmd"` to `Config` struct (for future CLI support)

### Task 2.2: Add base URL resolution logic
**Files:**
- `internal/agent/ollama.go`

**Code:**
- Add function `ResolveOllamaBaseURL(cfg *Config) string`:
  1. If `cfg != nil` and `cfg.OllamaBaseURL != ""`: return `cfg.OllamaBaseURL`
  2. If env var `OLLAMA_HOST` is set: return its value
  3. Otherwise return default `"http://localhost:11434"`
- Update `NewOllamaAgent(baseURL string)` to accept empty string and resolve via this logic (requires config injection or env var check)
- Consider: add `NewOllamaAgentWithConfig(cfg *Config) *OllamaAgent` constructor that uses resolution

### Task 2.3: Update agent factory to support Ollama config
**Files:**
- `internal/agent/agent.go` (or create helper in config package)

**Code:**
- Consider where agent instantiation happens (likely in daemon/worker)
- Ensure when "ollama" agent is resolved, it uses the configured base URL
- May need to pass config through to agent creation

### Task 2.4: Add model name validation
**Files:**
- `internal/agent/ollama.go`

**Code:**
- Add helper `validateOllamaModel(model string) error`:
  - Check model is not empty
  - Optionally: validate format (alphanumeric + `:`, `-`, `_`, `.`)
  - Return descriptive error if invalid
- Call this in `Review()` before making HTTP request

### Task 2.5: Test config loading with Ollama settings
**Files:**
- `internal/config/config_test.go`

**Tests:**
- `TestLoadConfig_OllamaBaseURL`: verify TOML with `ollama_base_url` loads correctly
- `TestResolveOllamaBaseURL_EnvVar`: verify `OLLAMA_HOST` env var is respected
- `TestResolveOllamaBaseURL_Default`: verify default URL when nothing configured

### Task 2.6: Update documentation with configuration examples
**Files:**
- `README.md`
- Create `docs/OLLAMA.md` (optional, for detailed Ollama setup)

**Documentation:**
- Add Ollama to list of supported agents in README
- Add example global config with `ollama_base_url`
- Add example per-repo config with Ollama agent
- Document model selection (how to find available models: `ollama list`)

---

## Stage 3: Advanced Features & Polish

### Task 3.1: Implement reasoning level to temperature/top_p mapping
**Files:**
- `internal/agent/ollama.go`

**Code:**
- Update `buildRequest()` method to set options based on reasoning level:
  - `ReasoningThorough`: `{"temperature": 0.3, "top_p": 0.9}`
  - `ReasoningStandard`: `{"temperature": 0.7, "top_p": 0.95}`
  - `ReasoningFast`: `{"temperature": 1.0, "top_p": 1.0}`
- Add comment explaining the mapping rationale

### Task 3.2: Add agentic mode prompt modifications
**Files:**
- `internal/agent/ollama.go`

**Code:**
- Add method `(a *OllamaAgent) augmentPromptForAgentic(prompt string) string`:
  - If agentic mode disabled, return prompt unchanged
  - If enabled, append tool descriptions (basic format that Qwen/similar models understand)
  - Example: Append JSON schema for read_file, write_file, run_command tools
- Call this in `Review()` before building request
- Add comment: "Note: Tool support depends on the model. Models like Qwen 2.5+ understand this syntax."

### Task 3.3: Implement streaming progress output
**Files:**
- `internal/agent/ollama.go`

**Code:**
- In `parseStream()`, when `output` writer is non-nil:
  - Write each response chunk as it arrives: `output.Write([]byte(chunk))`
  - Use `newSyncWriter(output)` for thread-safety (already exists in agent.go)
- Test that progress is streamed during review

### Task 3.4: Add health check helper
**Files:**
- `internal/agent/ollama.go`

**Code:**
- Add method `(a *OllamaAgent) checkHealth(ctx context.Context) error`:
  - Make GET request to `{baseURL}/api/tags`
  - Return nil if 200 OK
  - Return descriptive error if connection fails or non-200 status
- Optionally call this at start of `Review()` for faster failure feedback
- Consider: cache result with TTL (or keep simple with no caching)

### Task 3.5: Comprehensive error messages with next steps
**Files:**
- `internal/agent/ollama.go`

**Code:**
- Enhance `classifyError()` to include actionable next steps in every error:
  - Connection refused: "Is Ollama running? Start with: `ollama serve` in another terminal."
  - Model not found: "Pull the model first: `ollama pull {model}`. List available models: `ollama list`."
  - Timeout: "Request timed out. Try: 1) Use a smaller/faster model, 2) Increase timeout in config."
  - Server error: "Ollama server returned error: {msg}. Check Ollama logs: `journalctl -u ollama` or console output."

### Task 3.6: Integration tests with mock server
**Files:**
- `internal/agent/ollama_test.go`

**Tests:**
- `TestOllamaAgent_Integration_FullReview`: simulate full review with multi-line streaming response
- `TestOllamaAgent_Integration_ReasoningLevels`: verify temperature/top_p values for each level
- `TestOllamaAgent_Integration_AgenticMode`: verify prompt augmentation when agentic=true
- `TestOllamaAgent_Integration_StreamingOutput`: verify chunks are written to output writer in real-time

### Task 3.7: Update agent documentation
**Files:**
- `README.md`
- `AGENTS.md` (if exists, otherwise create)

**Documentation:**
- Add Ollama section to agents list
- Document supported models (link to Ollama model library)
- Add setup instructions:
  1. Install Ollama: https://ollama.ai/download
  2. Start server: `ollama serve`
  3. Pull a model: `ollama pull qwen2.5-coder:latest`
  4. Configure roborev: `default_agent = "ollama"`
- Add troubleshooting section:
  - "Server not reachable" → check `ollama serve` is running
  - "Model not found" → run `ollama pull {model}`
  - Performance tips: use smaller models for faster reviews (e.g., `:7b` vs `:70b`)

---

## Stage 4: CLI & User Experience

### Task 4.1: Add Ollama to agent availability check
**Files:**
- `internal/agent/agent.go`

**Code:**
- Update `IsAvailable(name string) bool` to handle Ollama:
  - For "ollama", check if base URL is reachable via health check
  - Use short timeout (1-2 seconds) for availability check
  - Return true if server responds, false otherwise
- Note: May want to make this async or cached to avoid slow `GetAvailable()` calls

### Task 4.2: Add Ollama to fallback list in `GetAvailable()`
**Files:**
- `internal/agent/agent.go`

**Code:**
- Add "ollama" to the fallback list in `GetAvailable()` (line ~165)
- Position: after "opencode", before "cursor" (priority: codex > claude > gemini > copilot > opencode > ollama > cursor > droid)
- Update error message to include "ollama" in installation suggestions

### Task 4.3: Add `roborev agent check` command for diagnostics
**Files:**
- `cmd/roborev/main.go` (or relevant command file)

**Code:**
- Add new subcommand `roborev agent check [agent-name]`:
  - If agent-name is "ollama" (or empty and ollama is configured):
    - Check base URL reachability
    - Call `/api/tags` to list available models
    - Report status: "✓ Ollama available at {url}" or "✗ Ollama not reachable"
    - List installed models
  - For other agents: check if command is in PATH

### Task 4.4: Add `--ollama-url` CLI flag for ad-hoc override
**Files:**
- `cmd/roborev/main.go` (review command flags)

**Code:**
- Add flag `--ollama-url` to `review` command (and other relevant commands)
- Pass this value through to agent creation
- Document in `--help` output

### Task 4.5: Create example `.roborev.toml` with Ollama config
**Files:**
- Create `examples/roborev.toml.ollama` (or add section to existing example)

**Config:**
```toml
# Example configuration for using Ollama
agent = "ollama"
model = "qwen2.5-coder:latest"

# If Ollama is running on a different host
# ollama_base_url = "http://remote-server:11434"

# Reasoning level: fast, standard, thorough
review_reasoning = "standard"
```

### Task 4.6: Create troubleshooting guide
**Files:**
- Create `docs/OLLAMA-TROUBLESHOOTING.md`

**Documentation:**
- **"Ollama server not reachable"**:
  - Verify `ollama serve` is running
  - Check `OLLAMA_HOST` env var or config
  - Test with `curl http://localhost:11434/api/tags`
- **"Model not found"**:
  - List installed models: `ollama list`
  - Pull missing model: `ollama pull {model}`
- **"Request timed out"**:
  - Try smaller model (e.g., `:7b` instead of `:70b`)
  - Check system resources (RAM, CPU)
  - Increase timeout in config (future: add timeout config)
- **"Poor review quality"**:
  - Try different model (qwen2.5-coder, deepseek-coder, etc.)
  - Adjust reasoning level
  - Use larger parameter count (e.g., `:32b` vs `:7b`)

### Task 4.7: Update main README with Ollama quick start
**Files:**
- `README.md`

**Documentation:**
- Add "Quick Start with Ollama" section:
```bash
# Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Start Ollama server
ollama serve &

# Pull a code review model
ollama pull qwen2.5-coder:latest

# Configure roborev
echo 'default_agent = "ollama"' >> ~/.roborev/config.toml

# Review your code
roborev review HEAD
```

---

## Summary

**Stage 1** creates the minimal working agent (can complete a review).
**Stage 2** adds configuration flexibility (base URL, model selection).
**Stage 3** brings it to feature parity with other agents (reasoning, agentic, streaming, errors).
**Stage 4** polishes the user experience (diagnostics, examples, troubleshooting).

Each stage produces a working, testable system. Stages can be implemented sequentially or Stage 1-2 can be combined for faster initial delivery.

## Implementation Notes

- Tasks are ordered to minimize dependencies between steps
- Each task is self-contained and testable
- Tests should be written alongside implementation (TDD encouraged)
- Stage 1 can be completed in a single focused session (2-4 hours)
- Stages 2-4 can each be completed in 1-2 hours
- Total estimated time: 6-10 hours for full implementation
