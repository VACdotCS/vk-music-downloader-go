package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/mattn/go-colorable"
	"github.com/pterm/pterm"
	"vk-music-downloader-go/internal/api"
	"vk-music-downloader-go/internal/cache"
	"vk-music-downloader-go/internal/config"
	"vk-music-downloader-go/internal/ffmpeg"
	"vk-music-downloader-go/internal/scenarios"
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
	pterm.Info.Println("Гитхаб оригинала: https://github.com/VACdotCS")
	fmt.Println()
}

func getAccessTokenData() error {
	prompt := &survey.Input{
		Message: "Введите свой access token (JSON объект, скопированный по гайду):",
	}
	var dataStr string
	if err := survey.AskOne(prompt, &dataStr); err != nil {
		return err
	}

	var tokenJson struct {
		Data *config.Token `json:"data"`
	}

	if err := json.Unmarshal([]byte(dataStr), &tokenJson); err != nil || tokenJson.Data == nil {
		// fallback если пользователь ввел просто токен, хотя в оригинале ждется JSON
		pterm.Error.Println("Неверный формат. Ожидался JSON с полем data.")
		return fmt.Errorf("invalid token format")
	}

	myConfig.Token = tokenJson.Data
	config.SaveConfig(myConfig)
	initVkService()

	pterm.Success.Println("Токен успешно сохранен!")
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
	
	// В JS, если `expires` отсутствует, он равен undefined.
	// Сравнение (Date.now() >= undefined) даёт false. 
	// В Go отсутствующее поле парсится как 0. Из-за этого токен "протухал" мгновенно.
	if myConfig.Token.Expires <= 0 {
		return nil
	}

	currentTime := time.Now().Unix()
	if currentTime >= myConfig.Token.Expires {
		pterm.Warning.Println("Токен истёк, нужен новый.")
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
		cache.ClearTempFiles(myConfig.SavePath)
		pterm.Info.Println("До свидания!")
		return false
	default:
		pterm.Warning.Println("Неизвестный пункт меню.")
	}

	fmt.Println()
	return true
}
