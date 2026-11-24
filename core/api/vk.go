package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type VkApiService struct {
	accessToken string
	userID      int
	client      *http.Client
}

func NewVkApiService(accessToken string, userID int) *VkApiService {
	return &VkApiService{
		accessToken: accessToken,
		userID:      userID,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *VkApiService) SetToken(token string, userID int) {
	s.accessToken = token
	s.userID = userID
}

func (s *VkApiService) get(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (s *VkApiService) GetAudiosList() ([]Audio, error) {
	var allAudios []Audio
	offset := 0
	batchSize := 1000

	for {
		url := fmt.Sprintf("https://api.vk.ru/method/audio.get?owner_id=%d&access_token=%s&v=5.289&client_id=6287487&count=%d&offset=%d", s.userID, s.accessToken, batchSize, offset)
		data, err := s.get(url)
		if err != nil {
			return nil, err
		}

		var vkResp VkAudioResponse
		if err := json.Unmarshal(data, &vkResp); err != nil {
			return nil, err
		}
		if vkResp.Error != nil {
			return nil, fmt.Errorf("API Error: %s", vkResp.Error.ErrorMsg)
		}

		items := vkResp.Response.Items
		if len(items) == 0 {
			break
		}

		allAudios = append(allAudios, items...)
		offset += len(items)

		if offset >= vkResp.Response.Count || len(items) < batchSize {
			break
		}
	}

	return allAudios, nil
}

func (s *VkApiService) GetPlaylists() ([]Playlist, error) {
	var allPlaylists []Playlist
	offset := 0
	batchSize := 200

	for {
		url := fmt.Sprintf("https://api.vk.ru/method/audio.getPlaylists?owner_id=%d&access_token=%s&v=5.289&client_id=6287487&count=%d&offset=%d", s.userID, s.accessToken, batchSize, offset)
		data, err := s.get(url)
		if err != nil {
			return nil, err
		}

		var vkResp VkPlaylistsResponse
		if err := json.Unmarshal(data, &vkResp); err != nil {
			return nil, err
		}
		if vkResp.Error != nil {
			return nil, fmt.Errorf("API Error: %s", vkResp.Error.ErrorMsg)
		}

		items := vkResp.Response.Items
		if len(items) == 0 {
			break
		}

		allPlaylists = append(allPlaylists, items...)
		offset += len(items)

		if offset >= vkResp.Response.Count || len(items) < batchSize {
			break
		}
	}

	return allPlaylists, nil
}

func (s *VkApiService) GetTracksOfUserPlaylist(playlistID int) ([]Audio, error) {
	var allAudios []Audio
	offset := 0
	batchSize := 1000

	for {
		url := fmt.Sprintf("https://api.vk.ru/method/audio.get?owner_id=%d&access_token=%s&v=5.289&client_id=6287487&playlist_id=%d&count=%d&offset=%d", s.userID, s.accessToken, playlistID, batchSize, offset)
		data, err := s.get(url)
		if err != nil {
			return nil, err
		}

		var vkResp VkAudioResponse
		if err := json.Unmarshal(data, &vkResp); err != nil {
			return nil, err
		}
		if vkResp.Error != nil {
			return nil, fmt.Errorf("API Error: %s", vkResp.Error.ErrorMsg)
		}

		items := vkResp.Response.Items
		if len(items) == 0 {
			break
		}

		allAudios = append(allAudios, items...)
		offset += len(items)

		if offset >= vkResp.Response.Count || len(items) < batchSize {
			break
		}
	}

	return allAudios, nil
}

func (s *VkApiService) GetTracksOfPlaylistByLink(link string) ([]Audio, error) {
	re := regexp.MustCompile(`[-]?\d+_\d+_[a-f0-9]+`)
	match := re.FindString(link)
	if match == "" {
		return nil, fmt.Errorf("неверная ссылка на плейлист")
	}

	parts := strings.Split(match, "_")
	if len(parts) != 3 {
		return nil, fmt.Errorf("неверный формат ссылки на плейлист")
	}

	var allAudios []Audio
	offset := 0
	batchSize := 1000

	for {
		url := fmt.Sprintf("https://api.vk.ru/method/audio.get?owner_id=%s&playlist_id=%s&access_key=%s&access_token=%s&v=5.289&client_id=6287487&count=%d&offset=%d", parts[0], parts[1], parts[2], s.accessToken, batchSize, offset)
		data, err := s.get(url)
		if err != nil {
			return nil, err
		}

		var vkResp VkAudioResponse
		if err := json.Unmarshal(data, &vkResp); err != nil {
			return nil, err
		}
		if vkResp.Error != nil {
			return nil, fmt.Errorf("API Error: %s", vkResp.Error.ErrorMsg)
		}

		items := vkResp.Response.Items
		if len(items) == 0 {
			break
		}

		allAudios = append(allAudios, items...)
		offset += len(items)

		if offset >= vkResp.Response.Count || len(items) < batchSize {
			break
		}
	}

	return allAudios, nil
}

func (s *VkApiService) GetAudioByLink(link string) (*Audio, error) {
	re := regexp.MustCompile(`audio-?(\d+)_(\d+)`)
	matches := re.FindStringSubmatch(link)
	if len(matches) < 3 {
		return nil, fmt.Errorf("неверная ссылка на трек")
	}

	ownerID := matches[1]
	audioID := matches[2]

	urls := []string{
		fmt.Sprintf("https://api.vk.ru/method/audio.getById?audios=%s_%s&access_token=%s&v=5.289&client_id=6287487", ownerID, audioID, s.accessToken),
		fmt.Sprintf("https://api.vk.ru/method/audio.getById?audios=-%s_%s&access_token=%s&v=5.289&client_id=6287487", ownerID, audioID, s.accessToken),
	}

	for _, u := range urls {
		data, err := s.get(u)
		if err != nil {
			continue
		}

		var vkResp VkAudioByIdResponse
		if err := json.Unmarshal(data, &vkResp); err != nil {
			continue
		}

		if vkResp.Error != nil && vkResp.Error.ErrorCode != 100 {
			return nil, fmt.Errorf("API Error: %s", vkResp.Error.ErrorMsg)
		}

		if len(vkResp.Response) > 0 {
			return &vkResp.Response[0], nil
		}
	}

	return nil, fmt.Errorf("трек не найден")
}
