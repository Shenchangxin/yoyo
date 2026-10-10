if (Test-Path role-in-harbor.txt) { exit 1 }
if (-not (Test-Path harbor-ok.txt)) { exit 1 }
if (-not (Select-String -Path harbor-ok.txt -Pattern 'ok' -Quiet)) { exit 1 }
exit 0
