# Shrew: Ultra-lightweight Agent

Shrew is a powerful, zero-dependency AI agent that connects to any API through its skills paradigm and automates your command line. It's a single static binary written in Go that is configuration-oriented, managing sessions, skills, and secrets in a local database (`shrew.db`).

## Philosophy

- **Lightweight & Fast**: A single static binary written in Go with minimal dependencies, ensuring maximum portability and performance.
- **Model Agnostic**: Connect to any model provider (Gemini, OpenAI, Anthropic, etc.) or run locally using Ollama.
- **Security First (Vault)**: Sensitive tokens and keys are stored in a local vault. They are **never** sent to the models or stored in conversation history.
- **Agentic Execution**: The model can execute shell commands directly in your terminal, write code, and manage files.
- **Dynamic Skills System**: Teach Shrew new workflows by adding Markdown files to the `skills/` directory or by asking it to "learn" new documentation on the fly.

## Installation

The easiest way to install Shrew on a new machine is with the one-command installer:

```bash
curl -fsSL https://raw.githubusercontent.com/Masmedeam/shrew/main/install.sh | sh
```

## Usage

Simply run the agent to start an interactive session:
```bash
shrew
```

Shrew provides two interfaces simultaneously:
- **Terminal REPL**: Direct interaction in your shell.
- **Web UI**: A modern interface available at `http://localhost:8080`.

## Security & Vault

Shrew features a secure Vault system to handle sensitive information (API keys, bearer tokens) without exposing them to AI models.

### How it works:
1. **Store Secrets**: Add secrets via the Web UI (Vault tab) or the terminal.
2. **Safe Referencing**: Models use the `[[vault:KEY_NAME]]` placeholder in commands.
3. **Just-in-Time Injection**: Shrew resolves these placeholders only at the moment of execution on your host machine.

This ensures that your private keys are never part of the prompt context, protecting you from prompt injection leaks or model training data inclusion.

## Configuration

Configure Shrew via the Web UI "Vault > System Config" tab or by setting environment variables in a `.env` file.

The `SHREW_MODEL` variable defines the provider and model you want to use in the format `provider/model-name`.

### OpenAI Compatible API

```bash
SHREW_API_KEY=your_api_key
SHREW_API_URL=https://api.openai.com/v1/chat/completions
SHREW_MODEL=openai/gpt-4o
```

## License

This project is licensed under the MIT License. See the LICENSE file for details.
