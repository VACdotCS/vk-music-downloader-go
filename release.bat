@echo off
chcp 65001 > nul
echo 🔨 Начинаем сборку релизных версий...

if not exist "release" mkdir release

echo.
echo 🪟  [1/3] CLI — Windows (amd64)...
set GOOS=windows
set GOARCH=amd64
go build -ldflags="-s -w" -o release\vk-music-downloader-cli-windows-amd64.exe cmd\cli\main.go
if errorlevel 1 ( echo ❌ Ошибка сборки CLI Windows & pause & exit /b 1 )

echo 🐧  [2/3] CLI — Linux (amd64)...
set GOOS=linux
set GOARCH=amd64
go build -ldflags="-s -w" -o release\vk-music-downloader-cli-linux-amd64 cmd\cli\main.go
if errorlevel 1 ( echo ❌ Ошибка сборки CLI Linux & pause & exit /b 1 )

echo 🖥️  [3/3] GUI — Windows (wails build)...
set GOOS=
set GOARCH=
cd cmd\gui
wails build -ldflags "-s -w" -o ..\..\release\vk-music-downloader-gui-windows-amd64.exe
if errorlevel 1 ( echo ❌ Ошибка сборки GUI & cd ..\.. & pause & exit /b 1 )
cd ..\..

echo.
echo ✅ Готово! Бинарники лежат в папке release\
echo.
dir /b release\*.exe release\*-linux-amd64 2>nul
pause
