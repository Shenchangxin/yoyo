# Persist

Yoyo persists operator data as JSON files under `$YOYO_HOME` (default `~/.yoyo`). The learning module is the one writable product SQLite database. Video SQLite stays a reader only.

| Path | What |
|---|---|
| `config.yaml` | Appearance, workspace, gate, budget, `paused`, `packs.<id>.enabled`, `skills.auto_invoke` (`off`/`hint`/`on`), `skills.catalog_budget_chars`. Partial patches merge; omitted keys stay. |
| `packs/<id>/` | Installed skill packs (`pack.json` + `skills/`). Not CAS; origin is GitHub or a local checkout. |
| `connections/*.json` | Accounts (chat, image, video, speech, music, stock, asr, storage, search, workflow, OTEL, MCP HTTP). Secrets never live here — only a vault key name. |
| `connections/_defaults.json` | Default connection id per capability. |
| `video/docs/{collection}/{id}.json` | Dramas, episodes, assets, jobs, canvas projects, `studio_topics`, `studio_projects`, `studio_publishes`, `cut_projects`. Atomic temp+rename. |
| `video/docs/_studio_session_binds.json` | Video session id → Studio project id. |
| `video/docs/_cut_session_binds.json` | Video session id → cut/footage project id. |
| `video/exports/{id}.mp4` | Operator-facing copies of assembled shorts for drag-drop onto creator pages. |
| `cas/objects` | Content-addressed blobs. |
| `vault.json` / OS keychain | API keys. |
| `personal/workspace.json` | Durable personal tasks, hashed proposals (with activity), ideas, goals, monitors, artifacts. Atomic temp+rename. |
| `learn/learn.sqlite` | Learning system of record: notes, libraries, chunks, embeddings, paths, practice, journal, books. WAL. |
| `learn/raw/<sha256>` | Original imported bytes. Linked libraries keep the operator's files in place. |
| `pages/{spaceId}/{pageId}.md` | Archive of notes imported once into `learn/learn.sqlite`. Later edits stay in the database. |
| `pages/index.json` | Old rebuildable page tree. Not read after the one-shot import. |
| `pages/reviews.json` | Old awaiting reviews, copied once into `note_reviews`. |
| `profiles/{id}.json` | Named specialist profiles (role, tool grants, space access). Harbor never loads these. |
| `sessions/{id}.meta.json` | Thread metadata including `profile_id`, `page_id`, `library_id`, `path_id`, and `material_id`. Harbor eval ignores them. |

Video SQLite remains only as a **reader**:

1. One-shot migrator: if `video/video.sqlite` exists from an older build, it is copied into `video/docs` and `connections`, then renamed to `video.sqlite.bak`.
2. Huobao import: an external `.sqlite` / `.db` project file is read once and written into the file store. Huobao is not Yoyo's persist format.

`learn/learn.sqlite` is the exception. Notes, retrieval, mastery, practice, the journal, and books share that one database. The rest of Yoyo stays JSON. Harbor does not open the learning database.

## Files vs SQLite

For this desktop app the file tree still wins outside learning:

- Inspectable. An operator can open a drama JSON or a connection row without a DB browser.
- No schema migrations for product records. Adding a field is adding a JSON key.
- Same layout as the rest of the home (`sessions/*.jsonl`, `memory`, `journal`).
- Atomic rename is enough. Yoyo is one GUI process plus its job workers, serialized by the store mutex.
- Backup and copy are ordinary files.

SQLite would win at concurrent writers, ad-hoc SQL, and secondary indexes. None of those are the workload: the GUI lists a few dozen dramas and a handful of connections, and generation jobs already serialize on the engine.

Secrets stay in the vault either way. The file store holds public connection metadata only.
