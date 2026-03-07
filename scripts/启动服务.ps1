param(
  [string]$ProjectRoot
)

$ErrorActionPreference = 'Stop'

if (-not $ProjectRoot) {
  $ProjectRoot = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
}

Set-Location $ProjectRoot

$logDir = Join-Path $ProjectRoot 'logs'
if (-not (Test-Path $logDir)) {
  New-Item -ItemType Directory -Path $logDir | Out-Null
}

$stdoutLog = Join-Path $logDir 'server.stdout.log'
$stderrLog = Join-Path $logDir 'server.stderr.log'
$exePath = Join-Path $ProjectRoot 'health-go.exe'
$envFile = Join-Path $ProjectRoot '.env'

if (-not (Test-Path $exePath)) {
  throw "未找到 $exePath，请先执行 安装开机自启动.ps1 生成可执行文件。"
}

if (Test-Path $envFile) {
  Get-Content $envFile | ForEach-Object {
    $line = $_.Trim()
    if (-not $line -or $line.StartsWith('#')) { return }
    $idx = $line.IndexOf('=')
    if ($idx -lt 1) { return }
    $k = $line.Substring(0, $idx).Trim()
    $v = $line.Substring($idx + 1).Trim().Trim('"').Trim("'")
    if (-not [string]::IsNullOrWhiteSpace($k)) {
      Set-Item -Path "env:$k" -Value $v
    }
  }
}

$running = Get-Process -Name 'health-go' -ErrorAction SilentlyContinue
if ($running) {
  exit 0
}

Start-Process -FilePath $exePath `
  -WorkingDirectory $ProjectRoot `
  -RedirectStandardOutput $stdoutLog `
  -RedirectStandardError $stderrLog `
  -WindowStyle Hidden
