# Vakula

[![Go Version](https://img.shields.io/badge/Go-1.26.2+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Vakula watches a task inbox, sends each prompt to an LLM agent (Gemini or OpenAI via [langchaingo](https://github.com/tmc/langchaingo)), and gives the agent two tools: **run Go code in a Docker sandbox** (`go_interpreter`) and **export multi-file projects to disk** (`project_exporter`). It is aimed at generating and validating small Go codebases autonomously.

The name comes from Slavic folklore - a blacksmith who sits at his forge and works. That felt right for a daemon that waits for tasks and builds things.

---

## How It Works

Each file dropped into `data/in/` triggers the agent loop:

1. The file body is sent as a natural-language prompt to the LLM
2. The agent decides which tool to call - and when:
   - **`go_interpreter`** - writes the generated Go code to a temp directory, then compiles and runs it inside an ephemeral Docker container
   - **`project_exporter`** - writes a structured multi-file project to `data/out/<taskID>/`
3. The agent iterates until it is satisfied with the output or exhausts its tool budget

### The Sandbox

The interesting part is not the agent - it is the execution boundary.

LLM-generated code cannot be trusted. Before any generated file touches disk, paths are validated against traversal attacks - both the naive `../` case and the subtler post-`filepath.Join` escape. Every execution happens inside a Docker container with:

- **Network disabled** - no outbound calls, no data exfiltration
- **512 MB memory cap** - no runaway allocations
- **Ephemeral temp directory** - wiped after execution regardless of outcome
- **Deferred container removal** - cleanup happens even if the run panics

The host filesystem is never directly exposed to generated code.

---

## Prerequisites

- **Go** 1.26.2+
- **Docker** running locally (uses `golang:1.26.2-alpine`; pulled on first run)
- **API key**: `GOOGLE_API_KEY` for Gemini, or `OPENAI_API_KEY` if you switch providers (see `internal/agent/factory.go`)

---

## Setup

```bash
cp .env.sample .env
# edit .env and set GOOGLE_API_KEY=...

mkdir -p data/in data/out
```

## Run

```bash
go run ./cmd
```

You should see `Vakula is at the forge. Press Ctrl+C to stop.`

Leave the process running while you add tasks. Shutdown is graceful - in-flight work is cancelled after the signal is received.

---

## Tasks

1. Put a **new file** under `data/in/`. The **basename without extension** becomes the task ID (used for `data/out/<taskID>/` on export).
2. The **file body** is the prompt. Markdown is supported and recommended - headings, lists, and code fences help structure longer specs.

The inbox watcher reacts to **create** events only, not in-place edits. Prepare tasks elsewhere, then **move or copy** them into `data/in/`. That is the most reliable way to enqueue work.

Sample tasks live in `data/sample-tasks/` - edit there, then move into `data/in/` when ready.

---

## Layout

| Path                    | Role                                          |
|-------------------------|-----------------------------------------------|
| `data/in/`              | Drop task files here (watched)                |
| `data/out/`             | Exported projects (`<taskID>/...`)            |
| `data/sample-tasks/`    | Sample prompts - move or copy into `data/in/` |
| `internal/agent/`       | LLM agent loop and tool definitions           |
| `internal/executor/`    | Docker sandbox and project exporter           |
| `internal/task/`        | Inbox watcher (fsnotify)                      |

---

## Configuration

Provider and model are configured in `cmd/main.go` via `agent.Config`. The default is Gemini with a Go-architect system prompt. Switch `Provider` to `"openai"` and set `ModelName` accordingly for OpenAI.

---

## Authors

- **VicDeo** - [GitHub](https://github.com/VicDeo) · [LinkedIn](https://linkedin.com/in/dubiniuk)

---

*Built on openSUSE Tumbleweed.*
