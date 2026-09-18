if (-not (Test-Path forgotten.txt)) { exit 1 }
$c = Get-Content forgotten.txt -Raw
if ($c -notmatch 'forgotten') { exit 1 }
Get-ChildItem -Recurse -File | ForEach-Object { if ((Get-Content $_.FullName -Raw -ErrorAction SilentlyContinue) -match 'SECRET_TOKEN_XYZ') { exit 1 } }
exit 0
