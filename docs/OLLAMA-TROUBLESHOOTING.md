# Ollama Troubleshooting Guide

This guide helps resolve common issues when using Ollama with roborev.

## "Ollama server not reachable"

### Symptoms
- Error: `ollama server not reachable at http://localhost:11434`
- Reviews fail immediately without attempting to run

### Solutions

1. **Verify Ollama is running:**
   ```bash
   # Check if Ollama process is running
   ps aux | grep ollama

   # Start Ollama if not running
   ollama serve
   ```

2. **Check the Ollama server URL:**
   ```bash
   # Test connectivity
   curl http://localhost:11434/api/tags
   ```

   If this works, Ollama is running correctly.

3. **Verify configuration:**
   - Check `OLLAMA_HOST` environment variable: `echo $OLLAMA_HOST`
   - Check `ollama_base_url` in `~/.roborev/config.toml` or `.roborev.toml`
   - Priority: config TOML > `OLLAMA_HOST` env var > default `http://localhost:11434`

4. **Remote Ollama server:**
   If running Ollama on a different machine:
   ```toml
   # In ~/.roborev/config.toml
   ollama_base_url = "http://remote-server:11434"
   ```

## "Model not found"

### Symptoms
- Error: `model "qwen2.5-coder:latest" not found`
- Suggests running `ollama pull`

### Solutions

1. **List installed models:**
   ```bash
   ollama list
   ```

2. **Pull the required model:**
   ```bash
   ollama pull qwen2.5-coder:latest
   ```

3. **Use a different model:**
   ```toml
   # In .roborev.toml
   model = "llama3:latest"  # or any installed model
   ```

4. **Browse available models:**
   - Visit [ollama.ai/library](https://ollama.ai/library)
   - Search for code-optimized models

## "Request timed out"

### Symptoms
- Error: `ollama request timed out`
- Reviews start but never complete
- High CPU usage during review

### Solutions

1. **Use a smaller model:**
   Larger models take more time and resources.

   | Model Size | Speed | Quality | RAM Required |
   |------------|-------|---------|--------------|
   | `:7b` | Fast | Good | ~8 GB |
   | `:13b` | Medium | Better | ~16 GB |
   | `:32b` | Slow | Excellent | ~32 GB |
   | `:70b` | Very Slow | Best | ~64 GB |

   ```bash
   # Switch to a smaller variant
   ollama pull qwen2.5-coder:7b
   ```

   ```toml
   # In .roborev.toml
   model = "qwen2.5-coder:7b"
   ```

2. **Check system resources:**
   ```bash
   # Check available RAM
   free -h  # Linux
   vm_stat  # macOS

   # Check CPU usage
   top
   ```

3. **Adjust reasoning level:**
   Lower reasoning levels run faster:
   ```toml
   review_reasoning = "fast"  # instead of "thorough"
   ```

## "Poor review quality"

### Symptoms
- Reviews are vague or miss obvious issues
- Suggestions are not actionable
- Code analysis is shallow

### Solutions

1. **Try a different model:**

   **Code-optimized models (recommended):**
   - `qwen2.5-coder:latest` - Best for code, supports tool syntax
   - `deepseek-coder:latest` - Excellent code understanding
   - `codellama:latest` - Meta's code-specialized model

   **General-purpose models:**
   - `llama3:70b` - High quality, needs more RAM
   - `mixtral:latest` - Good balance of speed and quality

2. **Adjust reasoning level:**
   ```toml
   review_reasoning = "thorough"  # More careful analysis
   ```

   This uses lower temperature (0.3) for more focused, deterministic reviews.

3. **Use a larger parameter count:**
   ```bash
   ollama pull qwen2.5-coder:32b  # instead of :7b
   ```

4. **Add custom review guidelines:**
   ```toml
   review_guidelines = """
   Focus on:
   - Security vulnerabilities (SQL injection, XSS, etc.)
   - Error handling and edge cases
   - Performance bottlenecks
   - Code maintainability and readability
   """
   ```

## "Reviews are too slow"

### Symptoms
- Reviews take several minutes
- System becomes unresponsive
- Multiple reviews queue up

### Solutions

1. **Use a smaller model:**
   ```bash
   ollama pull qwen2.5-coder:7b
   ```

2. **Use fast reasoning mode:**
   ```toml
   review_reasoning = "fast"
   ```

3. **Check for other Ollama users:**
   ```bash
   # Ollama processes only one request at a time
   # Check if another user/process is using Ollama
   ollama ps
   ```

4. **Increase concurrency (if you have GPU):**
   Ollama can run multiple models with sufficient VRAM:
   ```bash
   # Set environment variable before starting Ollama
   export OLLAMA_NUM_PARALLEL=2
   ollama serve
   ```

## General Debugging

### Enable verbose logging

Check Ollama logs for detailed error messages:

**Linux (systemd):**
```bash
journalctl -u ollama -f
```

**macOS/Linux (manual start):**
```bash
# Start Ollama in foreground to see logs
ollama serve
```

**Check Ollama version:**
```bash
ollama --version
```

### Test Ollama directly

Verify Ollama works outside of roborev:

```bash
# Test chat
ollama run qwen2.5-coder:latest "Review this Python code: print('hello')"

# List models
ollama list

# Check server status
curl http://localhost:11434/api/tags
```

## Getting Help

If you're still experiencing issues:

1. **Check roborev issues:** [github.com/roborev-dev/roborev/issues](https://github.com/roborev-dev/roborev/issues)
2. **Check Ollama issues:** [github.com/ollama/ollama/issues](https://github.com/ollama/ollama/issues)
3. **Verify versions:**
   ```bash
   roborev --version
   ollama --version
   ```

## See Also

- [Ollama Documentation](https://github.com/ollama/ollama/tree/main/docs)
- [roborev Configuration Guide](https://roborev.io/configuration/)
- [Model Library](https://ollama.ai/library)
