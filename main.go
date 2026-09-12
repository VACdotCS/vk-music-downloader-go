package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/pterm/pterm"
	"vk-music-downloader-go/internal/api"
	"vk-music-downloader-go/internal/cache"
	"vk-music-downloader-go/internal/config"
	"vk-music-downloader-go/internal/scenarios"
)

var myConfig *config.Config
var vkService *api.VkApiService

func main() {
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
		"🎶 Скачать все плейлисты (В разработке)",
		"🔗 Скачать трек по ссылке (В разработке)",
		"🔗 Скачать плейлист по ссылке (В разработке)",
		"⚙️ Показать путь для скачивания",
		"⚙️ Изменить путь для скачивания",
		"⚙️ Вывести путь к приложению",
		"⚙️ Очистить весь кэш (удалит всё, кроме пути сохранения треков)",
		"⚙️ Вывести заблокированные треки",
		"⚙️ Получить бесконечный токен (В разработке)",
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
	case choices[4]:
		fmt.Printf("Путь: %s\n", myConfig.SavePath)
	case choices[5]:
		getSaveFolder()
	case choices[6]:
		dir, _ := os.Getwd()
		fmt.Printf("Путь: %s\n", dir)
	case choices[7]:
		cache.ClearCache()
	case choices[8]:
		cache.OutputBlockedTracks()
	case choices[10]:
		cache.ClearTempFiles(myConfig.SavePath)
		pterm.Info.Println("До свидания!")
		return false
	default:
		pterm.Warning.Println("Эта функция пока не перенесена на Go-версию.")
	}

	fmt.Println()
	return true
}
