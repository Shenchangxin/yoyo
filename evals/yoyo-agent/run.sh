#!/bin/sh
# Harbor agent wrapper: yoyo run --workspace <ws> "<instruction>" (see docs/architecture/harbor.md).
set -e
WS=${1:-/app}
INST=${2:-instruction.md}
if [ ! -f "$INST" ] && [ -f /instruction.md ]; then
  INST=/instruction.md
fi
exec yoyo run --workspace "$WS" "$(cat "$INST")"
