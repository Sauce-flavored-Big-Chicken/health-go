@echo off
setlocal EnableExtensions EnableDelayedExpansion

set "TASK_NAME=HealthGoAutoStart"
set "REBUILD=0"

:parse_args
if "%~1"=="" goto args_done
if /I "%~1"=="-Rebuild" (
  set "REBUILD=1"
  shift
  goto parse_args
)
if /I "%~1"=="-TaskName" (
  if "%~2"=="" (
    echo [安装开机自启动] 参数错误: -TaskName 后缺少任务名
    exit /b 1
  )
  set "TASK_NAME=%~2"
  shift
  shift
  goto parse_args
)
echo [安装开机自启动] 忽略未知参数: %~1
shift
goto parse_args

:args_done
set "SCRIPT_DIR=%~dp0"
for %%I in ("%SCRIPT_DIR%..") do set "PROJECT_ROOT=%%~fI"
set "EXE_PATH=%PROJECT_ROOT%\health-go.exe"
set "LAUNCHER=%PROJECT_ROOT%\scripts\启动服务.bat"

if "%REBUILD%"=="1" goto do_build
if exist "%EXE_PATH%" goto after_build

:do_build
echo [安装开机自启动] Building health-go.exe ...
pushd "%PROJECT_ROOT%" >nul
go build -o .\health-go.exe .\cmd\server
if errorlevel 1 (
  popd >nul
  echo [安装开机自启动] 编译失败
  exit /b 1
)
popd >nul

:after_build
if not exist "%LAUNCHER%" (
  echo [安装开机自启动] 未找到启动脚本: %LAUNCHER%
  exit /b 1
)

schtasks /query /tn "%TASK_NAME%" >nul 2>nul
if not errorlevel 1 (
  echo [安装开机自启动] Updating existing scheduled task: %TASK_NAME%
  schtasks /delete /f /tn "%TASK_NAME%" >nul 2>nul
)

set "TASK_CMD=\"%SystemRoot%\System32\cmd.exe\" /c \"\"%LAUNCHER%\" \"%PROJECT_ROOT%\" -Background\""
schtasks /create /f /tn "%TASK_NAME%" /sc ONSTART /ru SYSTEM /rl HIGHEST /tr %TASK_CMD% >nul
if errorlevel 1 (
  echo [安装开机自启动] 创建计划任务失败
  exit /b 1
)

echo [安装开机自启动] Installed startup task: %TASK_NAME%
echo [安装开机自启动] Executable: %EXE_PATH%
echo [安装开机自启动] Launcher: %LAUNCHER%
echo [安装开机自启动] You can test it now with: schtasks /run /tn %TASK_NAME%
exit /b 0
