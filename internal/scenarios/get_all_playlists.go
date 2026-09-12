package scenarios

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pterm/pterm"
	"vk-music-downloader-go/internal/api"
	"vk-music-downloader-go/internal/downloader"
	"vk-music-downloader-go/internal/utils"
)

func GetAllPlaylistsTracksScenario(savePath string, vkService *api.VkApiService) error {
	spinner, _ := pterm.DefaultSpinner.Start("Скачиваю список ваших плейлистов")

	playlists, err := vkService.GetPlaylists()
	if err != nil {
		spinner.Fail("Ошибка получения плейлистов")
		return err
	}
	spinner.Success("Плейлисты найдены")

	sp2, _ := pterm.DefaultSpinner.Start("Узнаю, что в них за треки")
	type playlistMetadata struct {
		Title  string
		Tracks []api.Audio
	}
	var playlistsMetadata []playlistMetadata

	for _, p := range playlists {
		tracks, err := vkService.GetTracksOfUserPlaylist(p.ID)
		if err != nil {
			sp2.Fail("Ошибка получения треков из плейлиста " + p.Title)
			return err
		}
		playlistsMetadata = append(playlistsMetadata, playlistMetadata{
			Title:  p.Title,
			Tracks: tracks,
		})
	}
	sp2.Success("Узнал списки треков\n")

	for _, p := range playlistsMetadata {
		title := utils.SanitizeFolderName(p.Title)
		pDownloadSpinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Скачиваю плейлист: %s", p.Title))
		_savePath := filepath.Join(savePath, title)

		os.MkdirAll(_savePath, 0755)

		tracksInDir, _ := os.ReadDir(_savePath)
		counter := 0

		for _, trackInDir := range tracksInDir {
			for _, track := range p.Tracks {
				trackFileName := utils.GetNormalFileName(track.Artist, track.Title)
				if strings.Contains(trackInDir.Name(), trackFileName) {
					counter++
				}
			}
		}

		if counter == len(p.Tracks) && len(p.Tracks) > 0 {
			pDownloadSpinner.Success(fmt.Sprintf("Плейлист уже скачан: %s", p.Title))
			continue
		}

		data, _ := json.MarshalIndent(p.Tracks, "", "  ")
		os.WriteFile(filepath.Join(_savePath, fmt.Sprintf("%s-music-data.json", title)), data, 0644)

		var toDownload []api.Audio
		namingIndex := 1

		for _, audio := range p.Tracks {
			found := false
			for _, dFile := range tracksInDir {
				if strings.Contains(dFile.Name(), utils.GetNormalFileName(audio.Artist, audio.Title)) {
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

		pDownloadSpinner.Success(fmt.Sprintf("Начинаю загрузку: %s", p.Title))
		downloader.DownloadBatchOfTracks(toDownload, _savePath, namingIndex)
		pterm.Success.Printf("Плейлист скачан: %s\n\n", p.Title)
	}
	return nil
}
