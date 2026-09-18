if (-not (Test-Path deck.pptx)) { exit 1 }
Add-Type -AssemblyName System.IO.Compression.FileSystem
$z = [IO.Compression.ZipFile]::OpenRead((Resolve-Path deck.pptx))
$n = ($z.Entries | Where-Object { $_.FullName -like 'ppt/slides/slide*.xml' }).Count
$z.Dispose()
if ($n -lt 3) { exit 1 }
exit 0
