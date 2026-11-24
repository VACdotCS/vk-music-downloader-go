package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	stdruntime "runtime"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"vk-music-downloader-go/core/api"
	"vk-music-downloader-go/core/browser"
	"vk-music-downloader-go/core/config"
	"vk-music-downloader-go/core/downloader"
	"vk-music-downloader-go/core/logger"
	"vk-music-downloader-go/core/utils"
)

// App struct
type App struct {
	ctx            context.Context
	config         *config.Config
	downloadCancel context.CancelFunc
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	logger.SetLogHook(func(entry logger.LogEntry) {
		runtime.EventsEmit(a.ctx, "app-log", entry)
	})
	logger.Info("GUI приложение запущено")
	cfg, _ := config.LoadConfig()
	a.config = cfg
}

// shutdown is called when the app closes
func (a *App) shutdown(ctx context.Context) {
	logger.Info("GUI приложение закрывается, очищаем дочерние процессы браузера...")
	browser.CleanupActiveBrowsers()
}

// CancelDownload прерывает текущую загрузку
func (a *App) CancelDownload() {
	if a.downloadCancel != nil {
		a.downloadCancel()
	}
}

// HasValidToken проверяет, есть ли токен
func (a *App) HasValidToken() bool {
	return a.config != nil && a.config.Token != nil && a.config.Token.AccessToken != ""
}

// SaveToken сохраняет токен из JSON строки (Kate Mobile)
func (a *App) SaveToken(dataStr string) error {
	var tokenJson struct {
		Data *config.Token `json:"data"`
	}

	if err := json.Unmarshal([]byte(dataStr), &tokenJson); err != nil || tokenJson.Data == nil {
		return fmt.Errorf("неверный формат токена")
	}

	a.config.Token = tokenJson.Data
	config.SaveConfig(a.config)
	return nil
}

// ParseAndSaveUrlToken парсит ссылку из адресной строки (OAuth)
func (a *App) ParseAndSaveUrlToken(tokenLink string) error {
	parts := strings.Split(tokenLink, "#")
	if len(parts) < 2 {
		return fmt.Errorf("неверный формат ссылки")
	}

	res := make(map[string]string)
	params := strings.Split(parts[1], "&")
	for _, p := range params {
		kv := strings.Split(p, "=")
		if len(kv) == 2 {
			res[kv[0]] = kv[1]
		}
	}

	accessToken := res["access_token"]
	if accessToken == "" {
		return fmt.Errorf("токен не найден в ссылке")
	}

	// userID не всегда нужен, но попробуем спарсить
	var userID int
	if uid, ok := res["user_id"]; ok {
		fmt.Sscanf(uid, "%d", &userID)
	}

	a.config.Token = &config.Token{
		AccessToken: accessToken,
		UserID:      userID,
	}
	config.SaveConfig(a.config)
	return nil
}

// OpenAuthPage открывает vk.ru в системном браузере
func (a *App) OpenAuthPage() {
	runtime.BrowserOpenURL(a.ctx, "https://vk.ru")
}

// GetAvailableBrowsers возвращает список установленных в системе браузеров
func (a *App) GetAvailableBrowsers() []browser.BrowserInfo {
	return browser.DetectAllBrowsers()
}

// LoginViaBrowser открывает системный браузер (Chrome/Edge/Brave/Firefox) с сохранением профиля,
// перехватывает токен после авторизации и закрывает браузер.
func (a *App) LoginViaBrowser(browserPath string) error {
	token, err := browser.LoginInteractive(context.Background(), browserPath)
	if err != nil {
		return err
	}
	if a.config == nil {
		a.config = &config.Config{}
	}
	a.config.Token = token
	return config.SaveConfig(a.config)
}

// RefreshSessionHeadless запускает браузер в фоновом (headless) режиме и обновляет токен без GUI.
// Работает ТОЛЬКО если есть сохранённая рабочая сессия в профиле браузера.
func (a *App) RefreshSessionHeadless() error {
	if !browser.HasSavedSession() {
		return fmt.Errorf("нет сохранённой сессии браузера: войдите через браузер")
	}
	token, err := browser.RefreshTokenHeadless(context.Background())
	if err != nil {
		return err
	}
	if a.config == nil {
		a.config = &config.Config{}
	}
	a.config.Token = token
	return config.SaveConfig(a.config)
}

// HasBrowserSession проверяет, есть ли сохранённая сессия браузера
func (a *App) HasBrowserSession() bool {
	return browser.HasSavedSession()
}

// GetLogs возвращает все накопленные логи
func (a *App) GetLogs() []logger.LogEntry {
	return logger.GetLogs()
}

