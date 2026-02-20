<div align="center">
  <img src="shrew.png" width="300" alt="Shrew Header Image">
</div>

# Shrew: The Universal CLI Agent

Shrew is a powerful, zero-dependency TUI agent that connects to any API and automates your command line. It's a single static binary written in Go that gives Large Language Models a rich, interactive terminal to perform real-world tasks.

While it feels like a sophisticated, modern application, its only dependency is the Bubble Tea TUI framework. The core logic remains dependency-free, ensuring maximum portability and performance.

## Features

- **Universal API Bridge**: Natively supports Gemini, OpenAI, and Ollama, but can connect to *any* API via its generic OpenAI-compatible layer or a custom shell command adapter.
- **Rich TUI**: An interactive, full-screen terminal interface powered by Bubble Tea that beautifully renders Markdown, thinking processes, and command executions.
- **Agentic Execution**: The model can execute shell commands directly in your terminal via `<run>` tags, allowing it to write code, manage files, and run tests.
- **Extensible Skills System**: Teach Shrew new workflows and give it long-term knowledge by adding simple Markdown files to the `skills/` directory.
- **Zero Core Dependencies**: A single static binary with no required runtimes like Node.js or Python.

## Installation

The easiest way to install Shrew on a new machine is with the one-command installer. This script will automatically detect your OS and architecture, download the latest binary from GitHub Releases, and install it.

```bash
curl -fsSL https://raw.githubusercontent.com/Masmedeam/shrew/main/install.sh | sh
```

### Other Methods

#### From Source
```bash
git clone https://github.com/Masmedeam/shrew.git
cd shrew
go build -o shrew main.go
sudo mv shrew /usr/local/bin/
```

#### Go Install
```bash
go install github.com/Masmedeam/shrew@latest
```

## Configuration

Configure Shrew by creating a `.env` file in your project root or by setting environment variables.

### Google Gemini (Default)
```bash
GEMINI_API_KEY=your_api_key
```

### OpenAI or any Compatible API
Use this for providers like DeepSeek, MiniMax, Groq, etc.
```bash
SHREW_PROVIDER=openai
OPENAI_API_KEY=your_api_key
SHREW_API_URL=https://api.example.com/v1/chat/completions # The provider's URL
SHREW_MODEL=model-name
```

### Custom Command Bridge
For any other API, you can provide a shell command that takes the conversation history as JSON via stdin and returns the model's response via stdout.
```bash
SHREW_PROVIDER=cmd
SHREW_COMMAND="python3 ./my_api_adapter.py"
```

## Usage

Simply run the agent to start an interactive session:
```bash
shrew
```
You can then issue high-level instructions, and Shrew will use its skills, tools, and reasoning to accomplish the task.

## License

This project is licensed under the MIT License. See the LICENSE file for details.
