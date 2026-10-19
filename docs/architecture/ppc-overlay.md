# Session Propose–Probe–Compose overlay

Task-level harness edits for **this thread**, without ground-truth scores.
Literature: [Harness²](../research/corpus/2610.14224-harness2.md). Global
promotion stays Harbor (`ShouldPromote`) and a human checkout.

This document is the v1 contract. An evolving agent must not change it.

## One sentence

An operator may spend a matched budget of **4k solver rollouts and 3k improver
sessions** to pin the current session to a new L1 snapshot — or leave the
session exactly as it was. `k=1` is four runs and three edits.

## Non-goals

- Not `yoyo evolve`. That loop mines a Harbor suite and writes `refs/canary`.
- Not every `Send`. Overlay is an explicit operator action.
- Width `agg` (2k candidates then k compositions) and depth `seq` (k nested
  rounds) share the same `R=4k,P=3k` identity. Finals are chosen by anonymized
  selection; picking the unmodified final is valid.
- Not a second runtime, Python, OpenCode, or Codex.
- Not a path onto `refs/active`. The experience library stores a CAS hash plus
  `cue/lesson/evidence`; it never moves production pointers.

## Pointers this path may touch

| Pointer | Overlay v1 |
| --- | --- |
| `session.Harness` + `HarnessPolicy=pin` | **only if a successor is adopted** |
| `refs/active` | never |
| `refs/canary` | never |
| `refs/staging` | never (staging is the global ACE buffer; DOM must not leak into the next evolve cycle) |
| `refs/archive/*` | never |
| CAS objects | yes: new snapshot + L1 payloads, parented on the session’s current hash |

`KEEP_BASE` (compose finds no supported behavioral contrast, or both probes
fail) leaves `Harness` and `HarnessPolicy` unchanged. Pinning the current
active hash would freeze the session against future checkouts; v1 will not do
that by accident.

Invariant 15 (enforced by `internal/evolve/overlay_test.go`):

> Session overlay may pin a session to a snapshot. It must not write
> `refs/active`, `refs/canary`, or `refs/staging`. `KEEP_BASE` leaves
> `HarnessPolicy` unchanged.

## Budget (`R=4k`, `P=3k`)

Identity from Harness², including the shared bases. `k` is clamped to 1–4.
Suite evolve `--k` is still rejected on this path; overlay width is
`--overlay-k` / RPC `overlay_k`.

| Kind | `k=1` labels | `k>1` labels | Count |
| --- | --- | --- | --- |
| Solver rollouts `R` | `base`, `inv1`, `dom1`, `c1` | `base1..k`, `inv1..k`, `dom1..k`, `c1..k` | 4k |
| Improver sessions `P` | `inv`, `dom`, `compose` | `inv1..k`, `dom1..k`, `compose1..k` | 3k |

Selection and journal reflection sit **outside** the ledger, same as Harness².
Retries of an invalid proposal (one retry, then inherit) are `session_attempts`,
not extra `P`. Distilling an experience row is also outside the ledger.

If the operator triggers overlay **after a finished turn**, that turn is the
first base (`cached=true` on the ledger). Remaining bases (when `k>1`) run
live. If there is no reusable turn, every base runs live.

A live overlay that cannot record `R==4k` and `P==3k` is a bug, not a partial
success. Stop and `KEEP_BASE`.

Candidate probes and the k finals run concurrently. Live improvers may propose
the 2k INV/DOM sessions in parallel; fixture / scripted clients stay ordered
so a single retry still hits the identity. Retries are `session_attempts`, not
extra `P`.

## Flow

```
operator: Evolve this thread
        │
        ▼
resolve parent hash = session.ResolveHarness(policy, harness, refs/active)
reuse last turn as base, else roll base on a scratch worktree
        │
        ▼
k INV proposers + k DOM proposers (ordered; shared base evidence)
        │
        ▼
2k candidate probes in parallel (inv1..k, dom1..k)
        │
        ▼
k composers (independent; no scores)
        │
        ▼
k finals in parallel (c1..ck)
        │
        ▼
anonymized selection among the k finals (outside the ledger)
        │
        ├── KEEP_BASE  → stop, policy unchanged (pick unmodified final is valid)
        └── pin session to the chosen successor hash

After an adopt, a reflector may write one `cue/lesson/evidence` row into the
per-domain library. Incomplete rows are NO_ENTRY. KEEP_BASE never writes the
library. Distil does not count toward P.
```

