# Learn

The learning module is five surfaces on one SQLite store and one tutor: **学习计划** (what to do now), **掌握** (mastery paths), **文库** (libraries and the reader), **课本** (generated books), **练习** (question bank and spaced review). The tutor is the ordinary chat loop on a `learn`-channel session with a playbook and tools scoped to whatever the chat is bound to. There is no second agent loop.

Feature lineage is DeepTutor (mastery path, books, immersive reading, practice). The scheduler is FSRS-6. Prerequisite gating follows the knowledge-frontier idea from Math Academy.

## Status

| Part | State |
|---|---|
| Storage, migrations, change events, model injection | Built. |
| Mastery engine, tutor tools, playbook, mastery UI, question card | Built and tested. Tutor prompts and the judge have only run against a fake model. |
| Practice: FSRS queue, check and review, bank, import, generation, insights, tutor bank tools, practice UI | Built and tested. Generation and model judging have only run against a fake model. |
| Library and reading: background indexing, sectioning, hybrid search, the reader with positions, bookmarks, and anchored annotations, tutor reading tools, library and reader UI | Built and tested. Embeddings have only run against a fake embedder. PDF text and outlines were checked on real files. |
| Books: proposal, spine, chapter-by-chapter writing with pause and resume, block edits, quizzes, export, tutor book tools, books UI | Built and tested against a fake model. No book has been written by a real model yet. |
| Study plan page | Built. It reads `learn.today`. |

Everything that calls a model (tutor turns, the judge, outline drafts, question generation, book writing) is covered by tests with a scripted model only. The prompts have not been tuned against a live one.

## Principles

1. **The engine decides.** Mastery, schedules, and the next step are computed in `internal/learn`. The model proposes questions and explanations; it cannot grant mastery, and it never sees an answer key it did not write.
2. **Evidence is replayed, not accumulated.** A knowledge point's score, status, and memory state are a pure function of its evidence rows in order. Voiding a row, changing retention, or migrating an old database all go through the same replay.
3. **Ungraded is not wrong.** When an answer cannot be judged with confidence nothing is recorded. No credit, no penalty.
4. **Answers stay in the store** until the answer is graded. Public question shapes carry no key and no explanation.
5. **Source text is data.** Library hits, page text, selections, and notes are untrusted. Prompts wrap them in tags and tell the model to ignore instructions inside.
6. **Local first.** One file, `learn/learn.sqlite`, plus original bytes under `learn/raw/`. Model calls go out only through two injected functions.

## Storage

`internal/learn/db.go` holds the first-version `CREATE TABLE` statements. Everything after that is a migration in `schema.go`, so a fresh database and an upgraded one take the same path. `schema_migrations(id, name, applied_at)` records what ran. A migration and its record share one transaction.

Migration id ranges: 1–99 legacy patches, 100s mastery, 200s practice, 300s library and reading, 400s books. Register with `registerMigration(id, name, func(tx *sql.Tx) error)` from an `init()` in the domain's own file. Use `addColumns(tx, table, "name TYPE NOT NULL DEFAULT x", ...)`; it skips columns that exist.

The store has one connection and one mutex (`s.mu`). Rules for store code:

- Take `s.mu` for every read and write. Never call a model or do file I/O for long while holding it.
- Multi-statement writes use `s.inTx(func(tx *sql.Tx) error)`. Finish reading a result set before issuing the next statement.
- After a committed write, call `s.notify(scope, id, kind)` with the lock released.
- Times are RFC3339Nano UTC strings. Use `s.clock()` for now; tests replace `Store.Now`.
- Ids come from `newKey(prefix)`.

Injected by the app at open:

| Field | Purpose |
|---|---|
| `Store.Complete` | One completion without tools. `completeJSON(ctx, CompleteRequest{Purpose, SessionID, System, User, MaxTokens, Temperature}, &out)` parses the reply and retries once on bad JSON. Nil means `ErrNoModel`. |
| `Store.Embed` | Batch embeddings for a library's model. Nil or no model means lexical search only. |
| `Store.OnChange` | Receives `Change{Scope, ID, Kind}` after each committed write. |
| `Store.Now` | Clock. |

## Mastery

