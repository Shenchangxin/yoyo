# Yoyo

Self-harnessing local agent workstation. Go kernel, React/TS desktop shell (Wails v3), versioned harness materials, trajectory tracing, Harbor-style eval, and a gated Self-Harness loop.

## Quick start

```bash
go test ./...
go run ./cmd/yoyo init
go run ./cmd/yoyo eval
go run ./cmd/yoyo evolve
go run ./cmd/yoyo serve --addr 127.0.0.1:3080
```

Desktop (requires [Wails v3](https://v3.wails.io)):

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@latest
cd frontend && npm install && cd ..
wails3 task dev
```

API keys: `YOYO_API_KEY` or `OPENAI_API_KEY`. Home directory: `YOYO_HOME` or `~/.yoyo`.

## Layout

- `cmd/yoyo` — CLI (`run`, `eval`, `evolve`, `harness`, `serve`)
- `main.go` — Wails desktop entry
- `internal/kernel` — load/unload fibers with revertible effects
- `internal/artifact` — CAS + git-like refs + Agent Skills
- `internal/eval` — sealed splits and promotion gate
- `internal/evolve` — weakness mining, L1 proposals, WASM admission, DGM archive
- `evals/` — Harbor-format smoke tasks
- `docs/architecture` — invariants and threat model

Harness snapshots and binary updates are separate channels. The agent cannot modify the evaluator, journal, vault, or updater keys.
