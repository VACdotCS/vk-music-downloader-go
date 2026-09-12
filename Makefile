.PHONY: build release clean run tidy

# Имена выходных файлов
APP_NAME_WIN = vk-music-downloader-windows-amd64.exe
APP_NAME_LINUX = vk-music-downloader-linux-amd64
BUILD_DIR = release

# Флаги компиляции (-s -w убирают отладочную информацию, уменьшая размер файла)
LDFLAGS = -s -w

# Быстрый билд под вашу текущую систему
build:
	go build -ldflags="$(LDFLAGS)" -o vk-music-downloader.exe main.go

# Билд релизных бинарников под Windows и Linux
release: clean
	mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME_WIN) main.go
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME_LINUX) main.go
	@echo "Релизные бинарники успешно собраны в папке $(BUILD_DIR)/"

# Очистка скомпилированных файлов и кэша
clean:
	rm -rf $(BUILD_DIR)
	rm -f vk-music-downloader.exe
	rm -f config.json errors.json temp-*.ts *-music-data.json

# Быстрый запуск без явной компиляции
run:
	go run main.go

# Обновление и подтяжка зависимостей
tidy:
	go mod tidy
