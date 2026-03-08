@echo off
setlocal EnableExtensions

set "TASK_NAME=HealthGoAutoStartVisible"

:parse_args
if "%~1"=="" goto args_done
if /I "%~1"=="-TaskName" (
  if "%~2"=="" (
    echo [卸载登录自启动可视窗口] 参数错误: -TaskName 后缺少任务名
    exit /b 1
  )
  set "TASK_NAME=%~2"
  shift
  shift
  goto parse_args
)
echo [卸载登录自启动可视窗口] 忽略未知参数: %~1
shift
goto parse_args

:args_done
schtasks /query /tn "%TASK_NAME%" >nul 2>nul
if errorlevel 1 (
  echo [卸载登录自启动可视窗口] 未找到计划任务: %TASK_NAME%
  exit /b 0
)

schtasks /delete /f /tn "%TASK_NAME%" >nul 2>nul
if errorlevel 1 (
  echo [卸载登录自启动可视窗口] 删除计划任务失败: %TASK_NAME%
  exit /b 1
)

echo [卸载登录自启动可视窗口] 已删除计划任务: %TASK_NAME%
exit /b 0