No Harbor metrics, no `ShouldPromote`, no manifesto hit-rate on overlay.
Those gates exist for **global** canary. Overlay’s only accept rule is:

1. `AdmitQuality` on each proposal (existing deterministic gate).
2. Composer cites a trajectory moment for every kept edit.
3. `c1` delivered (the user-asked artifact exists) **or** composer chose
   `KEEP_BASE`.
4. Loop topology and policy pack hash equal the parent (`LoopTopologyEqual`).
   Overlay cannot take L3.

If `c1` fails to deliver, do not pin a “maybe better prompt”. `KEEP_BASE`.

## Surfaces

Parent snapshot is the session’s current materials, not DGM `SelectParent`.
Archive sampling is for suite-level evolve.

| Dimension | May write | Must not write |
| --- | --- | --- |
| **INV** | one of: `instruction` (`verification` / `failure_recovery`), `middleware`, procedure `skill` | playbook, DOM facts, L3 topology, tools, WASM |
| **DOM** | one on-demand `skill` | playbook, always-on fragments, middleware, loop copy, matter facts as durable memory |

One proposal, one L1 surface — reuse `AdmitQuality` / `payloadCount`.
`tool_use` stays frozen. DOM is skill-only so a later Harbor promote cannot
smuggle this-matter fees into the global playbook. Matter facts belong in the
task material, not the harness.

Always-on inflation: INV instruction is the expensive option. `AdmitOverlay`
rejects a verification / failure_recovery slot that has a recognisable trigger
(`when …`, `if the user …`) — that edit must be a skill — and rejects
instruction text over 2_000 runes. Middleware is not always-on.

## Isolation

Probes are **not** the operator workspace.

1. Copy the session tool root with `runtime.PersistWorktree` into a temp dir
   (reuse `ScratchGit` only when the tool root is already a git repo you can
   worktree; do not `git init` a user’s documents folder).
2. Each label (`base` if live, `inv1`, `dom1`, `c1`) gets its own copy.
3. Run `runtime.Run` with `HarnessPolicy` effectively pinned, packs off,
   profiles/pages off (same as Harbor: overlay is harness search, not Personal
   OS). Capability stays deny-first; HighRisk / `send_as_you` stay off.
4. Dispose copies. Probe files never merge back.

The triggering `base` turn, if reused, already happened on the live tree.
Do not rewind it. Subsequent probes still use copies.

## Evidence the improver may see

A scratch evidence tree, not one giant JSON chat:

```
task.md                 # last user message, criteria stripped if any
source/                 # optional attachments; not page identity
rollouts/<label>.md     # [ACTION]/[OBSERVATION]/[ERROR], secrets redacted
deliverables/<label>/   # extracted artifacts, or names-only if huge
harness/                # parent snapshot rendered as materials, not refs
decisions/<label>.md    # proposer notes
```

Must not appear: Harbor suite ids, held-out / transfer / Index identities,
grader markers (`pass_rate`, `ShouldPromote`, `held-out`), vault, updater.
Reuse the existing held-out sanitizer. Prompt slots follow Harness²: do not
name rubric / score / benchmark / grader.

Trajectory observation truncation floor: 10_000 characters. Below that we
fabricate defects (Harness² `obs_trunc` comment). v1 may cap a single
observation at 50_000.

Page bodies, mail, and voice transcripts stay untrusted source data
(invariant 13). They may sit under `source/` as objects. They must not enter
identity or a playbook bullet.

## APIs

Operator-facing name: **Evolve this thread**. Internals: `overlay`.

```
RPC    evolve.overlay   { session, max_usd, max_wall, reuse_turn, overlay_k, overlay_mode }
       evolve.library   { session }
CLI    yoyo evolve --overlay --session <id> [--overlay-k N] [--overlay-mode agg|seq]
       yoyo evolve --overlay --session <id> --overlay-library
GUI    Harness Propose → Evolve this thread (Width / Depth)
       Domain library (read-only INDEX)
       Prove this pin (Harbor on the pin; no checkout)
       Command palette (RSI group)
```

