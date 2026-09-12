package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"vk-music-downloader-go/core/config"
)

// App struct
type App struct {
	ctx    context.Context
	config *config.Config
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

// HasValidToken проверяет, есть ли токен
func (a *App) HasValidToken() bool {
	return a.config != nil && a.config.Token != nil
}

// SaveToken сохраняет токен из строки
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
