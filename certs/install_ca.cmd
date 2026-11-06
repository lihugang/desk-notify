@echo off
chcp 65001 >nul
title 传纸条 (DeskNotify) - 根证书信任工具

echo ========================================================
echo        传纸条 (DeskNotify) - 受信任根证书安装
echo ========================================================
echo.
echo 本脚本将把 传纸条 (DeskNotify) 根证书导入 Windows 受信任存储区。
echo 导入后，DeskNotify 软件将不再提示“未知发布者”，局域网 HTTPS 也将受信任。
echo.

:: 检查是否具备管理员权限，若无则自动请求 UAC 提权
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo [提示] 正在请求系统管理员权限...
    powershell -Command "Start-Process cmd -ArgumentList '/c \"%~f0\"' -Verb RunAs"
    exit /b
)

cd /d "%~dp0"
if not exist "DeskNotify-CA.cer" (
    echo [错误] 未在当前目录下找到 DeskNotify-CA.cer 证书文件！
    echo 请确保 DeskNotify-CA.cer 与此脚本放置在同一文件夹下。
    echo.
    pause
    exit /b 1
)

echo [1/2] 正在将根证书导入“受信任的根证书颁发机构”...
certutil -addstore -f "Root" "DeskNotify-CA.cer" >nul 2>&1
if %errorlevel% neq 0 (
    echo [警告] 导入根证书存储区失败，错误码: %errorlevel%
) else (
    echo [成功] 根证书已成功导入受信任的根证书颁发机构 (Root)！
)

echo.
echo [2/2] 正在将发布者导入“受信任的发布者”...
certutil -addstore -f "TrustedPublisher" "DeskNotify-CA.cer" >nul 2>&1
if %errorlevel% neq 0 (
    echo [警告] 导入受信任发布者失败，错误码: %errorlevel%
) else (
    echo [成功] 发布者已成功信任 (TrustedPublisher)！
)

echo.
echo ========================================================
echo   证书安装完成！
echo   现在您可以畅快运行 DeskNotify.exe，无需担心拦截提示。
echo ========================================================
echo.
pause
