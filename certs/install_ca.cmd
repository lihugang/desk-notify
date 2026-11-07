@echo off
chcp 65001 >nul
title Lihugang 个人根证书信任安装工具

echo ========================================================
echo        Lihugang 个人根证书 (Root CA) 一键信任安装
echo ========================================================
echo.
echo 本脚本将把 Lihugang 个人统一根证书导入 Windows 受信任存储区。
echo 导入一次后，由该作者发布的所有软件 (如传纸条 DeskNotify 及未来应用)
echo 均将受到系统永久信任，不再提示“未知发布者”，局域网 HTTPS 也将正常信任。
echo.

:: 检查是否具备管理员权限，若无则自动请求 UAC 提权
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo [提示] 正在请求系统管理员权限...
    powershell -Command "Start-Process cmd -ArgumentList '/c \"%~f0\"' -Verb RunAs"
    exit /b
)

cd /d "%~dp0"
if not exist "Lihugang-CA.cer" (
    echo [错误] 未在当前目录下找到 Lihugang-CA.cer 证书文件！
    echo 请确保 Lihugang-CA.cer 与此脚本放置在同一文件夹下。
    echo.
    pause
    exit /b 1
)

echo [1/2] 正在将根证书导入“受信任的根证书颁发机构”...
certutil -addstore -f "Root" "Lihugang-CA.cer" >nul 2>&1
if %errorlevel% neq 0 (
    echo [警告] 导入根证书存储区失败，错误码: %errorlevel%
) else (
    echo [成功] Lihugang 根证书已成功导入受信任的根证书颁发机构 (Root)！
)

echo.
echo [2/2] 正在将发布者导入“受信任的发布者”...
certutil -addstore -f "TrustedPublisher" "Lihugang-CA.cer" >nul 2>&1
if %errorlevel% neq 0 (
    echo [警告] 导入受信任发布者失败，错误码: %errorlevel%
) else (
    echo [成功] 发布者已成功信任 (TrustedPublisher)！
)

echo.
echo ========================================================
echo   证书安装完成！
echo   现在您可以畅快运行该作者发布的任何应用，无需担心拦截提示。
echo ========================================================
echo.
pause
