package config

import (
	"encoding/json"
	"os"
)

var IsGUI = false

type Config struct {
	SavePath string `json:"save_path,omitempty"`
	Token    *Token `json:"token,omitempty"`
}

type Token struct {
	AccessToken string `json:"access_token"`
	Expires     int64  `json:"expires"`
	UserID      int    `json:"user_id"`
}

func LoadConfig() (*Config, error) {
	data, err := os.ReadFile("config.json")
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}

	var cfg Config
	if len(data) > 0 {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
	}
	return &cfg, nil
}

func SaveConfig(cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile("config.json", data, 0644)
}
