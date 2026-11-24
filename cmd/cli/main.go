package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/mattn/go-colorable"
	"github.com/pterm/pterm"
	"vk-music-downloader-go/core/api"
	"vk-music-downloader-go/core/browser"
	"vk-music-downloader-go/core/cache"
	"vk-music-downloader-go/core/config"
	"vk-music-downloader-go/core/ffmpeg"
	"vk-music-downloader-go/core/scenarios"
)

var (
	myConfig  *config.Config
	vkService *api.VkApiService
)

func main() {
	// Подключаем парсер ANSI-цветов, который переводит их в нативные вызовы Windows API
	pterm.SetDefaultOutput(colorable.NewColorableStdout())

	cache.InitCache()
	authorInfo()

	cfg, err := config.LoadConfig()
	if err != nil {
		pterm.Error.Println("Ошибка загрузки конфига:", err)
		return
	}
	myConfig = cfg

	if myConfig.Token == nil || myConfig.Token.AccessToken == "" {
		if err := getAccessTokenData(); err != nil {
			return
		}
	} else {
		initVkService()
	}

	// 3. Проверка FFmpeg (скачивание при необходимости)
	err = ffmpeg.CheckAndDownload()
	if err != nil {
		pterm.Error.Println("Критическая ошибка: невозможно установить FFmpeg. Конвертация треков может не работать.")
		fmt.Println("Попробуйте установить его вручную.")
	}

	// 4. Папка сохранения
	if myConfig.SavePath == "" {
		if err := getSaveFolder(); err != nil {
			return
		}
	}

	for {
		if err := checkToken(); err != nil {
			return
		}
		if !mainMenu() {
			break
		}
	}
}

func authorInfo() {
	pterm.DefaultHeader.WithFullWidth().WithBackgroundStyle(pterm.NewStyle(pterm.BgCyan)).Println("CLI VK Audio Downloader (Golang Port)")
	pterm.Info.Println("Автор оригинала: Vladimir Taburkin")
	pterm.Info.Println("Гайд по получению токена и Гитхаб оригинала:")
	pterm.Info.Println("👉 https://github.com/VACdotCS/vk-music-downloader")
	fmt.Println()
}

func openBrowserURL(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		pterm.Warning.Printf("Не удалось автоматически открыть браузер. Перейдите по ссылке вручную: %s\n", url)
	}
}

func getAccessTokenData() error {
	choices := []string{
		"🌐 Войти через браузер (Автоматически, с сохранением сессии)",
		"🔑 Вставить токен вручную (Kate Mobile JSON)",
		"❓ Открыть гайд в браузере",
	}

	var choice string
	prompt := &survey.Select{
		Message: "Способ авторизации:",
		Options: choices,
	}
	if err := survey.AskOne(prompt, &choice); err != nil {
		return err
	}

	switch choice {
	case choices[0]:
		pterm.Info.Println("Запускаем браузер...")
		pterm.Info.Println("Пожалуйста, войдите в аккаунт ВКонтакте и разрешите доступ.")
		pterm.Info.Println("Окно закроется автоматически сразу после завершения.")
		tok, err := browser.LoginInteractive(context.Background())
		if err != nil {
			pterm.Error.Printf("Ошибка авторизации: %v\n", err)
			return err
		}
		myConfig.Token = tok
		_ = config.SaveConfig(myConfig)
		initVkService()
		pterm.Success.Println("Авторизация через браузер прошла успешно! Сессия сохранена.")
		return nil

	case choices[1]:
		inputPrompt := &survey.Input{
			Message: "Введите свой access token (JSON объект):",
		}
		var dataStr string
		if err := survey.AskOne(inputPrompt, &dataStr); err != nil {
			return err
		}
		var tokenJson struct {
			Data *config.Token `json:"data"`
		}
		if err := json.Unmarshal([]byte(dataStr), &tokenJson); err != nil || tokenJson.Data == nil {
			pterm.Error.Println("Неверный формат. Ожидался JSON с полем data.")
			return fmt.Errorf("invalid token format")
		}
		myConfig.Token = tokenJson.Data
		_ = config.SaveConfig(myConfig)
		initVkService()
		pterm.Success.Println("Токен успешно сохранен!")
		return nil

	case choices[2]:
		pterm.Info.Println("Открываем гайд в браузере...")
		openBrowserURL("https://github.com/VACdotCS/vk-music-downloader")
		return getAccessTokenData()
	}

	return nil
}

