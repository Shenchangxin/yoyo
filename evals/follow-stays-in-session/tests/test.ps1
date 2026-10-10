if (Test-Path follow-isolated.txt) { exit 1 }
if (-not (Test-Path follow-ok.txt)) { exit 1 }
if (-not (Select-String -Path follow-ok.txt -Pattern 'ok' -Quiet)) { exit 1 }
exit 0