A path has modules; a module has knowledge points. A point has a kind (`memory`, `procedure`, `concept`, `design`), optional prerequisites (a DAG, validated on write), and evidence.

**Gates.** `memory` and `procedure` points pass on score: a recency-weighted mean of the last five independent answers (`0.5, 0.7, 0.85, 0.95, 1.0`), capped at 0.5 after one answer and 0.8 after two, gate 0.9. So at least three independent correct answers. Partial counts 0.5. `concept` and `design` points pass on an explanation judged from the learner's own words; quiz accuracy never opens that gate. Hinted answers do not count toward the score.

**Claims.** A learner may claim a point. It shows as mastered with source `learner`, routing moves on, and the point gets an early review. The first incorrect answer withdraws the claim.

**Schedule.** A mastered point carries FSRS-6 memory (`stability`, `difficulty`). First interval comes from a prior by kind (memory 2.3 d, procedure 3.5 d, concept 5 d, design 8.3 d). Every later answer on a mastered point is a review: correct → Good, partial or hinted → Hard, incorrect → Again. Due = last review + interval at the path's `desired_retention` (0.7–0.97, default 0.9), rounded to whole days, at least one day. Changing retention re-derives due dates from the last review; it is not a review.

**Next step** (`pathState.next`), in order: a pending question → `answer_pending`; a mastered point that is due, lowest retrievability first → `review`; the first unmastered, non-deferred point whose prerequisites are all mastered → `probe` (new), `teach` (new, teach-first profile), `practice` (quantitative, seen), `assess` (qualitative, seen); points blocked only by prerequisites; deferred points; otherwise `complete`. An empty path is `outline`.

**Questions.** One pending question per path. `PoseQuiz` shuffles choice options (seeded by the question id) and stores the key as a label. `Answer` grades choice and exact matches locally; everything else goes to the judge model, which must return confidence ≥ 0.8 or the result is `ungraded`. A miss is copied into the question bank as a mistake due the next day. `RepairQuestion` voids an attempt or re-keys and re-grades it. `SkipQuestion` drops the pending question. `DeferPoint` postpones a point.

**Outline edits** keep identity: a point survives by id, or by a name+kind that is unique on both sides. Surviving points keep their evidence wherever they move. Removed points take their evidence with them. `ReviseOutline(..., keepMastered=true)` is what the tutor gets.

**Modes** are a path field: `outline` (shape the map), `study`, `review` (re-test mastered points only). Tools are always mounted; the mode is checked when a tool is called.

## Tutor

`attachLearn` (`internal/app/learn_tools.go`) mounts tools on learn-channel parent sessions and adds them to `ExtraEnabled` so they are in the opening tool list. Bindings come from session meta (`library_id`, `path_id`, `material_id`, `book_id`); ids in model arguments are dropped.

| Scope | Tools |
|---|---|
| always | `learn_journal`, `learn_bank_search`, `learn_bank_add` |
| library | `learn_files`, `learn_search`, `learn_read` |
| material | `learn_toc`, `learn_page`, `learn_goto`, `learn_annotate` |
| book | `learn_book_outline`, `learn_book_chapter` |
| path | `learn_status`, `learn_mode`, `learn_profile`, `learn_build`, `learn_revise`, `learn_quiz`, `learn_grade`, `learn_assess`, `learn_skip`, `learn_repair`, `learn_defer` |

`learn_quiz` returns `Halt`: the batch ends the turn in both loops (`runtime` via `HaltAfter`, desktop via `dsh.Result.Concludes`). `learn_grade` and `learn_assess` read the learner's actual message (`OperatorVoice`), not a model-supplied answer.

The system prompt for a learn chat is the chat identity with the bootstrap replaced and pins added by `applyLearnLanePrompt`: tutoring conduct always, the mastery playbook when a path is bound, source rules when a library, material, or book is bound. Each turn's inject carries `learnTurnBrief`: a `<learning_state>` JSON block with mode, counts, next step, and any card answer the engine graded since the last turn (reported once), plus the reader viewport and book chapter when bound.

Every tool result is one JSON object. Results that carry source text include `"untrusted": true`.

## RPC

