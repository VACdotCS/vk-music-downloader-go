package downloader

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"vk-music-downloader-go/core/ffmpeg"
	"vk-music-downloader-go/core/logger"
)

type SegmentKey struct {
	Method string
	URI    string
	IV     string
}

type Segment struct {
	URL string
	Key *SegmentKey
}

type Downloader struct {
	client *http.Client
}

func NewDownloader() *Downloader {
	return &Downloader{
		client: &http.Client{},
	}
}

func (d *Downloader) doRequest(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (d *Downloader) ParseM3U8(ctx context.Context, m3u8Url string) ([]Segment, error) {
	data, err := d.doRequest(ctx, m3u8Url)
	if err != nil {
		logger.Error("ParseM3U8: ошибка запроса плейлиста: %v", err)
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	var segments []Segment
	var currentKey *SegmentKey
	var segmentPrefix string

	methodRe := regexp.MustCompile(`METHOD=([^,]+)`)
	uriRe := regexp.MustCompile(`URI="([^"]+)"`)
	ivRe := regexp.MustCompile(`IV=([^,]+)`)

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		if strings.HasPrefix(line, "#EXT-X-KEY:") {
			methodMatch := methodRe.FindStringSubmatch(line)
			uriMatch := uriRe.FindStringSubmatch(line)
			ivMatch := ivRe.FindStringSubmatch(line)

			method := ""
			uri := ""
			iv := ""
			if len(methodMatch) > 1 {
				method = methodMatch[1]
			}
			if len(uriMatch) > 1 {
				uri = uriMatch[1]
			}
			if len(ivMatch) > 1 {
				iv = ivMatch[1]
			}

			if segmentPrefix == "" && uri != "" && method != "NONE" {
				segmentPrefix = strings.Replace(uri, "key.pub?siren=1", "", 1)
			}

			if method == "AES-128" {
				currentKey = &SegmentKey{Method: method, URI: uri, IV: iv}
			} else {
				currentKey = nil
			}
		} else if strings.HasPrefix(line, "#EXTINF:") {
			if i+1 < len(lines) {
				i++
				segmentUrl := strings.TrimSpace(lines[i])
				segments = append(segments, Segment{
					URL: segmentPrefix + segmentUrl,
					Key: currentKey,
				})
			}
		}
	}

	if len(segments) == 0 {
		return nil, fmt.Errorf("m3u8 плейлист не содержит сегментов")
	}

	return segments, nil
}

func (d *Downloader) downloadAndDecryptSegment(ctx context.Context, segmentUrl string, key []byte, iv []byte) ([]byte, error) {
	encryptedData, err := d.doRequest(ctx, segmentUrl)
	if err != nil {
		return nil, err
	}

	if len(key) == 0 {
		return encryptedData, nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(encryptedData)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext is not a multiple of the block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	decryptedData := make([]byte, len(encryptedData))
	mode.CryptBlocks(decryptedData, encryptedData)

	paddingLen := int(decryptedData[len(decryptedData)-1])
	if paddingLen > 0 && paddingLen <= aes.BlockSize {
		decryptedData = decryptedData[:len(decryptedData)-paddingLen]
	}

	return decryptedData, nil
}

func (d *Downloader) downloadDirect(ctx context.Context, audioUrl, outputMp3File string, progressCb func(float64)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, audioUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	tempFile := outputMp3File + ".part"
	out, err := os.Create(tempFile)
	if err != nil {
		return err
	}

	cleanup := true
	defer func() {
		out.Close()
		if cleanup {
			_ = os.Remove(tempFile)
		}
	}()

	totalSize := resp.ContentLength
	var downloaded int64
	buf := make([]byte, 32*1024)

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		n, rErr := resp.Body.Read(buf)
		if n > 0 {
			if _, wErr := out.Write(buf[:n]); wErr != nil {
				return wErr
			}
			downloaded += int64(n)
			if progressCb != nil && totalSize > 0 {
				progressCb(float64(downloaded) / float64(totalSize))
			}
		}
		if rErr != nil {
			if rErr == io.EOF {
				break
			}
			return rErr
		}
	}

	if downloaded == 0 {
		return fmt.Errorf("скачанный файл пуст (0 байт)")
	}

	out.Close()
	cleanup = false

	_ = os.Remove(outputMp3File)
	return os.Rename(tempFile, outputMp3File)
}

// ProcessStream скачивает трек: определяет тип (прямой MP3 или M3U8 HLS поток), скачивает и при необходимости конвертирует.
func (d *Downloader) ProcessStream(ctx context.Context, streamUrl, outputTsFile, outputMp3File string, progressCb func(float64)) error {
	if strings.TrimSpace(streamUrl) == "" {
		return fmt.Errorf("URL аудиозаписи пуст (трек заблокирован)")
	}

	// Если это прямой MP3 (например okcdn.ru или ссылка на .mp3 файл)
	if !strings.Contains(streamUrl, ".m3u8") {
		return d.downloadDirect(ctx, streamUrl, outputMp3File, progressCb)
	}

	// Иначе это M3U8 HLS поток
	segments, err := d.ParseM3U8(ctx, streamUrl)
	if err != nil {
		return err
	}

	if len(segments) == 0 {
		return fmt.Errorf("m3u8 плейлист не содержит сегментов")
	}

	tsFile, err := os.Create(outputTsFile)
	if err != nil {
		return err
	}

	defer func() {
		tsFile.Close()
		_ = os.Remove(outputTsFile)
	}()

	total := len(segments)
	for i, segment := range segments {
		// Проверка отмены между кусками
		if ctx.Err() != nil {
			return ctx.Err()
		}

		var key, iv []byte
		if segment.Key != nil && segment.Key.Method == "AES-128" {
			key, err = d.doRequest(ctx, segment.Key.URI)
			if err != nil {
				return fmt.Errorf("ошибка получения ключа шифрования: %w", err)
			}

			if segment.Key.IV != "" {
				ivHex := strings.TrimPrefix(segment.Key.IV, "0x")
				iv, _ = hex.DecodeString(ivHex)
			} else {
				iv = make([]byte, 16)
			}
		}

		decryptedData, err := d.downloadAndDecryptSegment(ctx, segment.URL, key, iv)
		if err != nil {
			if ctx.Err() == nil {
				logger.Error("ProcessStream: ошибка загрузки/дешифровки сегмента %d/%d: %v", i+1, total, err)
			}
			return err
		}

		if progressCb != nil {
			progressCb(float64(i) / float64(total))
		}

		_, err = tsFile.Write(decryptedData)
		if err != nil {
			return err
		}
	}
	tsFile.Close()

	err = TsToMp3(ctx, outputTsFile, outputMp3File)
	return err
}

func TsToMp3(ctx context.Context, inputTs, outputMp3 string) error {
	cmd := exec.CommandContext(ctx, ffmpeg.GetPath(), "-y", "-i", inputTs, "-acodec", "libmp3lame", "-f", "mp3", outputMp3)

	setHideWindow(cmd)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		logger.Error("ffmpeg error: %v, stderr: %s", err, stderr.String())
		return fmt.Errorf("ffmpeg error: %v, stderr: %s", err, stderr.String())
	}
	return nil
}
