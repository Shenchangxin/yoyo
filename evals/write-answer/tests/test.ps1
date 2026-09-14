if (-not (Test-Path answer.txt)) { exit 1 }
$c = Get-Content answer.txt -Raw
if ($c -notmatch '42') { exit 1 }
exit 0