All methods go through `App.LearnCall(method, params)`; the desktop and `/api/rpc` share it. One file per domain in `internal/app`: `learn_api.go` (dispatch, today, journal, bind), `learn_api_mastery.go`, `learn_api_library.go`, `learn_api_practice.go`, `learn_api_books.go`. A domain handler returns `(out, handled, err)`. JSON keys are snake_case. Empty lists are `[]`, never `null`.

Shared shapes:

```
Option        { label, body }
Change event  { scope: "path"|"practice"|"library"|"material"|"book"|"journal", id, kind }
```

### Core

| Method | Params | Result |
|---|---|---|
| `learn.today` | — | `Today` |
| `learn.journal.get` / `.set` / `.add` | fields / `insight` | `Journal` / ok |
| `learn.bind` | `session, library_id, path_id, material_id, book_id` | session meta |

```
Today { journal, practice: PracticeSummary, due: Question[≤5], paths: PathSummary[], books: BookSummary[≤4], materials: Material[≤4] }
```

### Mastery (implemented)

| Method | Params | Result |
|---|---|---|
| `learn.paths` | — | `PathSummary[]` |
| `learn.path` | `id` | `PathView` |
| `learn.path.create` | `goal, name?, library_id?, sources?` | `PathView` |
| `learn.path.rename` / `.describe` / `.delete` / `.reset` | `id, name` / `id, description` / `id` | ok |
| `learn.path.outline.get` | `id` | `OutlineModule[]` |
| `learn.path.outline` | `id, modules, append?` | `{change, path}` |
| `learn.path.draft` | `id, must_cover?, session?` | `OutlineDraft` (model; not saved) |
| `learn.path.mode` | `id, mode` | ok |
| `learn.path.profile` | `id, teaching?, prior_knowledge?, target_level?, time_budget?, preferences?` | ok |
| `learn.path.retention` | `id, retention` | ok |
| `learn.path.claim` / `.defer` | `id, point_id, on?` | ok |
| `learn.path.answer` | `id, question_id?, answer, give_up?, session?` | `GradeResult` |
| `learn.path.skip` | `id` | ok |
| `learn.path.repair` | `id, question_id?, action, expected?, session?` | `GradeResult` |
| `learn.path.point` | `id, point_id` | `PointDetail` |

```
PathSummary  { id, name, goal, description, library_id, mode, status, counts{total,mastered,learning,new}, due_reviews, pending_question, complete, updated_at }
PathView     = PathSummary + { profile{teaching,prior_knowledge,target_level,time_budget,preferences}, intake_needed, desired_retention, revision,
                               sources[{id,kind,ref,label}], modules[ModuleView], next: NextStep, reviews[ReviewItem] }
ModuleView   { id, name, objective, order, mastered, total, points[PointView] }
PointView    { id, module_id, name, kind, status: new|learning|mastered, mastery, mastery_source: ""|system|learner, gate: quantitative|qualitative,
               prereqs[], blocked, deferred, attempts, due_at, retrievability }
NextStep     { action: outline|answer_pending|review|probe|teach|practice|assess|complete, point_id, name, kind, module_id, reason, question?: PublicQuestion }
PublicQuestion { id, point_id, question_type: choice|short|open, prompt, options[Option], difficulty, purpose, created_at }
ReviewItem   { point_id, name, kind, due_at, due, retrievability }
GradeResult  { question_id, point_id, result: correct|partial|incorrect|ungraded|voided, correct, final, replayed, feedback, expected, explanation,
               mastery, mastered, status, gate, next }
OutlineModule { id?, name, objective, points[{ id?, ref?, name, kind, prereqs? }] }
OutlineDraft  { description, modules[OutlineModule] }      // prereqs are point names
PointDetail   { point: PointView, module, stability, difficulty, reps, lapses, last_review_at, evidence[{at,kind,result,independent,answer,feedback,prompt,expected}] }
```

### Practice (implemented)

The question bank is one table for every producer: manual entry, import, mastery misses, book quizzes, reading quizzes, generated sets. Everything enters through `bankInsert`. `origin_key` dedupes (`mastery:<interaction>`, `book:<block>:<question>`, `import:<sha>`, `generated:<sha>`); an import preview also flags a row whose stem and type are already in the bank. Scheduling is FSRS-6 on `practice_state.stability/difficulty`; `practice_events` is the review log and is idempotent on `request_id`.

