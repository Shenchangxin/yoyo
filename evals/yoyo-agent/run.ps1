param(
  [string]$Workspace = "/app",
  [string]$Instruction = "instruction.md"
)
if (-not (Test-Path $Instruction) -and (Test-Path "/instruction.md")) {
  $Instruction = "/instruction.md"
}
$msg = Get-Content -Raw $Instruction
& yoyo run --workspace $Workspace $msg
