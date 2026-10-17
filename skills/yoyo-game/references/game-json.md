# Playable contract

`playable/game.json` is the only file the sealed runtime reads.

Required fields:

- `schema_version` (1)
- `objective`, `win`, `fail`
- `need_collect`
- `camera`: `tps` | `fps` | `top`
- `player.position` `[x,y,z]`
- `scene` heightfield (`width`, `depth`, `cell_size`, `heights`, `surfaces`)
- `elements[]` with `id`, `type`, `position`
- `interactions[]` with `id`, `participants`, `trigger`, `precondition`, `delta`

`delta` whitelist: `score`, `hp`, `flag`, `inventory`, `win`, `fail`, `teleport`, `teleport_to`.

Logical types (`player_spawn`, `*_spawn`, `*_anchor`) must not become meshes. Models pick `candidate_id` only. Never invent `world_xyz`.
