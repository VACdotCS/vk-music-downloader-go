package scenarios

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pterm/pterm"
	"vk-music-downloader-go/core/api"
	"vk-music-downloader-go/core/downloader"
	"vk-music-downloader-go/core/utils"
)

func GetAllAudioScenario(savePath string, vkService *api.VkApiService) error {
	spinner, _ := pterm.DefaultSpinner.Start("Скачиваю список ваших треков")

	audioList, err := vkService.GetAudiosList()
	if err != nil {
		spinner.Fail("Ошибка получения списка треков")
		return err
	}

	// Reverse list to match the behavior of the original Node.js app
	for i, j := 0, len(audioList)-1; i < j; i, j = i+1, j-1 {
		audioList[i], audioList[j] = audioList[j], audioList[i]
	}

	spinner.Success("Список треков получен. Запускаю скачивание.\n")

	// Save all-music-data.json
	data, _ := json.MarshalIndent(audioList, "", "  ")
	os.WriteFile("all-music-data.json", data, 0644)

	// Read already downloaded files
	downloadedFilesMeta := make(map[string]bool)
	files, err := os.ReadDir(savePath)
	if err == nil {
		for _, f := range files {
			if !f.IsDir() {
				downloadedFilesMeta[f.Name()] = true
			}
		}
	} else {
		// Create dir if not exists
		os.MkdirAll(savePath, 0755)
	}

	var toDownload []api.Audio
	namingIndex := 1

	for _, audio := range audioList {
		expectedName := utils.GetNormalFileName(audio.Artist, audio.Title)

		// Check if file already downloaded
		found := false
		for dFile := range downloadedFilesMeta {
			if strings.Contains(dFile, expectedName) {
				found = true
				break
			}
		}

		if found {
			namingIndex++
		} else {
			toDownload = append(toDownload, audio)
		}
	}

	if len(toDownload) == 0 {
		pterm.Info.Println("Все треки уже скачаны!")
		return nil
	}

	fmt.Printf("К скачиванию: %d треков\n", len(toDownload))
	downloader.DownloadBatchOfTracks(nil, toDownload, savePath, namingIndex)

	return nil
}
