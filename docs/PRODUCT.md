# Yoyo product

Yoyo is a **local personal OS client** of one Go loop. Harness / RSI remains the control tower. The academic and product lineage for that tower lives in [docs/research/README.md](research/README.md). Inbox, Projects, Artifacts, Automations, Memory, Connectors, **Pages**, and **Profiles** are application surfaces on that loop — not a second runtime and not personality marketing.

The GUI is a client of one Go loop. It is not an IDE, not a cloud multi-agent home, and not a ChatGPT marketing skin.

## Who it is for

An operator who wants to run turns against a local workspace, review git hunks, and promote a harness only when Harbor evidence says so.

## What the client is

```
[ Thread rail ] [ Transcript + composer ] [ Review ]
                 Harness is one RSI workspace
```

- **Thread** is an object: create, rename, fork, search, running indicator. A thread may bind a **profile** (capability projection) and a **page** (untrusted document context). Identity is a session attribute, not a second agent.
- **Pages** are the durable knowledge home under `~/.yoyo/pages`. Chat is process; an approved page is the result. Agent writes go through hashed `review_page`, never `write_file` into the pages tree.
- **Profile** is a named role plus tool/space grants on the same Go loop. Harbor, `task` subagents, and the video channel ignore it.
- **Plan** is a composer chip, not a page. It proposes; it does not write.
- **Pause** is a workstation freeze: running turns, scheduled jobs, the personal worker, and voice hang up. It does not rewind completed side effects.
- **Follow** jobs continue an existing thread. Isolated cron/heartbeat/webhook jobs stay a separate class and still isolate.
- **Approvals** render in the stream. Review is a queue mirror, not the only Once/Deny surface.
- **Harness** is one workspace: Overview → Propose (Evolve) → Prove (Harbor) → Promote (refs). It is not a chat sibling of equal weight. Chat dock there is opt-in.
- **Video** is one chrome with four lanes. **创作** is the finish desk:
  empty, it asks for a clip, a 翻拍 reference, or a workshop, and keeps the packed
  case catalog on the page (prompts in `creation-inspirations.ts`, covers in
  `frontend/public/short-drama-styles`; a click fills the conductor and does
  not generate); with a clip, the
  player and edit list sit beside the conductor. **短片 / 短剧 / 画布** are
  the workshops and still own text-to-video. Shared post
  (`internal/video/cut`: Breakdown, EditList, Stems, Captions) is the
  finish desk, not a fifth lane. 翻拍 keeps a production project on this
  desk: words, shots, a plan, code pictures, and connection-backed image,
  video, speech, music, and matting steps. Ranking, podcast, and street-interview
  structures are included. **送到成片台** is the explicit handoff onto the
  existing cut. Assembled shorts, stitched episodes, and
  canvas video nodes send here with **送到成片台**, then the cut is finished
  on that desk. The canvas timeline is a different editor. Video chats and
  Agent chats stay separate session channels.
- **Idol** is a desktop person on the companion window. Chat and calls stay in her own record. When the operator asks her to work, that line is handed to a normal agent session on this machine. Her models, speech, recognition, and image endpoints are chosen in Settings. The emotion ball stays the actor in the main window unless a portrait is pinned on the companion.
- **Control** is a gear surface (vault, policy, updater). Opening it hides the thread rail. The agent cannot change these.
- **Skills** is a catalog plus **Packs**. A pack is a versioned skill tree. Methodology packs (Superpowers) session-start bootstrap; domain packs (NovelToGame) do not preload bodies. Enabling a pack is a workspace choice. Chat matches catalog descriptions and may auto-load a domain orchestrator; `@skill` is an override. Harbor never loads packs.
- **Connections** are JSON files under `~/.yoyo/connections`. Chat, image, video, speech, music, matting, stock, asr, search, storage, and workflow accounts share one registry. Generation catalogs domestic vendors (Qwen / Jimeng / Kolors / CogView / Hunyuan / Vidu / Kling / CosyVoice / Spark / Mureka / DashScope / Volcengine / Tencent COS) alongside international ones. Keys stay in the vault. The conductor only sees enabled rows that have a key. See [architecture/persist.md](architecture/persist.md).

## Distinctive, not decorative

CAS hashes, Harbor gates, untrusted staging, and deny-first policy. Distinctive craft comes from review density and lab evidence — not Sparkles, gold leaf, or personality marketing.

## Non-goals

- A second agent runtime in the frontend (no CopilotKit, AG-UI, LangChain, Vercel AI SDK, or assistant-ui loop).
- OpenBot / gVisor / one container PC per identity. Isolated Chrome and the virtual desktop stay the computer.
- L4 UI. Evaluator, vault, and updater are never agent-editable.
- Fake sandbox badges. OS isolation is opt-in and reported honestly.
- File-tree-as-home Cursor clone.