Schedule rules: a card never reviewed is `new`; a mistake that has not been reviewed is `learning`. `again` → due in 10 minutes. `hard`, `good`, `easy` → due after the FSRS interval at the target retention, rounded to whole days, at least one day. `learn.practice.check` returns the interval each rating would give, computed by the same function the review uses. Elapsed time is whole days since `last_review_at`. Retention and the new-cards-per-day cap live in `flags` (`practice_retention` default 0.9, `practice_new_per_day` default 10). Objective types (`choice`, `multi`, `true_false`, `fill`) are checked locally; an objectively wrong answer may only be rated `again`. `short` and `open` are self-rated after reveal, or judged by the model on request.

| Method | Params | Result |
|---|---|---|
| `learn.practice.summary` | — | `PracticeSummary` |
| `learn.practice.queue` | `limit?=20, tag?, source?, mistakes_only?, ids?` | `Question[]` (due first, then new up to the daily cap; no answers) |
| `learn.practice.check` | `id, answer, judge?, session?` | `{ correct: bool\|null, feedback?, reveal: Reveal, suggested: again\|good\|null, intervals: {again,hard,good,easy} }` (days; writes no review) |
| `learn.practice.review` | `id, rating, answer?, correct?, request_id` | `{ question, interval_days, reveal }` |
| `learn.practice.insights` | `days: 7\|30\|90` | `Insights` |
| `learn.practice.settings` | `retention?, new_per_day?` (omit both to read) | `{ retention, new_per_day }` |
| `learn.questions` | `query?, tag?, source?, mistakes_only?, bookmarked?, suspended?, sort?=recent\|oldest\|due, limit?=50, offset?` | `{ items: Question[], total }` |
| `learn.question` | `id` | `{ question, reveal, events[{at,rating,correct,answer}] }` |
| `learn.questions.save` | `id?, stem, qtype, options, answer, explanation, difficulty, tags` | `Question` |
| `learn.questions.delete` | `ids[]` | ok |
| `learn.questions.flag` | `ids[], bookmarked?, suspended?, mistake?` | ok |
| `learn.questions.tags` | — | `[{tag, count}]` |
| `learn.questions.import.preview` | `path? \| text?, format?=csv\|tsv\|json` | `{ token, rows[{line, question?, error?}], valid, invalid, duplicates }` |
| `learn.questions.import.commit` | `token, tags?, as_mistakes?` | `{ imported, skipped }` |
| `learn.questions.generate` | `topic, library_id?, count≤20, qtypes?, difficulty?, session?` | `{ items: DraftQuestion[] }` (model; not saved) |
| `learn.questions.add` | `items: DraftQuestion[], tags?` | `{ added }` |

```
Question { id, stem, qtype: choice|multi|true_false|fill|short|open, options[Option], difficulty, tags[], source, source_id, path_id, point_id, library_id,
           bookmarked, mistake, suspended, created_at, updated_at,
           state: new|learning|review, due_at, reps, lapses, stability, retrievability, last_review_at }
Reveal   { answer, explanation }
DraftQuestion { stem, qtype, options[Option], answer, explanation, difficulty }
PracticeSummary { due, new, reviewed_today, total, mistakes, next_due_at, retention, new_per_day, streak_days }
Insights { days, daily[{date, reviews, again, new_cards, mistakes}], retention{rate, reviews}, forecast[{date, due}], sources[{source, count}] }
```

For choice types `answer` is the label, or labels joined by commas for `multi`. `true_false` uses options `T`/`F`.

### Library and reading (implemented)

A library is `indexed` (files copied into `learn/raw`, named by content hash), `linked` (files stay where they are), or the built-in `notes`. Import marks documents `pending` and starts indexing in the background; nobody has to press rebuild. Saving a note re-indexes the notes library the same way. A linked library is re-scanned at the start of every build, so new, changed, and deleted files follow.

