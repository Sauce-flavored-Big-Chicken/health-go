param(
  [string]$TaskName = 'HealthGoAutoStart',
  [switch]$Rebuild
)

$projectRoot = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$cnScript = Join-Path $projectRoot 'scripts\安装开机自启动.bat'
$argsList = @('-TaskName', $TaskName)
if ($Rebuild) {
  $argsList += '-Rebuild'
}
& cmd.exe /c $cnScript @argsList
