# Invariants

These rules are enforced by tests and must not be changed by an evolving agent.

1. TCB services (`cas`, `journal`, `eval`, `vault`, `capability`) are mounted by the `tcb` fiber and cannot be replaced by a candidate plugin.
2. Every CAS object is blake3-addressed. `refs/active`, `refs/canary`, `refs/staging`, `refs/archive/*`, and `refs/models/<fp>/{active,canary}` are the only mutable pointers.
3. Disposing a fiber must reverse every effect it registered (services, event listeners, WASM modules).
4. Secrets stay in the vault. Proposers and WASM guests never receive raw keys.
5. Promotion is `eval.ShouldPromote`: held-in and held-out must not drop, and at least one split must improve. Safety failures are never promoted.
6. Held-out and transfer task identities are stored on the suite object and are not included in the proposer evidence bundle or trial logs.
7. Journal records are append-only hash-chained. External actions are recorded as intents and later committed.
8. Binary auto-update is a separate channel from harness refs. Agents cannot rotate updater keys.
9. Evolve writes `refs/canary` (and archive) by default. Checkout is the only operator path that moves `refs/active`, unless the operator passes an explicit `--promote`.
10. Loop instruction copy and declarative middleware are L1 (Harbor-gated). Loop topology (`max_turns`, compaction, plan mode, budget) and policy pack hashes remain L3.
11. Unsigned WASM cannot be checked out onto `refs/active`. Evolve never compiles WASM.
