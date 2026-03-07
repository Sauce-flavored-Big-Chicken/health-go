param(
  [string]$TaskName = 'HealthGoAutoStart',
  [switch]$Rebuild
)

$projectRoot = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$cnScript = Join-Path $projectRoot 'scripts\安装开机自启动.ps1'
$argsList = @('-ExecutionPolicy', 'Bypass', '-File', $cnScript, '-TaskName', $TaskName)
if ($Rebuild) {
  $argsList += '-Rebuild'
}
& powershell.exe @argsList
