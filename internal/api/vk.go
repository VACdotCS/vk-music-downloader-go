package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
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

// Позже сюда можно добавить методы GetPlaylists, GetTracksOfUserPlaylist, и т.д.