`PromoteActive`, `--sealed`, `--baselines`, `--promote` are **rejected** on
this path. Mixing overlay with suite evolve in one RPC is a bug.

Journal:

```
ppc.overlay.intent   { session, parent, reuse_turn }
ppc.overlay.commit   { session, adopted, hash, kept_base, R, P, usd, session_attempts, id_map, diversity }
```

`adopted` is false on `KEEP_BASE`. Intent without commit (crash) leaves
pointers untouched.

## UI

Harness Propose grows a second action, visually below suite evolve, not a
sibling chat. Copy:

- Title: Evolve this thread
- Hint: `{r} runs, {p} edits. Pins this chat. Never moves active.`
- Next: KEEP_BASE → “No contrast. Active unchanged.”
         pin → “This chat is pinned to \<hash8\>. Checkout still owns production.”

Show the overlay ledger (`R`/`P`/USD/`k`) on the same lab spend table as BoN
when either ran; overlay itself does not run BoN.

## Experience library

On-disk: `{home}/library/{domain}/INDEX.md` plus
`entries/{task}/EXPERIENCE.md` and a `hash` file holding the adopted CAS
pointer. Domain is the workspace basename. Retrieval is by construction —
alpha never lists beta. Held-out ids and grader tokens (`pass_rate`,
`ShouldPromote`, `rubric`, …) make the write `NO_ENTRY`. The improver sees INDEX.md plus L1 materials expanded from each entry's CAS
hash (`entries/<task>/harness/`: instruction, middleware, skills). Playbook
and held-out identities never enter that tree. Distil writes
helped/hurt/no observed effect/confounded rows first; a lesson that rests only
on confounded rows is NO_ENTRY.

## Code map (when implementing)

| File | Role |
| --- | --- |
| `internal/evolve/overlay.go` | `Overlay`, ledger, adopt/KEEP_BASE, width k |
| `internal/evolve/overlay_prompt.go` | INV / DOM / compose / distil prompts |
| `internal/evolve/library.go` | per-domain `cue/lesson/evidence` or NO_ENTRY |
| `internal/evolve/overlay_test.go` | pointer, budget, surface, leak tests |
| `internal/app/session_overlay.go` | resolve session, isolate copies, pin |
| `internal/api/rpc.go` | `evolve.overlay` |
| `cmd/yoyo/main.go` | `--overlay --session` |
| `frontend/.../EvolveLab.tsx` + `copy.ts` | Propose action |
| `docs/architecture/invariants.md` | invariant 15, with tests |

Do not add an `internal/harness` package. Overlay writes CAS snapshots
through the existing `ApplyProposal` / `MergeProposals` helpers.

## Tests that must exist before UI

1. Overlay of a fixture session does not change `refs/active`, `canary`,
   `staging`.
2. `KEEP_BASE` leaves `HarnessPolicy` as `follow_active` when it started that
   way.
3. Adopt sets `pin` and a child snapshot whose `Parent` is the pre-overlay
   hash; `LoopTopologyEqual` holds.
4. DOM proposal that includes a playbook payload fails `AdmitQuality` before
   any rollout.
5. Evidence bundle / trial logs contain none of the held-out ids from the
   seeded suite.
6. Ledger identity: `R==4k`, `P==3k` (one rollout may be `cached`).
7. Invalid INV JSON retries once, then inherits parent (fell_back), and still
   hits the budget identity.
8. KEEP_BASE does not create a library INDEX. Adopt writes cue/lesson/evidence
   or NO_ENTRY. Domain B cannot read domain A's entries.
9. `k=2` records R=8 P=6 and still freezes active/canary/staging.

## Promote later

An overlay pin is **not** a canary. A later operator action may run Harbor
against the pinned hash (`yoyo eval --snapshot HASH`, RPC `eval.run` with
`snapshot`, GUI **Prove this pin**) and checkout if `ShouldPromote` passes.
That is the existing Prove → Promote rail. Overlay must not grow a hidden
`--promote`, and the prove shortcut must not checkout.
