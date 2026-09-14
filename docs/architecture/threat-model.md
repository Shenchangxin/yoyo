# Threat model

- Candidate plugins escape the loader, disable the sandbox, read vault secrets, or replace the evaluator.
- Eval gaming via malformed outputs, budget relaxation, or split leakage to the proposer.
- Prompt injection from files or the web that triggers self-install of a high-risk plugin.
- Overfitting held-in tasks while held-out performance collapses.
- Playbook full-rewrite (context collapse) instead of ACE-style deltas.
- Crash leaving a half-installed candidate as `refs/active`.
- Supply chain: unsigned WASM, unsigned desktop binaries, stolen update keys.

Mitigations already in the kernel: frozen TCB fiber, sealed splits, hash-chain journal, WASM size/time limits, capability broker, canary refs before active, L3 human gate (policy/loop topology), L4 never self-modifiable.