// ClearLogs очищает журнал логов
func (a *App) ClearLogs() {
	logger.ClearLogs()
}

// ClearToken удаляет текущий токен (Logout)
func (a *App) ClearToken() {
	if a.config != nil {
		a.config.Token = nil
		config.SaveConfig(a.config)
	}
}

// SelectDirectory открывает диалог выбора папки
func (a *App) SelectDirectory() string {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Выберите папку для сохранения музыки",
	})
	if err != nil || dir == "" {
		return ""
	}
	
	a.config.SavePath = dir
	config.SaveConfig(a.config)
	return dir
}

// GetSavePath возвращает текущий путь сохранения
func (a *App) GetSavePath() string {
	if a.config == nil {
		return ""
	}
	return a.config.SavePath
}

// initProgressCallback привязывает вызов Wails к загрузчику
func (a *App) initProgressCallback() {
	downloader.GUIProgressCallback = func(index int, title string, percentage float64, status string) {
		runtime.EventsEmit(a.ctx, "download-progress", map[string]interface{}{
			"index":      index,
			"title":      title,
			"percentage": percentage,
			"status":     status,
		})
	}
}

func (a *App) ensureValidToken() error {
	if a.config != nil && a.config.Token != nil && a.config.Token.AccessToken != "" {
		return nil
	}
	if browser.HasSavedSession() {
		logger.Info("Токен отсутствует в конфиге. Запускаем фоновое обновление через сессию браузера...")
		token, err := browser.RefreshTokenHeadless(context.Background())
		if err == nil && token != nil {
			if a.config == nil {
				a.config = &config.Config{}
			}
			a.config.Token = token
			_ = config.SaveConfig(a.config)
			return nil
		}
	}
	return fmt.Errorf("нет токена и нет сохранённой сессии браузера")
}

func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "authorization failed") ||
		strings.Contains(msg, "token has expired") ||
		strings.Contains(msg, "invalid token") ||
		strings.Contains(msg, "bad status: 401") ||
		strings.Contains(msg, "error_code: 5") ||
		strings.Contains(msg, "code 5") ||
		strings.Contains(msg, "error-token")
}

func (a *App) executeWithTokenRetry(fn func(vkService *api.VkApiService) error) error {
	if err := a.ensureValidToken(); err != nil {
		return err
	}
	vkService := api.NewVkApiService(a.config.Token.AccessToken, a.config.Token.UserID)
	err := fn(vkService)
	if err != nil && isAuthError(err) && browser.HasSavedSession() {
		logger.Warn("Токен устарел или отклонён (%v). Автоматически обновляем в фоне через сессию браузера...", err)
		token, refreshErr := browser.RefreshTokenHeadless(context.Background())
		if refreshErr == nil && token != nil {
			a.config.Token = token
			_ = config.SaveConfig(a.config)
			vkService = api.NewVkApiService(token.AccessToken, token.UserID)
			logger.Info("Токен успешно обновлён в фоне! Повторяем запрос...")
			return fn(vkService)
		}
		logger.Error("Не удалось обновить токен в фоне: %v", refreshErr)
	}
	return err
}