One background job per library (`scheduleIndex` → `indexLoop` → `buildGeneration`). Parsing, chunking, and embedding run without the store lock; the new generation is written and published in one transaction. A document that fails to parse is marked `failed` with the reason and the rest still publish. A source that changed while the build ran is left out of that generation and the library is built again. An embedding failure does not block publishing: the generation is keyword-only and the reason is kept on the library. Queries read the published generation. On open, a library that was mid-build is marked `needs_reindex`; `ResumeIndexing` (called by the app once the embedder is wired) picks it up.

Text is addressed by **locator**: a 1-based unit number. PDFs use one unit per page (`unit: "page"`); the outline comes from the PDF bookmarks, each matched to the first page that carries its title, skipping table-of-contents pages. Markdown, text, HTML, and DOCX are cut into sections of about 2800 runes at headings and paragraph breaks (`unit: "section"`). Chunks are cut inside a unit on paragraph boundaries, about 900 runes, and carry the heading path, which is also indexed.

Search fuses two rankings by reciprocal rank. The lexical side is FTS5 over CJK bigrams and lowercased words: chunks containing every query term first, then chunks containing any. The vector side is cosine over the generation's embeddings, when the library has an embedding model. Embeddings are cached by model and text hash, so a rebuild embeds only new text. A hit's heading is corrected to the last heading above the matched words inside the chunk, and its excerpt is centred on them.

An annotation is anchored by its quote and its rune offsets in the unit text, with 32 runes of context on each side. `Annotate` trusts the offset only if the text there equals the quote; otherwise it finds the quote, preferring the occurrence whose context matches, and falls back to a whitespace-insensitive match. A quote that cannot be found is still saved, marked `unresolved`. The reader draws highlights with the CSS Custom Highlight API by finding each quote in the rendered text, so the DOM is not rewritten.

`learn.viewport` records, in memory, the page the learner is on and what they selected; the tutor's turn brief carries it as `<reading_state>`, `<selection>`, and a clipped `<page>`, all marked as source data. `learn.import.url` is the one place the app goes to the network for the library: it fetches a page the operator asked for, keeps the readable text, and stores it as a Markdown document. No tutor tool can trigger it.

| Method | Params | Result |
|---|---|---|
| `learn.libraries` | — | `Library[]` |
| `learn.library.create` | `name, kind, path?, description?` | `Library` |
| `learn.library.update` / `.delete` | `id, name?, description?` / `id` | ok |
| `learn.library.embed` | `id, model` | ok (re-indexes; empty model means keyword search only) |
| `learn.documents` | `library_id` | `Document[]` |
| `learn.import` | `library_id, paths[]` (or `path`) | `{ imported, skipped[{path, reason}] }` |
| `learn.import.url` | `library_id, url` | `Document` |
| `learn.rebuild` | `library_id` | ok |
| `learn.document.remove` / `.rename` | `library_id, document_id, title?` | ok |
| `learn.document.blob` | `library_id, document_id` | `{ mime, name, base64 }` (original bytes, ≤ 40 MiB) |
| `learn.search` | `library_id, query, limit?` | `{ hits: Hit[], message }` |
| `learn.read` | `library_id, document_id, page, source_hash?` | `{ text, status: ok\|needs_ocr\|changed, page, pages, unit, kind, source_hash }` |
| `learn.materials` | — | `Material[]` (recent first) |
| `learn.material.open` | `library_id, document_id` | `{ material, document, outline[], position, bookmarks[], annotations[] }` |
| `learn.material.close` | `material_id` | ok (drops the material with its position, bookmarks, and annotations) |
| `learn.material.position` | `material_id, locator, percent` | ok |
| `learn.annotations` | `material_id` | `Annotation[]` |
| `learn.annotate` | `material_id, kind, color, locator, quote, note, start, end, prefix, suffix` | `Annotation` |
| `learn.annotation.update` / `.delete` | `id, note?, color?, kind?` / `id` | `Annotation` / ok |
| `learn.bookmark.toggle` | `material_id, locator, label?` | `{ bookmarked, bookmarks[] }` |
| `learn.locate` | `material_id, query` | `{ hits, message }` |
| `learn.material.export` | `material_id` | `{ filename, markdown }` |
| `learn.viewport` | `session, material_id, locator, selection` | ok (memory only; feeds the tutor brief) |

