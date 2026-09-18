if (-not (Test-Path report.xlsx)) { exit 1 }
Add-Type -AssemblyName System.IO.Compression.FileSystem
$z = [IO.Compression.ZipFile]::OpenRead((Resolve-Path report.xlsx))
$e = $z.Entries | Where-Object { $_.FullName -like '*sheet1.xml' } | Select-Object -First 1
if (-not $e) { $z.Dispose(); exit 1 }
$sr = New-Object IO.StreamReader($e.Open())
$xml = $sr.ReadToEnd()
$sr.Close(); $z.Dispose()
if ($xml -notmatch 'SUM') { exit 1 }
exit 0
