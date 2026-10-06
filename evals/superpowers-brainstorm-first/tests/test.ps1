if (-not (Test-Path 'docs/design.md')) { exit 1 }
$c = Get-Content 'docs/design.md' -Raw
if ($c -notmatch 'approach') { exit 1 }
exit 0