func initVkService() {
	if myConfig.Token != nil {
		vkService = api.NewVkApiService(myConfig.Token.AccessToken, myConfig.Token.UserID)
	}
}

func checkToken() error {
	if myConfig.Token == nil {
		return nil
	}

	if myConfig.Token.Expires <= 0 {
		return nil
	}

	currentTime := time.Now().Unix()
	if currentTime >= myConfig.Token.Expires {
		// Обновлять токен через headless можно ТОЛЬКО ЕСЛИ ЕСТЬ СОХРАНЁННАЯ СЕССИЯ БРАУЗЕРА!
		if browser.HasSavedSession() {
			pterm.Warning.Println("Токен истёк. Найдена сессия браузера, пробуем обновить в фоне...")
			token, err := browser.RefreshTokenHeadless(context.Background())
			if err == nil && token != nil {
				myConfig.Token = token
				_ = config.SaveConfig(myConfig)
				initVkService()
				pterm.Success.Println("Токен успешно продлен в фоновом режиме!")
				return nil
			}
			pterm.Warning.Println("Фоновое обновление не удалось (сессия браузера протухла). Требуется повторный вход.")
		} else {
			pterm.Warning.Println("Токен истёк. Сохранённой сессии браузера нет, требуется ввод токена:")
		}
		return getAccessTokenData()
	}
	return nil
}

func getSaveFolder() error {
	prompt := &survey.Input{
		Message: "Введите путь для сохранения файлов:",
	}
	var savePath string
	if err := survey.AskOne(prompt, &savePath); err != nil {
		return err
	}

	absPath, _ := filepath.Abs(savePath)
	myConfig.SavePath = absPath
	config.SaveConfig(myConfig)
	pterm.Success.Printf("Путь для сохранения установлен: %s\n", absPath)
	return nil
}

func mainMenu() bool {
	choices := []string{
		"🎶 Скачать все треки из основного плейлиста",
		"🎶 Скачать все плейлисты",
		"🔗 Скачать трек по ссылке",
		"🔗 Скачать плейлист по ссылке",
		"⚙️ Показать путь для скачивания",
		"⚙️ Изменить путь для скачивания",
		"⚙️ Вывести путь к приложению",
		"⚙️ Показать путь установки FFmpeg",
		"⚙️ Очистить весь кэш (удалит всё, кроме пути сохранения треков)",
		"⚙️ Вывести заблокированные треки",
		"⚙️ Получить бесконечный токен",
		"🌐 Авторизоваться / Обновить токен через браузер",
		"🚪👋 Выход",
	}

	var choice string
	prompt := &survey.Select{
		Message: "Выберите действие:",
		Options: choices,
	}

	if err := survey.AskOne(prompt, &choice); err != nil {
		return false
	}

	switch choice {
	case choices[0]:
		scenarios.GetAllAudioScenario(myConfig.SavePath, vkService)
	case choices[1]:
		scenarios.GetAllPlaylistsTracksScenario(myConfig.SavePath, vkService)
	case choices[2]:
		scenarios.GetTrackByLinkScenario(myConfig.SavePath, vkService)
	case choices[3]:
		scenarios.GetPlaylistTracksByLinkScenario(myConfig.SavePath, vkService)
	case choices[4]:
		fmt.Printf("Путь: %s\n", myConfig.SavePath)
	case choices[5]:
		getSaveFolder()
	case choices[6]:
		dir, _ := os.Getwd()
		fmt.Printf("Путь: %s\n", dir)
	case choices[7]:
		ffmpegPath := ffmpeg.GetPath()
		pterm.Info.Printf("FFmpeg используется по пути: %s\n", ffmpegPath)
	case choices[8]:
		cache.ClearCache()
	case choices[9]:
		cache.OutputBlockedTracks()
	case choices[10]:
		if err := scenarios.GetUnlimitedTokenScenario(myConfig); err == nil {
			initVkService()
		}
	case choices[11]:
		pterm.Info.Println("Запускаем браузер для авторизации...")
		tok, err := browser.LoginInteractive(context.Background())
		if err != nil {
			pterm.Error.Printf("Ошибка авторизации: %v\n", err)
		} else {
			myConfig.Token = tok
			_ = config.SaveConfig(myConfig)
			initVkService()
			pterm.Success.Println("Сессия успешно сохранена!")
		}
	case choices[12]:
		cache.ClearTempFiles(myConfig.SavePath)
		pterm.Info.Println("До свидания!")
		return false
	default:
		pterm.Warning.Println("Неизвестный пункт меню.")
	}

	fmt.Println()
	return true
}
