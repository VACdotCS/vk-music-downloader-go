package ffmpeg

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pterm/pterm"
)

const ffmpegUrl = "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-win64-gpl.zip"

// GetPath возвращает путь к локальному или системному FFmpeg
func GetPath() string {
	exePath, _ := os.Executable()
	localFfmpeg := filepath.Join(filepath.Dir(exePath), "ffmpeg.exe")
	if _, err := os.Stat(localFfmpeg); err == nil {
		return localFfmpeg
	}
	
	// Если локального нет, ищем в глобальных переменных среды (PATH)
	globalPath, err := exec.LookPath("ffmpeg")
	if err == nil {
		absPath, err := filepath.Abs(globalPath)
		if err == nil {
			return absPath
		}
		return globalPath
	}
	
	return "ffmpeg (не установлен)"
}

// CheckAndDownload проверяет наличие ffmpeg. Если его нет, скачивает и устанавливает локально.
func CheckAndDownload() error {
	_, err := exec.LookPath("ffmpeg")
	if err == nil {
		return nil // Уже установлен в системе
	}

	// Проверяем локально
	exePath, _ := os.Executable()
	localFfmpeg := filepath.Join(filepath.Dir(exePath), "ffmpeg.exe")
	if _, err := os.Stat(localFfmpeg); err == nil {
		return nil // Есть локальная копия
	}

	pterm.Warning.Println("FFmpeg не найден в системе (он нужен для конвертации треков в .mp3).")
	pterm.Info.Println("Сейчас мы автоматически скачаем и установим его (около 130 МБ)...")

	// Скачиваем ZIP
	zipPath := filepath.Join(filepath.Dir(exePath), "ffmpeg_temp.zip")
	
	spinner, _ := pterm.DefaultSpinner.Start("Скачивание FFmpeg...")
	
	err = downloadFile(ffmpegUrl, zipPath)
	if err != nil {
		spinner.Fail("Ошибка скачивания FFmpeg: " + err.Error())
		return err
	}

	spinner.UpdateText("Распаковка FFmpeg...")
	
	err = extractFfmpegExe(zipPath, localFfmpeg)
	if err != nil {
		spinner.Fail("Ошибка распаковки: " + err.Error())
		_ = os.Remove(zipPath)
		return err
	}

	_ = os.Remove(zipPath)
	spinner.Success("FFmpeg успешно установлен в папку с программой!")
	
	return nil
}

func downloadFile(url string, filepath string) error {
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	return err
}

func extractFfmpegExe(zipPath string, targetExe string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if strings.HasSuffix(f.Name, "ffmpeg.exe") {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			targetFile, err := os.OpenFile(targetExe, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer targetFile.Close()

			_, err = io.Copy(targetFile, rc)
			if err != nil {
				return err
			}
			
			return nil
		}
	}

	return fmt.Errorf("ffmpeg.exe не найден внутри архива")
}
