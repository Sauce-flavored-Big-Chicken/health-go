@echo off
setlocal EnableExtensions

set "TASK_NAME=HealthGoAutoStart"
set "STOP_PROCESS=0"

:parse_args
if "%~1"=="" goto args_done
if /I "%~1"=="-TaskName" (
  if "%~2"=="" (
    echo [卸载开机自启动] 参数错误: -TaskName 后缺少任务名
    exit /b 1
  )
  set "TASK_NAME=%~2"
  shift
  shift
  goto parse_args
)
if /I "%~1"=="-StopProcess" (
  set "STOP_PROCESS=1"
  shift
  goto parse_args
)
echo [卸载开机自启动] 忽略未知参数: %~1
shift
goto parse_args

:args_done
schtasks /query /tn "%TASK_NAME%" >nul 2>nul
if errorlevel 1 (
  echo [卸载开机自启动] 未找到计划任务: %TASK_NAME%
) else (
  schtasks /delete /f /tn "%TASK_NAME%" >nul 2>nul
  if errorlevel 1 (
    echo [卸载开机自启动] 删除计划任务失败: %TASK_NAME%
    exit /b 1
  )
  echo [卸载开机自启动] 已删除计划任务: %TASK_NAME%
)

if "%STOP_PROCESS%"=="1" (
  tasklist /fi "imagename eq health-go.exe" | find /i "health-go.exe" >nul
  if errorlevel 1 (
    echo [卸载开机自启动] 当前没有运行中的 health-go.exe
  ) else (
    taskkill /f /im health-go.exe >nul 2>nul
    echo [卸载开机自启动] 已停止正在运行的 health-go.exe
  )
)

exit /b 0
