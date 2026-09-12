package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"vk-music-downloader-go/internal/api"
)

func InitCache() {
	if _, err := os.Stat("errors.json"); os.IsNotExist(err) {
		os.WriteFile("errors.json", []byte("[]"), 0644)
	}
}

func CatchAudioStreamError(err error, audioMeta api.Audio, fileName string) error {
	audioBlockedMsg := fmt.Sprintf("Трек заблокирован в Вашем регионе: %s", fileName)

	data, readErr := os.ReadFile("errors.json")
	var errors []api.Audio
	if readErr == nil && len(data) > 0 {
		_ = json.Unmarshal(data, &errors)
	}

	errors = append(errors, audioMeta)
	updatedData, _ := json.MarshalIndent(errors, "", "  ")
	os.WriteFile("errors.json", updatedData, 0644)

	if audioMeta.ContentRestricted != 0 {
		return fmt.Errorf(audioBlockedMsg)
	}

	return err
}

func OutputBlockedTracks() {
	data, err := os.ReadFile("errors.json")
	if err != nil {
		fmt.Println("Нет заблокированных треков или файл кэша не найден.")
		return
	}

	var errors []api.Audio
	if err := json.Unmarshal(data, &errors); err != nil {
		fmt.Println("Ошибка чтения errors.json")
		return
	}

	if len(errors) == 0 {
		fmt.Println("Список заблокированных треков пуст.")
		return
	}

	fmt.Println("Заблокированные или ошибочные треки:")
	for _, t := range errors {
		fmt.Printf("- %s - %s\n", t.Artist, t.Title)
	}
}

func ClearCache() {
	_ = os.Remove("errors.json")
	InitCache()
	fmt.Println("Кэш успешно очищен!")
}

func ClearTempFiles(savePath string) {
	files, err := os.ReadDir(savePath)
	if err != nil {
		return
	}

	for _, file := range files {
		if !file.IsDir() && (strings.HasPrefix(file.Name(), "temp-") || strings.HasSuffix(file.Name(), ".ts")) {
			_ = os.Remove(filepath.Join(savePath, file.Name()))
		}
	}
}
