@echo off
setlocal EnableExtensions EnableDelayedExpansion

set "PROJECT_ROOT="
set "RUN_MODE=FOREGROUND"

:parse_args
if "%~1"=="" goto args_done
if /I "%~1"=="-Background" (
  set "RUN_MODE=BACKGROUND"
  shift
  goto parse_args
)
if "%PROJECT_ROOT%"=="" (
  set "PROJECT_ROOT=%~1"
  shift
  goto parse_args
)
echo [启动服务] 忽略未知参数: %~1
shift
goto parse_args

:args_done
if "%PROJECT_ROOT%"=="" (
  set "SCRIPT_DIR=%~dp0"
  for %%I in ("%SCRIPT_DIR%..") do set "PROJECT_ROOT=%%~fI"
)

cd /d "%PROJECT_ROOT%" >nul 2>nul
if errorlevel 1 (
  echo [启动服务] 项目目录不存在: %PROJECT_ROOT%
  exit /b 1
)

set "LOG_DIR=%PROJECT_ROOT%\logs"
if not exist "%LOG_DIR%" mkdir "%LOG_DIR%" >nul 2>nul

set "STDOUT_LOG=%LOG_DIR%\server.stdout.log"
set "STDERR_LOG=%LOG_DIR%\server.stderr.log"
set "EXE_PATH=%PROJECT_ROOT%\health-go.exe"
set "ENV_FILE=%PROJECT_ROOT%\.env"

if not exist "%EXE_PATH%" (
  echo [启动服务] 未找到 %EXE_PATH%，请先执行 安装开机自启动.bat 生成可执行文件。
  exit /b 1
)

if exist "%ENV_FILE%" (
  for /f "usebackq eol=# tokens=1* delims==" %%A in ("%ENV_FILE%") do (
    set "K=%%~A"
    set "V=%%~B"
    if not "!K!"=="" (
      if defined V (
        if "!V:~0,1!"=="^"" if "!V:~-1!"=="^"" set "V=!V:~1,-1!"
        if "!V:~0,1!"=="'" if "!V:~-1!"=="'" set "V=!V:~1,-1!"
      )
      set "!K!=!V!"
    )
  )
)

tasklist /fi "imagename eq health-go.exe" | find /i "health-go.exe" >nul
if not errorlevel 1 (
  echo [启动服务] health-go.exe 已在运行
  exit /b 0
)

if /I "%RUN_MODE%"=="BACKGROUND" (
  start "" /min cmd /c "\"%EXE_PATH%\" 1>>\"%STDOUT_LOG%\" 2>>\"%STDERR_LOG%\""
  echo [启动服务] 后台启动完成，可查看日志: %STDOUT_LOG%
  exit /b 0
)

echo [启动服务] 前台启动服务，实时日志将输出到当前终端
"%EXE_PATH%"
exit /b 0
