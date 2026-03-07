param(
  [string]$TaskName = 'HealthGoAutoStart',
  [switch]$StopProcess
)

$ErrorActionPreference = 'Stop'

function Write-Step($msg) {
  Write-Host "[卸载开机自启动] $msg"
}

try {
  $null = Get-ScheduledTask -TaskName $TaskName -ErrorAction Stop
  Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false
  Write-Step "已删除计划任务: $TaskName"
} catch {
  Write-Step "未找到计划任务: $TaskName"
}

if ($StopProcess) {
  $processes = Get-Process -Name 'health-go' -ErrorAction SilentlyContinue
  if ($processes) {
    $processes | Stop-Process -Force
    Write-Step '已停止正在运行的 health-go.exe'
  } else {
    Write-Step '当前没有运行中的 health-go.exe'
  }
}
