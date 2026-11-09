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
12. Harbor eval never loads specialist profiles, Pages pins, or skill packs. Pages and profiles are application objects on the chat loop, not harness topology.
13. Page bodies, mail, tool output, and voice transcripts are untrusted source data. They must not enter identity or rewrite the playbook.
14. `page.save` / `page.edit` promotions cannot stick as Always. Identity actions (`send_as_you`, `computer_use`, HighRisk) already cannot.
15. Session overlay (`Engine.Overlay`) may return a successor hash for the caller to pin on a session. It must not write `refs/active`, `refs/canary`, or `refs/staging`. `KEEP_BASE` returns the parent hash. Ledger identity is `R=4k`, `P=3k` (so `k=1` is `R=4`, `P=3`). Anonymized selection and experience distillation sit outside that ledger; `session_attempts` records retries without extra `P`. The per-domain experience library may append `cue/lesson/evidence` after an adopt (with a non-confounded edit row); it must not write global refs, and incomplete rows are `NO_ENTRY`. Overlay evidence is a scratch file tree; the selector sees anonymized deliverable files, never `id_map`.
16. The handheld listener is opened only by the operator. Agent tools cannot mint a pairing nonce or write `devices/`.
17. A phone call without a current device signature, or with a reused nonce, or from a revoked device, fails. A later socket from an already allowed phone uses `resume` signed by that device key. It does not mint a pairing code and does not ask the operator again.
18. Handheld responses never include vault material (`api_key`, `search_key`, tokens, passwords). `has_key` may.
19. A successful mutating RPC emits `state.changed`. The phone calls the same `Dispatch` method table as the desktop, including `thread.start` and `thread.model.set`. Pairing itself stays on the computer.
20. The LAN browser door is the same pairing QR, as an `http` URL on a private address. The first consumer of that nonce wins. A browser has no device key: the operator approves it on the computer, then an HttpOnly session cookie authorizes the existing UI and `/api`. JSON leaving that door is scrubbed. `yoyo serve` on `127.0.0.1:3080` stays loopback. The phone app still dials the TLS port in the query and pins the certificate.
21. External agents get none of yoyo's provider keys by default. A key reaches an agent only when the operator binds it explicitly in Buddy (its own `buddy.<id>.<NAME>` vault entry, or the per-agent "reuse from yoyo" switch), and only through the environment at process start. Keys never appear in argv, diagnostic logs, worker logs, log files, or handheld responses (`hasKey` may).
22. External agent processes run in the ticket directory with a scrubbed environment: allow-listed variables only, no `YOYO_*` and nothing secret-like except the keys bound to that agent. Writes and permission requests that go through yoyo pass the Broker's directory rules: ticket, named and shared directories are writable, predecessor directories are read-only.
23. The MCP control plane listens on `127.0.0.1` only. Each ticket worker gets its own token, revoked when the worker closes; foreign `Origin` and `Host` are refused; a token reaches only its own ticket's `orchestrator_*` tools.
24. Buddy downloads use HTTPS only. A published sha256 must match. Artifacts without a sha256 never come from a mirror. A third-party install script runs only after a person has seen the full command and confirmed it.
