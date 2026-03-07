param(
  [string]$TaskName = 'HealthGoAutoStart',
  [switch]$Rebuild
)

$ErrorActionPreference = 'Stop'

function Write-Step($msg) {
  Write-Host "[安装开机自启动] $msg"
}

$projectRoot = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $projectRoot

$exePath = Join-Path $projectRoot 'health-go.exe'
$launcher = Join-Path $projectRoot 'scripts\启动服务.ps1'

if ($Rebuild -or -not (Test-Path $exePath)) {
  Write-Step 'Building health-go.exe ...'
  go build -o .\health-go.exe .\cmd\server
}

if (-not (Test-Path $launcher)) {
  throw "未找到启动脚本: $launcher"
}

$pwsh = (Get-Command powershell.exe -ErrorAction Stop).Source
$actionArgs = "-ExecutionPolicy Bypass -File `"$launcher`" -ProjectRoot `"$projectRoot`""

$action = New-ScheduledTaskAction -Execute $pwsh -Argument $actionArgs
$trigger = New-ScheduledTaskTrigger -AtStartup
$settings = New-ScheduledTaskSettingsSet `
  -AllowStartIfOnBatteries `
  -DontStopIfGoingOnBatteries `
  -MultipleInstances IgnoreNew `
  -StartWhenAvailable
$principal = New-ScheduledTaskPrincipal -UserId 'SYSTEM' -LogonType ServiceAccount -RunLevel Highest

try {
  $null = Get-ScheduledTask -TaskName $TaskName -ErrorAction Stop
  Write-Step "Updating existing scheduled task: $TaskName"
  Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false
} catch {
}

Register-ScheduledTask `
  -TaskName $TaskName `
  -Action $action `
  -Trigger $trigger `
  -Settings $settings `
  -Principal $principal | Out-Null

Write-Step "Installed startup task: $TaskName"
Write-Step "Executable: $exePath"
Write-Step "Launcher: $launcher"
Write-Step "You can test it now with: schtasks /run /tn $TaskName"
