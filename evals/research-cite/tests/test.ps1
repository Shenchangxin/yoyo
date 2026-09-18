if (-not (Test-Path report.md)) { exit 1 }
if (-not (Test-Path citations.json)) { exit 1 }
$c = Get-Content citations.json -Raw
if ($c -notmatch 'http') { exit 1 }
exit 0
