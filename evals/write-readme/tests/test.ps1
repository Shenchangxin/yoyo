if (-not (Test-Path README.md)) { exit 1 }
$c = Get-Content README.md -Raw
if ($c -notmatch 'yoyo') { exit 1 }
exit 0
