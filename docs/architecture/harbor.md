# Harbor adapter

Yoyo tasks use the Harbor / Terminal-Bench layout:

```
task-id/
  instruction.md
  task.toml
  tests/test.sh
  tests/test.ps1
  tests/expect.toml   # optional; preferred over scripts when present
  environment/        # optional Dockerfile; verifier runs in docker when available
```

## Smoke vs sealed

`yoyo eval` runs the **smoke** seed: held-in `write-hello`, held-out `write-answer`, safety `no-escape`. Chat and local tests stay on this suite (`repeats=1`).

`yoyo eval --sealed` materializes the private catalog (20 held-in / 10 held-out / 5 transfer, `repeats=2`). Identities live on the suite object. Propose never receives held-out or transfer ids.

`yoyo eval --transfer` grades only the transfer split (post-canary migration check). It is not part of `ShouldPromote`.

## Agent entry (`harbor run -a yoyo`)

Inside a task container, Harbor should exec the same CLI the Go engine uses:

```
yoyo run --workspace /app "$(cat /instruction.md)"
```

`internal/eval.AgentArgs(workspace, instruction)` returns that argv. Wrappers live in `evals/yoyo-agent/`:

```
evals/yoyo-agent/run.sh /app instruction.md
evals/yoyo-agent/run.ps1 -Workspace /app -Instruction instruction.md
```

Then the Harbor (or Yoyo) verifier runs `tests/expect.toml` if present, else `tests/test.sh` / `tests/test.ps1`. If `environment/Dockerfile` exists and `docker` is on PATH, the verifier is executed in that image with the workspace mounted at `/app` and `--network=none`.

## Promotion

**ShouldPromote:** held-in and held-out must not drop; at least one split must improve; any `safety_fail` blocks. Evolve writes `refs/canary` by default. Checkout (human, Promote page, or `yoyo evolve --promote`) is the only path that moves `refs/active`.

The academic mapping (Self-Harness gate, Harbor-Index, why held-out stays secret) is in [../research/README.md](../research/README.md).
