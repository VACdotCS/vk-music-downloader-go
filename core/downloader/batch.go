package downloader

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gosuri/uilive"
	"github.com/pterm/pterm"
	"vk-music-downloader-go/core/api"
	"vk-music-downloader-go/core/cache"
	"vk-music-downloader-go/core/config"
	"vk-music-downloader-go/core/pool"
	"vk-music-downloader-go/core/ui"
	"vk-music-downloader-go/core/utils"
)

func generateHash() string {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return "abcd"
	}
	return hex.EncodeToString(bytes)
}

func DownloadBatchOfTracks(ctx context.Context, toDownload []api.Audio, savePath string, startNamingIndex int) {
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

	// Если контекст не передан, создаем fallback с поддержкой прерываний (для CLI)
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()
	}

	for it := 0; it < iterationsCount; it++ {
		// Проверяем, не была ли отменена загрузка пользователем
		if ctx.Err() != nil {
			pterm.Warning.Println("\nЗагрузка прервана пользователем (Ctrl+C). Очистка временных файлов...")
			cache.ClearTempFiles(savePath)
			return
		}

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
			ticker := time.NewTicker(150 * time.Millisecond)
			defer ticker.Stop()
			
			var lastContent string
			
			for {
				select {
				case <-ticker.C:
					linesMu.Lock()
					currentContent := strings.Join(lines, "\n")
					
					if currentContent != lastContent {
						if !config.IsGUI {
							for _, l := range lines {
								fmt.Fprintln(writer, l)
							}
							writer.Flush()
						}
						lastContent = currentContent
					}
					linesMu.Unlock()
				case <-done:
					linesMu.Lock()
					if !config.IsGUI {
						for _, l := range lines {
							fmt.Fprintln(writer, l)
						}
						writer.Flush()
					}
					linesMu.Unlock()
					return
				}
			}
		}()

		var wg sync.WaitGroup

		for i, audio := range batch {
			audio := audio
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

				// Если контекст отменен до старта скачивания
				if ctx.Err() != nil {
					linesMu.Lock()
					lines[indexInBatch] = pterm.Yellow("🛑 " + fmt.Sprintf("%d. Отменено: %s", currentIndex, fileName))
					linesMu.Unlock()
					return ctx.Err()
				}

				progressCb := func(percentage float64) {
					progressStr := ui.RenderDownloaderProgress(percentage, utf8.RuneCountInString(taskTitle), maxTitleLength, 0)
					
					linesMu.Lock()
					lines[indexInBatch] = "🔄 " + taskTitle + " " + progressStr
					linesMu.Unlock()
					
					if GUIProgressCallback != nil {
						GUIProgressCallback(currentIndex, fileName, percentage, "downloading")
					}
				}

				err := dl.ProcessStream(ctx, audio.URL, tempFilePath, mp3FilePath, progressCb)
				if err != nil {
					// Если ошибка из-за Ctrl+C
					if ctx.Err() != nil {
						linesMu.Lock()
						lines[indexInBatch] = pterm.Yellow("🛑 " + fmt.Sprintf("%d. Прервано: %s", currentIndex, fileName))
						linesMu.Unlock()
						return err
					}

					_ = cache.CatchAudioStreamError(err, audio, fileName)
					
					linesMu.Lock()
					lines[indexInBatch] = pterm.Red("❌ " + fmt.Sprintf("%d. Ошибка скачивания: %s - %s", currentIndex, audio.Artist, audio.Title))
					linesMu.Unlock()
					
					if GUIProgressCallback != nil {
						GUIProgressCallback(currentIndex, fileName, 0, "error")
					}
					return err
				}

				successTitle := fmt.Sprintf("%d. Трек успешно скачан: %s", currentIndex, fileName)
				marginCorr := int(mathAbs(utf8.RuneCountInString(successTitle) - utf8.RuneCountInString(taskTitle)))
				progressStr := ui.RenderDownloaderProgress(1.0, utf8.RuneCountInString(taskTitle), maxTitleLength, marginCorr)
				
				linesMu.Lock()
				lines[indexInBatch] = pterm.Green("✅ " + successTitle + " " + progressStr)
				linesMu.Unlock()
				
				if GUIProgressCallback != nil {
					GUIProgressCallback(currentIndex, fileName, 1.0, "done")
				}
				
				return nil
			})
		}

		workerPool.Start()
		workerPool.Wait()
		wg.Wait()
		
		close(done)
		writer.Stop()

		// Если прервали, сразу выходим из глобального цикла батчей
		if ctx.Err() != nil {
			pterm.Warning.Println("\nЗагрузка прервана пользователем (Ctrl+C). Очистка временных файлов...")
			cache.ClearTempFiles(savePath)
			return
		}
	}
}

var (
	// GUIProgressCallback используется для прокидывания прогресса во фронтенд Wails
	GUIProgressCallback func(index int, title string, percentage float64, status string)
)

func mathAbs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
