# Fetch arXiv PDFs listed in docs/research/catalog.yaml into docs/research/originals/.
# Usage (from repo root): powershell -File scripts/fetch-research-originals.ps1

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$dest = Join-Path $root "docs\research\originals"
$catalog = Join-Path $root "docs\research\catalog.yaml"
New-Item -ItemType Directory -Force -Path $dest | Out-Null

$ids = Select-String -Path $catalog -Pattern 'id:\s+"(\d{4}\.\d{5})"' | ForEach-Object { $_.Matches[0].Groups[1].Value } | Select-Object -Unique
Write-Host "Fetching $($ids.Count) PDFs into $dest"

foreach ($id in $ids) {
  $out = Join-Path $dest "$id.pdf"
  if ((Test-Path $out) -and ((Get-Item $out).Length -gt 10000)) {
    Write-Host "skip $id"
    continue
  }
  $url = "https://arxiv.org/pdf/$id.pdf"
  Write-Host "get  $id"
  try {
    Invoke-WebRequest -Uri $url -OutFile $out -TimeoutSec 120 -UseBasicParsing
  } catch {
    Write-Host "FAIL $id : $_"
  }
}

Get-ChildItem $dest -Filter *.pdf | ForEach-Object {
  "{0,8:N1} KB  {1}" -f ($_.Length / 1KB), $_.Name
}
