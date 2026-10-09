# Persist

Yoyo persists operator data as JSON files under `$YOYO_HOME` (default `~/.yoyo`). There is no product SQLite database.

| Path | What |
|---|---|
| `config.yaml` | Appearance, workspace, gate, budget, `packs.<id>.enabled`. Partial patches merge; omitted keys stay. |
| `packs/<id>/` | Installed skill packs (`pack.json` + `skills/`). Not CAS; origin is GitHub or a local checkout. |
| `connections/*.json` | Accounts (chat, image, video, speech, storage, search, workflow, OTEL, MCP HTTP). Secrets never live here — only a vault key name. |
| `connections/_defaults.json` | Default connection id per capability. |
| `video/docs/{collection}/{id}.json` | Dramas, episodes, assets, jobs, canvas projects. Atomic temp+rename. |
| `cas/objects` | Content-addressed blobs. |
| `vault.json` / OS keychain | API keys. |
| `personal/workspace.json` | Durable personal tasks, hashed proposals (with activity), ideas, goals, monitors, artifacts. Atomic temp+rename. |

SQLite remains only as a **reader**:

1. One-shot migrator: if `video/video.sqlite` exists from an older build, it is copied into `video/docs` and `connections`, then renamed to `video.sqlite.bak`.
2. Huobao import: an external `.sqlite` / `.db` project file is read once and written into the file store. Huobao is not Yoyo's persist format.

## Files vs SQLite

For this desktop app the file tree wins:

- Inspectable. An operator can open a drama JSON or a connection row without a DB browser.
- No schema migrations for product records. Adding a field is adding a JSON key.
- Same layout as the rest of the home (`sessions/*.jsonl`, `memory`, `journal`).
- Atomic rename is enough. Yoyo is one GUI process plus its job workers, serialized by the store mutex.
- Backup and copy are ordinary files.

SQLite would win at concurrent writers, ad-hoc SQL, and secondary indexes. None of those are the workload: the GUI lists a few dozen dramas and a handful of connections, and generation jobs already serialize on the engine.

Secrets stay in the vault either way. The file store holds public connection metadata only.
