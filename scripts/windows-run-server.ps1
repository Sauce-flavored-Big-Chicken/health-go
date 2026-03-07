param(
  [string]$ProjectRoot
)

$projectRoot = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$cnScript = Join-Path $projectRoot 'scripts\启动服务.ps1'
& powershell.exe -ExecutionPolicy Bypass -File $cnScript -ProjectRoot $ProjectRoot
