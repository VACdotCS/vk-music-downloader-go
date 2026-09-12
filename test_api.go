package main

import (
	"fmt"
	"vk-music-downloader-go/core/config"
	"vk-music-downloader-go/core/api"
	"encoding/json"
	"os"
)

func main() {
	cfg, _ := config.LoadConfig()
	vk := api.NewVkApiService(cfg.Token.AccessToken, cfg.Token.UserID)
	pls, err := vk.GetPlaylists()
	if err != nil {
		fmt.Println(err)
		return
	}
	b, _ := json.MarshalIndent(pls, "", "  ")
	os.WriteFile("playlists_debug.json", b, 0644)
	fmt.Println("Wrote to playlists_debug.json")
}