// DownloadTrack скачивает один трек по ссылке
func (a *App) DownloadTrack(link string) error {
	a.initProgressCallback()
	var audioData *api.Audio
	err := a.executeWithTokenRetry(func(vkService *api.VkApiService) error {
		var e error
		audioData, e = vkService.GetAudioByLink(link)
		return e
	})
	if err != nil {
		return err
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	a.downloadCancel = cancel
	defer cancel()
	
	downloader.DownloadBatchOfTracks(ctx, []api.Audio{*audioData}, a.config.SavePath, 1)
	return nil
}

// DownloadPlaylist скачивает плейлист по ссылке
func (a *App) DownloadPlaylist(link string) error {
	a.initProgressCallback()
	var tracks []api.Audio
	err := a.executeWithTokenRetry(func(vkService *api.VkApiService) error {
		var e error
		tracks, e = vkService.GetTracksOfPlaylistByLink(link)
		return e
	})
	if err != nil {
		return err
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	a.downloadCancel = cancel
	defer cancel()
	
	downloader.DownloadBatchOfTracks(ctx, tracks, a.config.SavePath, 1)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return nil
}

// GetUserPlaylists возвращает список плейлистов пользователя
func (a *App) GetUserPlaylists() ([]api.Playlist, error) {
	var pls []api.Playlist
	err := a.executeWithTokenRetry(func(vkService *api.VkApiService) error {
		var e error
		pls, e = vkService.GetPlaylists()
		return e
	})
	return pls, err
}

// DownloadUserPlaylist скачивает конкретный плейлист пользователя по ID в свою папку
func (a *App) DownloadUserPlaylist(playlistID int, title string) error {
	a.initProgressCallback()
	var tracks []api.Audio
	err := a.executeWithTokenRetry(func(vkService *api.VkApiService) error {
		var e error
		tracks, e = vkService.GetTracksOfUserPlaylist(playlistID)
		return e
	})
	if err != nil {
		return err
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	a.downloadCancel = cancel
	defer cancel()

	// Очищаем имя папки
	safeTitle := title
	invalidChars := []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"}
	for _, char := range invalidChars {
		safeTitle = strings.ReplaceAll(safeTitle, char, "")
	}
	
	// Путь сохранения: базовая папка + имя плейлиста
	savePath := filepath.Join(a.config.SavePath, safeTitle)
	os.MkdirAll(savePath, 0755)

	// Сохраняем метаданные плейлиста в json
	jsonData, _ := json.MarshalIndent(tracks, "", "  ")
	os.WriteFile(filepath.Join(savePath, fmt.Sprintf("%s-music-data.json", safeTitle)), jsonData, 0644)
	
	downloader.DownloadBatchOfTracks(ctx, tracks, savePath, 1)
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Создаем лог с треками, которые не удалось скачать (авторские права и т.д.)
	var failedTracks []string
	for i, t := range tracks {
		fileName := fmt.Sprintf("%d. %s", i+1, utils.GetNormalFileName(t.Artist, t.Title))
		if _, err := os.Stat(filepath.Join(savePath, fileName)); os.IsNotExist(err) {
			failedTracks = append(failedTracks, fmt.Sprintf("%s - %s", t.Artist, t.Title))
		}
	}

	if len(failedTracks) > 0 {
		errorLogContent := "Эти треки не были скачаны (вероятно, они заблокированы правообладателем):\n\n"
		for _, f := range failedTracks {
			errorLogContent += "- " + f + "\n"
		}
		os.WriteFile(filepath.Join(savePath, "Ошибки_авторских_прав.txt"), []byte(errorLogContent), 0644)
	}

	return nil
}

// OpenPlaylistFolder открывает папку со скачанным плейлистом
func (a *App) OpenPlaylistFolder(title string) error {
	safeTitle := title
	invalidChars := []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"}
	for _, char := range invalidChars {
		safeTitle = strings.ReplaceAll(safeTitle, char, "")
	}
	savePath := filepath.Join(a.config.SavePath, safeTitle)

	var cmd *exec.Cmd
	switch stdruntime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", savePath)
	case "darwin":
		cmd = exec.Command("open", savePath)
	default: // linux
		cmd = exec.Command("xdg-open", savePath)
	}
	return cmd.Start()
}

type LocalProgress struct {
	Downloaded int `json:"downloaded"`
	Errors     int `json:"errors"`
}

// CheckPlaylistLocalProgress возвращает количество скачанных файлов и ошибок
func (a *App) CheckPlaylistLocalProgress(title string) LocalProgress {
	safeTitle := title
	invalidChars := []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"}
	for _, char := range invalidChars {
		safeTitle = strings.ReplaceAll(safeTitle, char, "")
	}
	dir := filepath.Join(a.config.SavePath, safeTitle)
	files, err := os.ReadDir(dir)
	if err != nil {
		return LocalProgress{0, 0}
	}
	
	count := 0
	for _, f := range files {
		if strings.HasSuffix(strings.ToLower(f.Name()), ".mp3") {
			count++
		}
	}

	errorsCount := 0
	if data, err := os.ReadFile(filepath.Join(dir, "Ошибки_авторских_прав.txt")); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "- ") {
				errorsCount++
			}
		}
	}
	
	return LocalProgress{Downloaded: count, Errors: errorsCount}
}

// DownloadAllAudio запускает скачивание всех сохраненных треков (не плейлистов)
func (a *App) DownloadAllAudio() error {
	a.initProgressCallback()
	var audioList []api.Audio
	err := a.executeWithTokenRetry(func(vkService *api.VkApiService) error {
		var e error
		audioList, e = vkService.GetAudiosList()
		return e
	})
	if err != nil {
		return err
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	a.downloadCancel = cancel
	defer cancel()

	for i, j := 0, len(audioList)-1; i < j; i, j = i+1, j-1 {
		audioList[i], audioList[j] = audioList[j], audioList[i]
	}
	
	downloader.DownloadBatchOfTracks(ctx, audioList, a.config.SavePath, 1)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return nil
}
