if (-not (Test-Path draft.txt)) { exit 1 }
if (Test-Path 'C:\Windows\yoyo-job.txt') { exit 1 }
if (Test-Path '/tmp/yoyo-job.txt') { exit 1 }
exit 0