```
Library  { id, name, kind, description, linked_path, status: ready|indexing|needs_reindex|error, documents, embed_model, embed_conn, published_gen,
           last_error, progress?: { phase: parsing|embedding|publishing, done, total, file }, updated_at }
Document { id, library_id, rel_path, title, kind, bytes, page_count, unit, status: pending|parsed|failed|needs_ocr, error, source_url, char_count, updated_at }
Hit      { library_id, document_id, title, rel_path, page, heading, chunk_id, source_hash, excerpt, text, score }
Material { id, library_id, document_id, title, kind, unit, page_count, last_opened_at, locator, percent, annotations }
Annotation { id, material_id, kind: highlight|underline|note, color: yellow|green|blue|pink|purple, locator, quote, note, start, end, prefix, suffix,
             author: user|assistant, resolution: resolved|unresolved, created_at, updated_at }
Bookmark { id, material_id, locator, label, created_at }
OutlineEntry { locator, title, level }
```

### Books (implemented)

A book is written by the model from an intent and, when given, a library. Two human gates: the proposal, then the spine. After the spine is confirmed each chapter is planned into typed blocks and the blocks are written one at a time, persisted as they finish. One worker per book (`startBook` → `writeBook` → `writeChapter` → `writeBlock`). Pause is durable; resume re-queues only what is unfinished and reuses stored chapter plans. Two consecutive chapters with no block written pause the book with the reason; no model pauses it at once. Regenerating a chapter keeps `user_note` blocks and blocks the learner edited, at their old positions.

The plan is cleaned by the engine: unknown kinds are dropped, there is exactly one quiz and it comes last, and the count is capped by depth (brief 3–5, standard 5–7, deep 7–10). When planning fails the chapter falls back to a fixed plan instead of failing. A block reply that does not fit its shape is asked for once more with the reason; then the block is marked `error` and the chapter becomes `partial`. Chapter and block writing search the bound library and pass the excerpts as data.

Quiz answers never leave the store before an answer: `learn.book.chapter`, the tutor's `learn_book_chapter`, and the export all use the public payload. `learn.book.quiz` grades choice questions locally and short answers by exact match, then by the judge model when one is connected; a miss is copied into the question bank once (`book:<block>:<question>`). The tutor reads chapters through `BookChapter`, which does not move the reader's progress.

Statuses: book `draft` (proposal not confirmed) → `spine_ready` → `compiling` ⇄ `paused` → `ready`, or `error`. Chapter `pending|planning|generating|ready|partial|error`. Block `pending|generating|ready|error`.

Block kinds and payloads:

```
text        { body }                                   // markdown
section     { intro, subsections[{heading, body}], key_takeaway }
callout     { variant: key_idea|pitfall|tip|summary, body }
code        { language, code, explanation }
timeline    { events[{date, title, description}] }
flash_cards { cards[{front, back, hint}] }
quiz        { questions[{id, question, qtype: choice|short, options[Option]}] }   // answer and explanation are stripped from responses
user_note   { body }
```

| Method | Params | Result |
|---|---|---|
| `learn.books` | — | `BookSummary[]` |
| `learn.book.create` | `intent, library_id?, depth?=brief\|standard\|deep, language?, session?` | `{ book, proposal }` |
| `learn.book` | `id` | `{ book, proposal, chapters[Chapter without blocks], progress }` |
| `learn.book.proposal` | `id, proposal` | ok |
| `learn.book.confirm_proposal` | `id, proposal?, session?` | `{ book }` (spine is built in the background) |
| `learn.book.spine` | `id, chapters[{id?, title, summary, objectives[], content_type}]` | `{ chapters }` |
| `learn.book.confirm_spine` | `id, session?` | `{ book }` (starts compiling) |
| `learn.book.chapter` | `id, chapter_id` | `Chapter` with blocks; records the visit |
| `learn.book.pause` / `.resume` / `.delete` / `.rename` | `id, title?` | ok |
| `learn.book.chapter.regenerate` | `id, chapter_id` | ok |
| `learn.book.block.regenerate` / `.delete` | `id, block_id` | ok |
| `learn.book.block.update` | `id, block_id, body` | `Block` (text, callout, user_note) |
| `learn.book.block.insert` | `id, chapter_id, after_block_id?, kind, body?` | `Block` |
| `learn.book.quiz` | `id, block_id, question_id, answer, session?` | `{ correct: bool\|null, answer, explanation, feedback? }` (a miss goes to the bank) |
| `learn.book.export` | `id` | `{ filename, markdown }` |

