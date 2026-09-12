# VK Music Downloader (Golang)

🇬🇧 [English](#english) | 🇷🇺 [Русский](#русский)

---

<a name="english"></a>
## 🇬🇧 English

A fast, fully standalone CLI utility for downloading music and playlists from VK (Vkontakte), completely rewritten in Go. Originally based on the [Node.js version by VACdotCS](https://github.com/VACdotCS/vk-music-downloader).

### ✨ Features
* **Zero Dependencies**: You only need a single `.exe` file. If your system lacks FFmpeg (required for audio conversion), the downloader will automatically download and install a portable version of it next to the executable.
* **Blazing Fast**: Uses Go goroutines and a custom worker pool for concurrent downloading.
* **Smart UI**: Interactive terminal menus and smooth, flicker-free progress bars (Event Loop style rendering).
* **On-the-fly Decryption**: Native parsing of HLS (`.m3u8`) streams and AES-128 decryption.
* **Graceful Interruption**: Pressing `Ctrl+C` instantly aborts the current download tasks, cleans up temporary files, and returns you safely to the main menu without killing the app.

### 🚀 Usage
1. Download the latest compiled executable for your OS (Windows `.exe` or Linux binary) from the [Releases page](https://github.com/VACdotCS/vk-music-downloader-go/releases/latest).
2. Run the executable in your terminal:
   ```cmd
   .\vk-music-downloader.exe
   ```
3. Follow the interactive menu:
   * Paste your VK `access_token` (Follow [this guide](https://github.com/VACdotCS/vk-music-downloader) to learn how to get one).
   * Choose a folder to save your tracks.
   * Select a download scenario (All tracks, specific playlist, or single track via link).

### 🛠 Building from source
If you want to compile the project yourself:
```bash
git clone <this-repo>
cd vk-music-downloader-go
go mod tidy
go build -o vk-music-downloader.exe main.go
```

---

<a name="русский"></a>
## 🇷🇺 Русский

Быстрая, полностью независимая CLI-утилита для скачивания музыки и плейлистов из ВКонтакте, полностью переписанная на Go. Оригинальная версия на Node.js: [VACdotCS/vk-music-downloader](https://github.com/VACdotCS/vk-music-downloader).

### ✨ Особенности
* **Ноль зависимостей**: Вам нужен только один файл `.exe`. Если в вашей системе не установлен FFmpeg (нужен для конвертации аудио), программа автоматически скачает его портативную версию и положит рядом с собой.
* **Высокая скорость**: Использование горутин и кастомного пула воркеров для параллельного скачивания.
* **Умный интерфейс**: Интерактивные меню в терминале и плавные прогресс-бары без мерцания (отрисовка в стиле Event Loop).
* **Дешифровка на лету**: Нативный парсинг потоков HLS (`.m3u8`) и снятие шифрования AES-128.
* **Безопасная отмена**: Нажатие `Ctrl+C` мгновенно прерывает текущие загрузки, подчищает за собой временные файлы и возвращает вас в главное меню, не убивая саму программу.

### 🚀 Использование
1. Скачайте свежий исполняемый файл для вашей ОС (Windows `.exe` или бинарник для Linux) со страницы [Релизов (Releases)](https://github.com/VACdotCS/vk-music-downloader-go/releases/latest).
2. Запустите его в терминале:
   ```cmd
   .\vk-music-downloader.exe
   ```
3. Следуйте интерактивному меню:
   * Вставьте ваш `access_token` от ВК (Гайд по получению токена есть в [оригинальном репозитории](https://github.com/VACdotCS/vk-music-downloader)).
   * Выберите папку для сохранения музыки.
   * Выберите сценарий (Скачать всё, конкретный плейлист или один трек по ссылке).

### 🛠 Сборка из исходников
Если хотите скомпилировать проект самостоятельно:
```bash
git clone <this-repo>
cd vk-music-downloader-go
go mod tidy
go build -o vk-music-downloader.exe main.go
```
