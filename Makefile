.PHONY: build build-gui release clean run dev-gui tidy

# Имена выходных файлов
CLI_WIN   = vk-music-downloader-cli-windows-amd64.exe
CLI_LINUX = vk-music-downloader-cli-linux-amd64
GUI_WIN   = vk-music-downloader-gui-windows-amd64.exe
BUILD_DIR = release

# Флаги компиляции (-s -w убирают отладочную информацию, уменьшая размер файла)
LDFLAGS = -s -w

# Сборка CLI-бинарника под текущую ОС
build:
	go build -ldflags="$(LDFLAGS)" -o vk-music-downloader.exe cmd/cli/main.go

# Сборка GUI (Wails) под Windows
build-gui:
	cd cmd/gui && wails build -ldflags "$(LDFLAGS)" -o ../../$(BUILD_DIR)/$(GUI_WIN)

# Полная сборка CLI + GUI под Windows и Linux
release: clean
	mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(CLI_WIN) cmd/cli/main.go
	GOOS=linux   GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(CLI_LINUX) cmd/cli/main.go
	$(MAKE) build-gui
	@echo "✅ Все бинарники собраны в папке $(BUILD_DIR)/"

# Очистка скомпилированных файлов и кэша
clean:
	rm -rf $(BUILD_DIR)
	rm -f vk-music-downloader.exe
	rm -f config.json errors.json temp-*.ts *-music-data.json

# Быстрый запуск CLI без явной компиляции
run:
	go run cmd/cli/main.go

# Запуск GUI в режиме разработки
dev-gui:
	cd cmd/gui && wails dev

# Обновление и подтяжка зависимостей
tidy:
	go mod tidy
