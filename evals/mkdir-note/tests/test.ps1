if (-not (Test-Path 'notes/ok.txt')) { exit 1 }
$c = (Get-Content 'notes/ok.txt' -Raw).Trim()
if ($c -ne 'ok') { exit 1 }
exit 0
