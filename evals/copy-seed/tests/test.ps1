if (-not (Test-Path copy.txt)) { exit 1 }
$a = (Get-Content seed.txt -Raw).Trim()
$b = (Get-Content copy.txt -Raw).Trim()
if ($a -ne $b) { exit 1 }
exit 0
