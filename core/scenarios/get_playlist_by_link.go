package scenarios

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/pterm/pterm"
	"vk-music-downloader-go/core/api"
	"vk-music-downloader-go/core/downloader"
	"vk-music-downloader-go/core/utils"
)

func GetPlaylistTracksByLinkScenario(savePath string, vkService *api.VkApiService) error {
	var link string
	prompt := &survey.Input{Message: "Введите ссылку на плейлист:"}
	if err := survey.AskOne(prompt, &link); err != nil {
		return err
	}

	spinner, _ := pterm.DefaultSpinner.Start("Скачиваю список треков плейлиста")
	tracks, err := vkService.GetTracksOfPlaylistByLink(link)
	if err != nil {
		spinner.Fail("Ошибка получения плейлиста: " + err.Error())
		return err
	}
	spinner.Success("Треки найдены.\n")

	data, _ := json.MarshalIndent(tracks, "", "  ")
	os.WriteFile("playlist-link-music-data.json", data, 0644)

	var toDownload []api.Audio
	namingIndex := 1

	files, _ := os.ReadDir(savePath)
	for _, audio := range tracks {
		expectedName := utils.GetNormalFileName(audio.Artist, audio.Title)
		found := false
		for _, f := range files {
			if strings.Contains(f.Name(), expectedName) {
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
		pterm.Info.Println("Все треки из плейлиста уже скачаны!")
		return nil
	}

	fmt.Printf("К скачиванию: %d треков\n", len(toDownload))
	downloader.DownloadBatchOfTracks(nil, toDownload, savePath, namingIndex)
	return nil
}
