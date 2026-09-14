# Harbor adapter

Yoyo tasks use the Harbor / Terminal-Bench layout:

```
task-id/
  instruction.md
  task.toml
  tests/test.sh
  tests/test.ps1
  environment/   # optional Dockerfile
```

Run the bundled smoke suite:

```
yoyo eval
```

To evaluate Yoyo as a Harbor agent, invoke the CLI inside the task container:

```
yoyo run --workspace /app "$(cat /instruction.md)"
```

then execute the task verifier. A dedicated `harbor run -a yoyo` wrapper can call the same `internal/eval` engine.
