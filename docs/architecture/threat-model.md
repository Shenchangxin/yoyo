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

## Buddy: external agents as workers

Buddy lets third-party ACP agents (Claude Code, Codex, Gemini CLI, custom commands) take tickets. See [buddy.md](buddy.md).

- A third-party agent runs with the user's rights. Its own sandbox, shell tool and file access are not yoyo's: yoyo confines only what the agent routes through yoyo (ACP file writes and permission requests). An agent that writes with its own tools, or runs its own shell, is bounded by its own settings and the OS, not by the Broker.
- A tampered mirror or package serves a modified agent or runtime.
- Another local process, or a web page in a browser, calls the loopback control plane to act as a worker or to reach another ticket's tools.
- Secrets leak through agent stderr, ACP protocol frames (including the control-plane token handed to the stdio bridge), install and probe logs, error details, Doctor, or the phone.
- A ticket reassigned to another agent carries the previous agent's private context, session or credentials, or the old agent keeps working on it.
- A phone triggers installs, logins or configuration changes the operator did not see.

Mitigations: keys stay in the vault as `buddy.<id>.<NAME>` and reach an agent only through its environment at start, after an explicit binding; the launch environment is allow-listed and drops `YOYO_*` and secret-like variables. yoyo never changes an agent's mode (it only sets the model the operator picked), so it never moves an agent into one that lowers supervision; it answers permission requests with `allow_once` when it allows them, checks ACP writes against the ticket's directory rules, and answers pending requests with cancelled when a turn is cancelled. Downloads are HTTPS-only with sha256 verified when published; artifacts without a sha256 never use a mirror, and agents installed without one are marked unverified; system-wide install scripts show the full command and need a person's confirmation. After a managed install, yoyo clears the Windows `Zone.Identifier` mark and the macOS quarantine attribute on that directory so the OS lets it run; that trust rests on the HTTPS and sha256 checks above. The control plane binds `127.0.0.1`, issues one token per ticket worker (revoked when the worker closes), refuses foreign `Origin` and `Host`, and exposes only that ticket's `orchestrator_*` tools; the token travels in an `Authorization` header or the bridge's environment, never in argv. Install logs, probe logs, stderr tails, error details and ACP frames are redacted by known secret values and by pattern (Bearer, `key=`, `*_KEY=`, and secret-named `name`/`value` entries); frames are recorded only when the `acp` debug category is on and are cut to 2 KB after redaction. Reassignment ends the old assignment, clears the external session pointer and usage, retires the old worker (its later `orchestrator_*` calls are told to stop), and starts the new agent from a handover note rather than the old transcript. The phone cannot install, uninstall, log in or out, or clear the cache; configuration and preference changes need confirmation; responses carry `hasKey`, never key material.
