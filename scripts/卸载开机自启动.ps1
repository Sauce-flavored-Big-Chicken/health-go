param(
  [string]$TaskName = 'HealthGoAutoStart',
  [switch]$StopProcess
)

$projectRoot = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$batScript = Join-Path $projectRoot 'scripts\卸载开机自启动.bat'
$argsList = @('-TaskName', $TaskName)
if ($StopProcess) {
  $argsList += '-StopProcess'
}
& cmd.exe /c $batScript @argsList
