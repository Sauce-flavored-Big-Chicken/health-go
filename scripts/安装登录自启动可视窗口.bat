@echo off
setlocal EnableExtensions EnableDelayedExpansion

set "TASK_NAME=HealthGoAutoStartVisible"
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
    echo [安装登录自启动可视窗口] 参数错误: -TaskName 后缺少任务名
    exit /b 1
  )
  set "TASK_NAME=%~2"
  shift
  shift
  goto parse_args
)
echo [安装登录自启动可视窗口] 忽略未知参数: %~1
shift
goto parse_args

:args_done
set "SCRIPT_DIR=%~dp0"
for %%I in ("%SCRIPT_DIR%..") do set "PROJECT_ROOT=%%~fI"
set "EXE_PATH=%PROJECT_ROOT%\health-go.exe"
set "LAUNCHER=%PROJECT_ROOT%\scripts\启动服务可视窗口.bat"

if "%REBUILD%"=="1" goto do_build
if exist "%EXE_PATH%" goto after_build

:do_build
echo [安装登录自启动可视窗口] Building health-go.exe ...
pushd "%PROJECT_ROOT%" >nul
go build -o .\health-go.exe .\cmd\server
if errorlevel 1 (
  popd >nul
  echo [安装登录自启动可视窗口] 编译失败
  exit /b 1
)
popd >nul

:after_build
if not exist "%LAUNCHER%" (
  echo [安装登录自启动可视窗口] 未找到启动脚本: %LAUNCHER%
  exit /b 1
)

schtasks /query /tn "%TASK_NAME%" >nul 2>nul
if not errorlevel 1 (
  echo [安装登录自启动可视窗口] Updating existing scheduled task: %TASK_NAME%
  schtasks /delete /f /tn "%TASK_NAME%" >nul 2>nul
)

set "TASK_CMD=\"%SystemRoot%\System32\cmd.exe\" /k \"\"%LAUNCHER%\" \"%PROJECT_ROOT%\"\""
schtasks /create /f /tn "%TASK_NAME%" /sc ONLOGON /it /rl HIGHEST /tr %TASK_CMD% >nul
if errorlevel 1 (
  echo [安装登录自启动可视窗口] 创建计划任务失败
  echo [安装登录自启动可视窗口] 提示: 首次创建可能需要管理员权限
  exit /b 1
)

echo [安装登录自启动可视窗口] Installed visible task: %TASK_NAME%
echo [安装登录自启动可视窗口] 登录后会弹出终端窗口并显示实时请求日志
echo [安装登录自启动可视窗口] 可测试: schtasks /run /tn %TASK_NAME%
exit /b 0
