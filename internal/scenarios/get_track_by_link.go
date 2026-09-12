package scenarios

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/pterm/pterm"
	"vk-music-downloader-go/internal/api"
	"vk-music-downloader-go/internal/cache"
	"vk-music-downloader-go/internal/downloader"
	"vk-music-downloader-go/internal/utils"
)

func GetTrackByLinkScenario(savePath string, vkService *api.VkApiService) error {
	var link string
	prompt := &survey.Input{Message: "Введите ссылку на трек:"}
	if err := survey.AskOne(prompt, &link); err != nil {
		return err
	}

	audioData, err := vkService.GetAudioByLink(link)
	if err != nil {
		pterm.Error.Println("Ошибка:", err)
		return err
	}

	fileName := utils.GetNormalFileName(audioData.Artist, audioData.Title)
	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Скачиваю трек: %s", fileName))

	tempFilePath := filepath.Join(savePath, "temp-single.ts")
	mp3FilePath := filepath.Join(savePath, fileName)

	dl := downloader.NewDownloader()
	
	// Контекст для отмены одиночной загрузки по Ctrl+C
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	err = dl.ProcessStream(ctx, audioData.URL, tempFilePath, mp3FilePath, nil)
	if err != nil {
		if ctx.Err() != nil {
			spinner.Fail("Загрузка прервана пользователем")
			cache.ClearTempFiles(savePath)
			return err
		}
		_ = cache.CatchAudioStreamError(err, *audioData, fileName)
		spinner.Fail("Ошибка скачивания")
		return err
	}

	spinner.Success(fmt.Sprintf("Трек скачан: %s", mp3FilePath))
	return nil
}
