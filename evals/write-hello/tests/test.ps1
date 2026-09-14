if (-not (Test-Path hello.txt)) { exit 1 }
$c = Get-Content hello.txt -Raw
if ($c -notmatch 'hello') { exit 1 }
exit 0
