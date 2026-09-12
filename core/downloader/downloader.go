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
	"syscall"

	"vk-music-downloader-go/core/ffmpeg"
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
	req.Header.Set("User-Agent", "KateMobileAndroid/56 lite-460 (Android 4.4.2; SDK 19; x86; unknown Android SDK built for x86; en)")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (d *Downloader) ParseM3U8(ctx context.Context, m3u8Url string) ([]Segment, error) {
	data, err := d.doRequest(ctx, m3u8Url)
	if err != nil {
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

// ProcessStream скачивает и расшифровывает все сегменты, затем склеивает и вызывает FFmpeg
func (d *Downloader) ProcessStream(ctx context.Context, m3u8Url, outputTsFile, outputMp3File string, progressCb func(float64)) error {
	segments, err := d.ParseM3U8(ctx, m3u8Url)
	if err != nil {
		return err
	}

	tsFile, err := os.Create(outputTsFile)
	if err != nil {
		return err
	}

	total := len(segments)
	for i, segment := range segments {
		// Проверка отмены между кусками
		if ctx.Err() != nil {
			tsFile.Close()
			return ctx.Err()
		}

		var key, iv []byte
		if segment.Key != nil && segment.Key.Method == "AES-128" {
			key, err = d.doRequest(ctx, segment.Key.URI)
			if err != nil {
				tsFile.Close()
				return err
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
			tsFile.Close()
			return err
		}

		if progressCb != nil {
			progressCb(float64(i) / float64(total))
		}

		_, err = tsFile.Write(decryptedData)
		if err != nil {
			tsFile.Close()
			return err
		}
	}
	tsFile.Close()

	err = TsToMp3(ctx, outputTsFile, outputMp3File)
	if err == nil {
		_ = os.Remove(outputTsFile)
	}

	return err
}

func TsToMp3(ctx context.Context, inputTs, outputMp3 string) error {
	cmd := exec.CommandContext(ctx, ffmpeg.GetPath(), "-y", "-i", inputTs, "-acodec", "libmp3lame", "-f", "mp3", outputMp3)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("ffmpeg error: %v, stderr: %s", err, stderr.String())
	}
	return nil
}
