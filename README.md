# VK Music Downloader (Golang)

🇬🇧 [English](#english) | 🇷🇺 [Русский](#русский)

---

<a name="english"></a>
## 🇬🇧 English

A fast, fully standalone utility for downloading music and playlists from VK (Vkontakte), rewritten in Go. Available in two flavors: **CLI** (terminal) and **GUI** (modern desktop app built with Wails + React). Originally based on the [Node.js version by VACdotCS](https://github.com/VACdotCS/vk-music-downloader).

### ✨ Features
* **Two interfaces**: A classic interactive CLI and a modern GUI desktop app with playlist cards, progress bars, and cover art.
* **Zero Dependencies**: A single `.exe` is all you need. FFmpeg is downloaded automatically if missing.
* **Blazing Fast**: Go goroutines + custom worker pool for concurrent downloading.
* **Smart Playlist UI**: Visual progress bars segmented into success (blue) and blocked tracks (red). Fully downloaded playlists open their folder on click.
* **On-the-fly Decryption**: Native HLS (`.m3u8`) stream parsing and AES-128 decryption.
* **Graceful Cancellation**: `Ctrl+C` (CLI) or Stop button (GUI) instantly aborts all downloads and cleans up temp files.
* **Copyright Log**: After each playlist download, a `Ошибки_авторских_прав.txt` file is created listing any tracks blocked by rights holders.

### 🚀 Usage
1. Download the latest binary for your OS from the [Releases page](https://github.com/VACdotCS/vk-music-downloader-go/releases/latest):
   - `vk-music-downloader-cli-windows-amd64.exe` — terminal interface
   - `vk-music-downloader-gui-windows-amd64.exe` — graphical interface
2. Run the executable. The app is **fully portable** — all config files and caches are created **in the same folder** as the binary.
3. Paste your VK Kate Mobile `access_token` JSON when prompted.

> **⚠️ Important**: Place the executable in a dedicated folder before running to avoid cluttering your directories.

### 🛠 Building from source
```bash
git clone <this-repo>
cd vk-music-downloader-go
go mod tidy

# CLI only
go build -ldflags="-s -w" -o release/vk-music-downloader-cli-windows-amd64.exe cmd/cli/main.go

# GUI (requires Wails CLI: go install github.com/wailsapp/wails/v2/cmd/wails@latest)
cd cmd/gui && wails build -ldflags "-s -w"

# Or use the build scripts:
make release      # Linux/macOS
release.bat       # Windows
task release      # Taskfile
```

---

<a name="русский"></a>
## 🇷🇺 Русский

Быстрая, полностью независимая утилита для скачивания музыки и плейлистов из ВКонтакте, переписанная на Go. Доступна в двух вариантах: **CLI** (терминал) и **GUI** (десктопное приложение на Wails + React). Оригинальная версия на Node.js: [VACdotCS/vk-music-downloader](https://github.com/VACdotCS/vk-music-downloader).

### ✨ Особенности
* **Два интерфейса**: Классический CLI и современный GUI с карточками плейлистов, прогресс-барами и обложками.
* **Ноль зависимостей**: Нужен только один `.exe`. FFmpeg скачивается автоматически при отсутствии.
* **Высокая скорость**: Горутины + кастомный пул воркеров для параллельного скачивания.
* **Умный UI плейлистов**: Визуальный прогресс-бар, разделённый на успешно скачанные (синий) и заблокированные треки (красный). Клик по скачанному плейлисту открывает папку.
* **Дешифровка на лету**: Нативный парсинг HLS-потоков (`.m3u8`) и снятие шифрования AES-128.
* **Безопасная отмена**: `Ctrl+C` (CLI) или кнопка «Стоп» (GUI) мгновенно прерывает загрузки и удаляет временные файлы.
* **Лог авторских прав**: После скачивания плейлиста создаётся `Ошибки_авторских_прав.txt` со списком заблокированных треков.

### 🚀 Использование
1. Скачайте бинарник со страницы [Релизов](https://github.com/VACdotCS/vk-music-downloader-go/releases/latest):
   - `vk-music-downloader-cli-windows-amd64.exe` — терминальный интерфейс
   - `vk-music-downloader-gui-windows-amd64.exe` — графический интерфейс
2. Запустите файл. Приложение **полностью портативно** — все файлы конфигурации создаются **рядом с исполняемым файлом**.
3. Вставьте JSON с токеном VK Kate Mobile при запросе.

> **⚠️ Важно**: Положите файл в отдельную папку перед запуском, чтобы не засорять директории.

### 🛠 Сборка из исходников
```bash
git clone <this-repo>
cd vk-music-downloader-go
go mod tidy

# Только CLI
go build -ldflags="-s -w" -o release/vk-music-downloader-cli-windows-amd64.exe cmd/cli/main.go

# GUI (требует Wails CLI: go install github.com/wailsapp/wails/v2/cmd/wails@latest)
cd cmd/gui && wails build -ldflags "-s -w"

# Или через скрипты сборки:
make release      # Linux/macOS
release.bat       # Windows
task release      # Taskfile
```

---

## 🤖 Built with Antigravity

The GUI portion of this project was developed with the help of **[Antigravity](https://antigravity.dev)** — an AI-powered coding assistant by Google DeepMind. The agentic CLI (`agy`) was used for iterative UI development, bug fixing, and feature implementation across the Wails + React frontend and Go backend.
