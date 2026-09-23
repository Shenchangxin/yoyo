package video

const schemaV1 = `
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS providers (
  id TEXT PRIMARY KEY,
  service_type TEXT NOT NULL,
  provider TEXT NOT NULL,
  name TEXT NOT NULL,
  base_url TEXT NOT NULL,
  vault_key TEXT NOT NULL,
  model TEXT NOT NULL DEFAULT '',
  models TEXT NOT NULL DEFAULT '[]',
  priority INTEGER NOT NULL DEFAULT 0,
  is_default INTEGER NOT NULL DEFAULT 0,
  is_active INTEGER NOT NULL DEFAULT 1,
  settings TEXT NOT NULL DEFAULT '{}',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS style_presets (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  value TEXT NOT NULL UNIQUE,
  prompt TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  sort_order INTEGER NOT NULL DEFAULT 0,
  is_active INTEGER NOT NULL DEFAULT 1,
  seeded INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS jobs (
  id TEXT PRIMARY KEY,
  type TEXT NOT NULL,
  status TEXT NOT NULL,
  provider TEXT NOT NULL DEFAULT '',
  model TEXT NOT NULL DEFAULT '',
  vault_key TEXT NOT NULL DEFAULT '',
  remote_id TEXT NOT NULL DEFAULT '',
  prompt TEXT NOT NULL DEFAULT '',
  params TEXT NOT NULL DEFAULT '{}',
  result_hash TEXT NOT NULL DEFAULT '',
  poster_hash TEXT NOT NULL DEFAULT '',
  error TEXT NOT NULL DEFAULT '',
  drama_id TEXT NOT NULL DEFAULT '',
  episode_id TEXT NOT NULL DEFAULT '',
  storyboard_id TEXT NOT NULL DEFAULT '',
  character_id TEXT NOT NULL DEFAULT '',
  scene_id TEXT NOT NULL DEFAULT '',
  prop_id TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  completed_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS jobs_status ON jobs(status);
CREATE INDEX IF NOT EXISTS jobs_episode ON jobs(episode_id);

CREATE TABLE IF NOT EXISTS dramas (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  genre TEXT NOT NULL DEFAULT '',
  style TEXT NOT NULL DEFAULT '3d',
  aspect_ratio TEXT NOT NULL DEFAULT '16:9',
  status TEXT NOT NULL DEFAULT 'draft',
  thumbnail_hash TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  deleted_at TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS episodes (
  id TEXT PRIMARY KEY,
  drama_id TEXT NOT NULL,
  episode_number INTEGER NOT NULL DEFAULT 1,
  title TEXT NOT NULL,
  content TEXT NOT NULL DEFAULT '',
  script_content TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'draft',
  video_hash TEXT NOT NULL DEFAULT '',
  poster_hash TEXT NOT NULL DEFAULT '',
  image_provider_id TEXT NOT NULL DEFAULT '',
  video_provider_id TEXT NOT NULL DEFAULT '',
  image_model TEXT NOT NULL DEFAULT '',
  video_model TEXT NOT NULL DEFAULT '',
  tts_provider_id TEXT NOT NULL DEFAULT '',
  tts_model TEXT NOT NULL DEFAULT '',
  resolution TEXT NOT NULL DEFAULT '720p',
  pipeline TEXT NOT NULL DEFAULT '{}',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  deleted_at TEXT NOT NULL DEFAULT '',
  UNIQUE(drama_id, episode_number)
);

CREATE TABLE IF NOT EXISTS characters (
  id TEXT PRIMARY KEY,
  drama_id TEXT NOT NULL,
  name TEXT NOT NULL,
  role TEXT NOT NULL DEFAULT '',
  appearance TEXT NOT NULL DEFAULT '',
  styling TEXT NOT NULL DEFAULT '',
  final_prompt TEXT NOT NULL DEFAULT '',
  image_hash TEXT NOT NULL DEFAULT '',
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  deleted_at TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS scenes (
  id TEXT PRIMARY KEY,
  drama_id TEXT NOT NULL,
  location TEXT NOT NULL,
  time_of_day TEXT NOT NULL DEFAULT '',
  prompt TEXT NOT NULL DEFAULT '',
  lighting TEXT NOT NULL DEFAULT '',
  final_prompt TEXT NOT NULL DEFAULT '',
  image_hash TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  deleted_at TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS props (
  id TEXT PRIMARY KEY,
  drama_id TEXT NOT NULL,
  name TEXT NOT NULL,
  type TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  final_prompt TEXT NOT NULL DEFAULT '',
  image_hash TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  deleted_at TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS episode_characters (
  episode_id TEXT NOT NULL,
  character_id TEXT NOT NULL,
  PRIMARY KEY (episode_id, character_id)
);
CREATE TABLE IF NOT EXISTS episode_scenes (
  episode_id TEXT NOT NULL,
  scene_id TEXT NOT NULL,
  PRIMARY KEY (episode_id, scene_id)
);
CREATE TABLE IF NOT EXISTS episode_props (
  episode_id TEXT NOT NULL,
  prop_id TEXT NOT NULL,
  PRIMARY KEY (episode_id, prop_id)
);

CREATE TABLE IF NOT EXISTS storyboards (
  id TEXT PRIMARY KEY,
  episode_id TEXT NOT NULL,
  scene_id TEXT NOT NULL DEFAULT '',
  shot_number INTEGER NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  shot_type TEXT NOT NULL DEFAULT '',
  angle TEXT NOT NULL DEFAULT '',
  movement TEXT NOT NULL DEFAULT '',
  atmosphere TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  video_prompt TEXT NOT NULL DEFAULT '',
  duration INTEGER NOT NULL DEFAULT 10,
  video_hash TEXT NOT NULL DEFAULT '',
  poster_hash TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'pending',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  deleted_at TEXT NOT NULL DEFAULT '',
  UNIQUE(episode_id, shot_number)
);

CREATE TABLE IF NOT EXISTS storyboard_characters (
  storyboard_id TEXT NOT NULL,
  character_id TEXT NOT NULL,
  PRIMARY KEY (storyboard_id, character_id)
);
CREATE TABLE IF NOT EXISTS storyboard_props (
  storyboard_id TEXT NOT NULL,
  prop_id TEXT NOT NULL,
  PRIMARY KEY (storyboard_id, prop_id)
);

CREATE TABLE IF NOT EXISTS session_binds (
  session_id TEXT PRIMARY KEY,
  episode_id TEXT NOT NULL,
  drama_id TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS merges (
  id TEXT PRIMARY KEY,
  episode_id TEXT NOT NULL,
  drama_id TEXT NOT NULL,
  status TEXT NOT NULL,
  shot_ids TEXT NOT NULL DEFAULT '[]',
  result_hash TEXT NOT NULL DEFAULT '',
  poster_hash TEXT NOT NULL DEFAULT '',
  error TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  completed_at TEXT NOT NULL DEFAULT ''
);
`
