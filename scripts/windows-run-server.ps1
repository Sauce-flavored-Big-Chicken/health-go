param(
  [string]$ProjectRoot
)

$projectRoot = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$cnScript = Join-Path $projectRoot 'scripts\启动服务.bat'
if ($ProjectRoot) {
  & cmd.exe /c $cnScript $ProjectRoot
} else {
  & cmd.exe /c $cnScript
}
