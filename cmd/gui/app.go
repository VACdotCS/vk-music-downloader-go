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
	"vk-music-downloader-go/core/config"
	"vk-music-downloader-go/core/downloader"
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
	cfg, _ := config.LoadConfig()
	a.config = cfg
}

// CancelDownload прерывает текущую загрузку
func (a *App) CancelDownload() {
	if a.downloadCancel != nil {
		a.downloadCancel()
	}
}

// HasValidToken проверяет, есть ли токен
func (a *App) HasValidToken() bool {
	return a.config != nil && a.config.Token != nil
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

// OpenAuthPage открывает страницу получения токена в браузере
func (a *App) OpenAuthPage() {
	url := "https://oauth.vk.com/authorize?client_id=6463690&scope=1073737727&redirect_uri=https://oauth.vk.com/blank.html&display=page&response_type=token&revoke=1"
	runtime.BrowserOpenURL(a.ctx, url)
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

// DownloadTrack скачивает один трек по ссылке
func (a *App) DownloadTrack(link string) error {
	a.initProgressCallback()
	vkService := api.NewVkApiService(a.config.Token.AccessToken, a.config.Token.UserID)
	audioData, err := vkService.GetAudioByLink(link)
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
	vkService := api.NewVkApiService(a.config.Token.AccessToken, a.config.Token.UserID)
	
	tracks, err := vkService.GetTracksOfPlaylistByLink(link)
	if err != nil {
		return err
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	a.downloadCancel = cancel
	defer cancel()
	
	downloader.DownloadBatchOfTracks(ctx, tracks, a.config.SavePath, 1)
	return nil
}

// GetUserPlaylists возвращает список плейлистов пользователя
func (a *App) GetUserPlaylists() ([]api.Playlist, error) {
	vkService := api.NewVkApiService(a.config.Token.AccessToken, a.config.Token.UserID)
	return vkService.GetPlaylists()
}

// DownloadUserPlaylist скачивает конкретный плейлист пользователя по ID в свою папку
func (a *App) DownloadUserPlaylist(playlistID int, title string) error {
	a.initProgressCallback()
	vkService := api.NewVkApiService(a.config.Token.AccessToken, a.config.Token.UserID)
	
	tracks, err := vkService.GetTracksOfUserPlaylist(playlistID)
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

// DownloadAllAudio запускает скачивание всех сохраненных треков (не плейлистов)
func (a *App) DownloadAllAudio() error {
	a.initProgressCallback()
	vkService := api.NewVkApiService(a.config.Token.AccessToken, a.config.Token.UserID)
	
	ctx, cancel := context.WithCancel(context.Background())
	a.downloadCancel = cancel
	defer cancel()
	
	// Получаем список всех треков пользователя
	
	audioList, err := vkService.GetAudiosList()
	if err != nil {
		return err
	}

	for i, j := 0, len(audioList)-1; i < j; i, j = i+1, j-1 {
		audioList[i], audioList[j] = audioList[j], audioList[i]
	}
	
	downloader.DownloadBatchOfTracks(ctx, audioList, a.config.SavePath, 1)
	return nil
}
