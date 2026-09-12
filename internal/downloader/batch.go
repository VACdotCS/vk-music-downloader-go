package downloader

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gosuri/uilive"
	"github.com/pterm/pterm"
	"vk-music-downloader-go/internal/api"
	"vk-music-downloader-go/internal/cache"
	"vk-music-downloader-go/internal/pool"
	"vk-music-downloader-go/internal/ui"
	"vk-music-downloader-go/internal/utils"
)

func generateHash() string {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return "abcd"
	}
	return hex.EncodeToString(bytes)
}

func DownloadBatchOfTracks(toDownload []api.Audio, savePath string, startNamingIndex int) {
	batchSize := 10
	iterationsCount := (len(toDownload) + batchSize - 1) / batchSize

	maxTitleLength := -1
	for i, audio := range toDownload {
		title := fmt.Sprintf("%d. Скачиваю: %s - %s", startNamingIndex+i, audio.Artist, audio.Title)
		length := utf8.RuneCountInString(title)
		if length > maxTitleLength {
			maxTitleLength = length
		}
	}

	namingIndex := startNamingIndex
	dl := NewDownloader()

	for it := 0; it < iterationsCount; it++ {
		start := it * batchSize
		end := start + batchSize
		if end > len(toDownload) {
			end = len(toDownload)
		}

		batch := toDownload[start:end]
		workerPool := pool.NewWorkerPool(batchSize)

		writer := uilive.New()
		writer.Start()

		lines := make([]string, len(batch))
		var linesMu sync.Mutex

		for i, audio := range batch {
			currentIndex := namingIndex + i
			taskTitle := fmt.Sprintf("%d. Скачиваю: %s - %s", currentIndex, audio.Artist, audio.Title)
			lines[i] = "⏳ " + taskTitle
		}

		done := make(chan struct{})
		go func() {
			ticker := time.NewTicker(150 * time.Millisecond) // Увеличиваем интервал (меньше нагрузка на консоль)
			defer ticker.Stop()
			
			var lastContent string
			
			for {
				select {
				case <-ticker.C:
					linesMu.Lock()
					currentContent := strings.Join(lines, "\n")
					
					if currentContent != lastContent {
						for _, l := range lines {
							fmt.Fprintln(writer, l)
						}
						writer.Flush()
						lastContent = currentContent
					}
					linesMu.Unlock()
				case <-done:
					linesMu.Lock()
					for _, l := range lines {
						fmt.Fprintln(writer, l)
					}
					writer.Flush()
					linesMu.Unlock()
					return
				}
			}
		}()

		var wg sync.WaitGroup

		for i, audio := range batch {
			audio := audio // capture loop variable
			indexInBatch := i
			currentIndex := namingIndex
			namingIndex++

			fileName := utils.GetNormalFileName(audio.Artist, audio.Title)
			tempFilePath := filepath.Join(savePath, fmt.Sprintf("temp-%s.ts", generateHash()))
			mp3FilePath := filepath.Join(savePath, fmt.Sprintf("%d. %s", currentIndex, fileName))

			taskTitle := fmt.Sprintf("%d. Скачиваю: %s - %s", currentIndex, audio.Artist, audio.Title)

			wg.Add(1)
			workerPool.AddTask(func() error {
				defer wg.Done()

				progressCb := func(percentage float64) {
					progressStr := ui.RenderDownloaderProgress(percentage, utf8.RuneCountInString(taskTitle), maxTitleLength, 0)
					
					linesMu.Lock()
					lines[indexInBatch] = "🔄 " + taskTitle + " " + progressStr
					linesMu.Unlock()
				}

				err := dl.ProcessStream(audio.URL, tempFilePath, mp3FilePath, progressCb)
				if err != nil {
					_ = cache.CatchAudioStreamError(err, audio, fileName)
					
					linesMu.Lock()
					lines[indexInBatch] = pterm.Red("❌ " + fmt.Sprintf("%d. Ошибка скачивания: %s - %s", currentIndex, audio.Artist, audio.Title))
					linesMu.Unlock()
					return err
				}

				successTitle := fmt.Sprintf("%d. Трек успешно скачан: %s", currentIndex, fileName)
				marginCorr := int(mathAbs(utf8.RuneCountInString(successTitle) - utf8.RuneCountInString(taskTitle)))
				progressStr := ui.RenderDownloaderProgress(1.0, utf8.RuneCountInString(taskTitle), maxTitleLength, marginCorr)
				
				linesMu.Lock()
				lines[indexInBatch] = pterm.Green("✅ " + successTitle + " " + progressStr)
				linesMu.Unlock()
				
				return nil
			})
		}

		workerPool.Start()
		workerPool.Wait()
		wg.Wait()
		
		close(done)
		writer.Stop()
	}
}

func mathAbs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
