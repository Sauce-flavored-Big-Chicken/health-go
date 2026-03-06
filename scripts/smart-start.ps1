param(
  [switch]$Build,
  [switch]$NoMirror,
  [string]$Addr
)

$ErrorActionPreference = 'Stop'

function Write-Step($msg) {
  Write-Host "[smart-start] $msg"
}

$projectRoot = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $projectRoot

# Load .env if exists (without overriding existing process env)
$envFile = Join-Path $projectRoot '.env'
if (Test-Path $envFile) {
  Get-Content $envFile | ForEach-Object {
    $line = $_.Trim()
    if (-not $line -or $line.StartsWith('#')) { return }
    $idx = $line.IndexOf('=')
    if ($idx -lt 1) { return }
    $k = $line.Substring(0, $idx).Trim()
    $v = $line.Substring($idx + 1).Trim().Trim('"').Trim("'")
    if (-not [string]::IsNullOrWhiteSpace($k) -and -not (Test-Path env:$k)) {
      Set-Item -Path "env:$k" -Value $v
    }
  }
  Write-Step 'Loaded .env'
}

if ($Addr) {
  $env:APP_ADDR = $Addr
}

# Configure Go proxy smartly
if (-not $NoMirror) {
  $candidate = 'https://proxy.golang.org'
  $ok = $false
  try {
    $resp = Invoke-WebRequest -Uri $candidate -Method Head -TimeoutSec 2 -UseBasicParsing
    if ($resp.StatusCode -ge 200 -and $resp.StatusCode -lt 500) { $ok = $true }
  } catch {
    $ok = $false
  }

  if ($ok) {
    $env:GOPROXY = 'https://proxy.golang.org,direct'
    Write-Step "Network looks global, using GOPROXY=$($env:GOPROXY)"
  } else {
    $env:GOPROXY = 'https://goproxy.cn,direct'
    Write-Step "Network looks domestic/restricted, using GOPROXY=$($env:GOPROXY)"
  }
}

if (-not (Test-Path env:DB_DRIVER)) { $env:DB_DRIVER = 'sqlite' }
if ($env:DB_DRIVER -eq 'sqlite' -and -not (Test-Path 'health.db')) {
  Write-Step 'health.db not found, creating from ../health/health.sql ...'
  go run .\cmd\sqlite_copy
}

Write-Step 'Running go mod tidy (for first run/dependency sync) ...'
go mod tidy

if ($Build) {
  Write-Step 'Building server binary ...'
  go build -o .\health-go.exe .\cmd\server
  Write-Step 'Starting .\\health-go.exe'
  .\health-go.exe
} else {
  Write-Step 'Starting with go run ...'
  go run .\cmd\server
}
