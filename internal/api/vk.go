package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
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
		client:      &http.Client{},
	}
}

func (s *VkApiService) SetToken(token string, userID int) {
	s.accessToken = token
	s.userID = userID
}

func (s *VkApiService) get(url string) ([]byte, error) {
	resp, err := s.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (s *VkApiService) GetAudiosList() ([]Audio, error) {
	url := fmt.Sprintf("https://api.vk.com/method/audio.get?owner_id=%d&access_token=%s&v=5.131&count=6000", s.userID, s.accessToken)
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

	return vkResp.Response.Items, nil
}

func (s *VkApiService) GetPlaylists() ([]Playlist, error) {
	url := fmt.Sprintf("https://api.vk.com/method/audio.getPlaylists?owner_id=%d&access_token=%s&v=5.131&count=200", s.userID, s.accessToken)
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

	return vkResp.Response.Items, nil
}

func (s *VkApiService) GetTracksOfUserPlaylist(playlistID int) ([]Audio, error) {
	url := fmt.Sprintf("https://api.vk.com/method/audio.get?owner_id=%d&access_token=%s&v=5.131&playlist_id=%d", s.userID, s.accessToken, playlistID)
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

	return vkResp.Response.Items, nil
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

	url := fmt.Sprintf("https://api.vk.com/method/audio.get?owner_id=%s&playlist_id=%s&access_key=%s&access_token=%s&v=5.131&count=200", parts[0], parts[1], parts[2], s.accessToken)
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

	return vkResp.Response.Items, nil
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
		fmt.Sprintf("https://api.vk.com/method/audio.getById?audios=%s_%s&access_token=%s&v=5.131", ownerID, audioID, s.accessToken),
		fmt.Sprintf("https://api.vk.com/method/audio.getById?audios=-%s_%s&access_token=%s&v=5.131", ownerID, audioID, s.accessToken),
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
