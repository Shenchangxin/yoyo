# Threat model

- Candidate plugins escape the loader, disable the sandbox, read vault secrets, or replace the evaluator.
- Eval gaming via malformed outputs, budget relaxation, or split leakage to the proposer.
- Prompt injection from files or the web that triggers self-install of a high-risk plugin.
- Overfitting held-in tasks while held-out performance collapses.
- Playbook full-rewrite (context collapse) instead of ACE-style deltas.
- Crash leaving a half-installed candidate as `refs/active`.
- Supply chain: unsigned WASM, unsigned desktop binaries, stolen update keys.
- Connector exfiltration: address books or mail posted to an arbitrary URL.
- Browser injection / secret paste into an isolated Chromium session.
- Scheduled jobs writing the operator home workspace or sending as the user.
- Memory poisoning: model-chosen “important” facts promoted without Harbor.

Mitigations already in the kernel: frozen TCB fiber, sealed splits, hash-chain journal, WASM size/time limits, capability broker, canary refs before active, L3 human gate (policy/loop topology), L4 never self-modifiable.

Office/personal-OS mitigations: OS job-object / sandbox-exec / bubblewrap around shell (honest badge), OS keychain vault, AutoSafe heuristics plus hooks veto, `send_as_you` NeverAlways, isolated browser profile (never the daily Chrome), virtual-display computer use with allowlist + recording, schedule isolation via worktree copy, memory writes stay staging until operator/Harbor promote, unsigned expert packs cannot admit to `refs/active`.
