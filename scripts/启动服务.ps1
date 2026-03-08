param(
  [string]$ProjectRoot
)

$projectRoot = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$batScript = Join-Path $projectRoot 'scripts\启动服务.bat'
if ($ProjectRoot) {
  & cmd.exe /c $batScript $ProjectRoot
} else {
  & cmd.exe /c $batScript
}