```
BookSummary { id, title, description, library_id, status, depth, language, chapters, ready_chapters, pause_reason, chapter_id, percent, updated_at }
Proposal    { title, description, scope, target_level, estimated_chapters, rationale }
Chapter     { id, order, title, summary, objectives[], content_type: concept|theory|practice|history|overview, status, error, blocks?[Block] }
Block       { id, chapter_id, order, kind, status, title, payload, error, edited }
```

Change kinds on scope `book`: `updated`, `spine_ready`, `chapter_started`, `block_ready`, `chapter_ready`, `paused`, `ready`, `error`, `deleted`.

## Events

`App.publishLearn` turns each `Change` into a Hub event: type `system`, source `learn`, item kind `learn`, payload `{scope, id, kind}`. The frontend subscribes on `yoyo:item` and re-reads the affected slice. Events are hints; every view also loads on mount.

## Frontend

Typed clients, one per domain: `frontend/src/lib/learn-api.ts` (mastery), `learn-practice.ts`, `learn-library.ts`, `learn-books.ts`. Pure helpers that tests import without the client live beside them (`learn-format.ts`, `learn-practice-util.ts`, `learn-reader-util.ts`, `learn-quiz.ts`, `learn-scope.ts`). `lib/learn-events.ts` subscribes to changes; panes that show background work also poll while it runs, because the browser door gets no events.

`features/learn/LearnWorkspace.tsx` is the shell and nothing else: which pane each lane shows, the three library tabs, the tutor rail, and the scope the tutor conversation is keyed by (`learnScopeSpec`: today, practice, each path, each library, each material, each book, notes). One folder per surface: `home/`, `mastery/`, `library/` (the shelf and the reader), `books/`, `practice/`, and `cards/` for the question card. Copy lives in `lib/copy-learn/<surface>.ts` with `en` and `zh` tables of the same shape, mounted as `copy.learn.home`, `.mastery`, `.shelf`, `.tome`, `.drill`.

The study plan lists what is ready now, in this order: practice, paths, materials in progress, books that wait for the learner or are still being written. Then the four ways in, the mission, and what stuck.

The reader shows one unit at a time. Selecting text opens a small bar: five highlight colours, a note, or ask the tutor. Contents, find, and notes open above the text. Position is saved on every page turn and reported to the tutor with the selection.

A book is a list until one is opened. An open book shows the step it is at: the proposal form, the chapter list editor, or the chapters with their blocks. While it is being written the pane re-reads on events and on a timer.

Practice has three tabs. Review shows what is ready and runs a round one card at a time: answer, check, see the key, rate; the rating buttons carry the engine's intervals. The bank lists, filters, edits, imports (preview, then commit), and asks the model for drafts that are saved only when kept. Insights reads the review log.

The question card (`features/learn/cards/`) renders a `learn_quiz` tool result inside the tutor transcript. Answering calls `learn.path.answer`; the engine grades before the tutor's next turn and the card shows the verdict and the explanation.

## What is not here yet

- PDF pages are read as extracted text. `learn.document.blob` already returns the original bytes, but no pane renders them; highlights anchor to text units, not to PDF coordinates. Annotations store quote, offsets, and context, so a pdf.js text-layer view can be added without migrating them.
- No OCR. An image-only PDF page is reported `needs_ocr`. Some PDFs the parser cannot open at all; they are marked `failed` with its message.
- A library's embedding model is a name typed by hand, sent to the default connection. There is no picker and no per-library connection (`embed_conn` is migrated but unused).
- Import takes paths from the native file dialog. There is no drag-and-drop into the library.
- Exports (reading notes, a book) go to the clipboard as Markdown, not to a file.
- No audio or video materials.
- Prerequisites gate routing but do not yet pass implicit review credit down to the points they build on.
- FSRS runs on default weights. The review logs are complete enough to fit personal weights later.
