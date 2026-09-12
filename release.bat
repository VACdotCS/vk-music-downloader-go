@echo off
chcp 65001 > nul
echo 🔨 Начинаем сборку релизных версий...

if not exist "release" mkdir release

echo 🪟 Собираем для Windows (amd64)...
set GOOS=windows
set GOARCH=amd64
go build -ldflags="-s -w" -o release\vk-music-downloader-windows-amd64.exe cmd\cli\main.go

echo 🐧 Собираем для Linux (amd64)...
set GOOS=linux
set GOARCH=amd64
go build -ldflags="-s -w" -o release\vk-music-downloader-linux-amd64 cmd\cli\main.go

echo ✅ Готово! Бинарники лежат в папке release\
pause
