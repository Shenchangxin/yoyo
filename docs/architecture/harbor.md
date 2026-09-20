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

The agent worktree does **not** receive `tests/`. Verifiers read the task source directory (or Docker `/grader:ro`). That is a safety gate, not an experiment.

## Smoke vs sealed

`yoyo eval` runs the **smoke** seed: held-in `write-hello`, held-out `write-answer`, safety `no-escape`. Chat and local tests stay on this suite (`repeats=1`). Production numbers use the live model (`--solver live`, the default), not the heuristic fixture.

`yoyo eval --sealed` materializes the private catalog as **12 evolve-in / 8 promote-in / 10 held-out / 5 transfer** (`repeats=2`). `ShouldPromote` sees promote-in + held-out. Mine/Propose and inner trials see only evolve-in. Identities stay on the suite object. Propose never receives held-out, transfer, or Harbor-Index ids.

The sealed tasks are isomorphic file writes. Splitting evolve-set from promote-set is **protocol-correct and low-signal**. Do not treat a sealed canary as a Terminal-Bench result.

`yoyo eval --transfer` grades only the sealed transfer split (post-canary migration check). It is not part of `ShouldPromote`.

`yoyo eval --index` grades the Harbor-Index **stand-in** subset as transfer-only. Real Index adapters may replace the files; the 82-task Index set is not in desktop CI, and those ids never enter Propose.

`yoyo eval --behavior` runs the cheap behavior probes (claim-complete, no-touch-tests, must-verify, no-invent-path). They belong in the evolve lab and CI, not the default 89-task promote gate.

`yoyo eval --best N --sealed` repeats the same suite the evolve lab uses. Spend (`tokens_in/out`, `usd`, `wall_ms`) is on every `RunReport`.

## Matched-budget lab

`yoyo evolve --baselines N` (and the Evolve lab checkbox) runs evolve against **best-of-N**, **IID random L1**, and **SCS** under the same model, suite, and `--max-usd` / `--max-wall` cap. If evolve cannot beat BoN and IID on held-out, freeze the search structure. Do not raise K.

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

Then the Harbor (or Yoyo) verifier runs `tests/expect.toml` if present, else `tests/test.sh` / `tests/test.ps1`. If `environment/Dockerfile` exists and `docker` is on PATH, the verifier is executed in that image with the workspace mounted at `/app`, tests mounted at `/grader:ro`, and `--network=none`.

## Promotion

**ShouldPromote:** held-in and held-out must not drop; at least one split must improve; any `safety_fail` blocks. Evolve writes `refs/canary` by default. Checkout (human, Promote page, or `yoyo evolve --promote`) is the only path that moves `refs/active`. Predicted fixes with `hit==0` do not enter canary. Quality gates (playbook cap, held-in n-gram leak, repo-overview, fluff skills, held-out ids) reject before Harbor.

The academic mapping (Self-Harness gate, Harbor-Index, why held-out stays secret) is in [../research/README.md](../research/README.md).
