package video

const schemaV2 = `
CREATE TABLE IF NOT EXISTS canvas_projects (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL DEFAULT '',
  payload_json TEXT NOT NULL DEFAULT '{}',
  timeline_json TEXT NOT NULL DEFAULT '',
  revision INTEGER NOT NULL DEFAULT 1,
  workspace_mode TEXT NOT NULL DEFAULT 'simple',
  theme TEXT NOT NULL DEFAULT '',
  project_link TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  deleted_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS canvas_projects_updated ON canvas_projects(updated_at);

CREATE TABLE IF NOT EXISTS canvas_snapshots (
  id TEXT PRIMARY KEY,
  canvas_id TEXT NOT NULL,
  revision INTEGER NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  payload_json TEXT NOT NULL DEFAULT '{}',
  reason TEXT NOT NULL DEFAULT 'automatic',
  node_count INTEGER NOT NULL DEFAULT 0,
  connection_count INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS canvas_snapshots_canvas ON canvas_snapshots(canvas_id, created_at);

CREATE TABLE IF NOT EXISTS canvas_resources (
  id TEXT PRIMARY KEY,
  cas_hash TEXT NOT NULL,
  kind TEXT NOT NULL DEFAULT 'file',
  mime TEXT NOT NULL DEFAULT '',
  bytes INTEGER NOT NULL DEFAULT 0,
  width INTEGER NOT NULL DEFAULT 0,
  height INTEGER NOT NULL DEFAULT 0,
  duration_ms INTEGER NOT NULL DEFAULT 0,
  poster_hash TEXT NOT NULL DEFAULT '',
  playback_status TEXT NOT NULL DEFAULT 'none',
  file_name TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  deleted_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS canvas_resources_hash ON canvas_resources(cas_hash);

CREATE TABLE IF NOT EXISTS canvas_asset_folders (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  position INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  deleted_at TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS canvas_assets (
  id TEXT PRIMARY KEY,
  folder_id TEXT NOT NULL DEFAULT '',
  kind TEXT NOT NULL DEFAULT 'image',
  category TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL DEFAULT '',
  resource_id TEXT NOT NULL DEFAULT '',
  payload_json TEXT NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'ready',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  deleted_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS canvas_assets_folder ON canvas_assets(folder_id);
CREATE INDEX IF NOT EXISTS canvas_assets_kind ON canvas_assets(kind);

CREATE TABLE IF NOT EXISTS canvas_channels (
  id TEXT PRIMARY KEY,
  plugin_id TEXT NOT NULL DEFAULT '',
  name TEXT NOT NULL,
  capability TEXT NOT NULL DEFAULT 'image',
  base_url TEXT NOT NULL DEFAULT '',
  vault_key TEXT NOT NULL DEFAULT '',
  model TEXT NOT NULL DEFAULT '',
  models TEXT NOT NULL DEFAULT '[]',
  settings TEXT NOT NULL DEFAULT '{}',
  sort_order INTEGER NOT NULL DEFAULT 0,
  enabled INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS canvas_session_binds (
  session_id TEXT PRIMARY KEY,
  canvas_id TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS canvas_lessons (
  id TEXT PRIMARY KEY,
  topic TEXT NOT NULL,
  category TEXT NOT NULL DEFAULT 'other',
  situation TEXT NOT NULL DEFAULT '',
  lesson TEXT NOT NULL DEFAULT '',
  steps_json TEXT NOT NULL DEFAULT '[]',
  status TEXT NOT NULL DEFAULT 'pending',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS canvas_uploads (
  id TEXT PRIMARY KEY,
  kind TEXT NOT NULL,
  file_name TEXT NOT NULL DEFAULT '',
  mime TEXT NOT NULL DEFAULT '',
  size INTEGER NOT NULL DEFAULT 0,
  chunk_size INTEGER NOT NULL DEFAULT 0,
  chunk_count INTEGER NOT NULL DEFAULT 0,
  width INTEGER NOT NULL DEFAULT 0,
  height INTEGER NOT NULL DEFAULT 0,
  duration_ms INTEGER NOT NULL DEFAULT 0,
  received INTEGER NOT NULL DEFAULT 0,
  dir TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS canvas_task_logs (
  id TEXT PRIMARY KEY,
  task_id TEXT NOT NULL,
  level TEXT NOT NULL DEFAULT 'info',
  stage TEXT NOT NULL DEFAULT '',
  message TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS canvas_task_logs_task ON canvas_task_logs(task_id);

CREATE TABLE IF NOT EXISTS canvas_text_deltas (
  id TEXT PRIMARY KEY,
  task_id TEXT NOT NULL,
  seq INTEGER NOT NULL,
  content TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS canvas_text_deltas_task ON canvas_text_deltas(task_id, seq);

CREATE TABLE IF NOT EXISTS canvas_project_units (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL,
  kind TEXT NOT NULL DEFAULT 'chapter',
  title TEXT NOT NULL DEFAULT '',
  payload_json TEXT NOT NULL DEFAULT '{}',
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  deleted_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS canvas_project_units_project ON canvas_project_units(project_id);
`
