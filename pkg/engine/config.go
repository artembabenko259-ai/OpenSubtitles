package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	OpenRouterAPIKey string `json:"openrouter_api_key"`
	WhisperModel     string `json:"whisper_model"`
	TranslationModel string `json:"translation_model"`
	TargetLanguage   string `json:"target_language"`
	SubtitleStyle    string `json:"subtitle_style"`
}

func DefaultConfig() Config {
	return Config{
		OpenRouterAPIKey: "",
		WhisperModel:     "openai/whisper-large-v3-turbo",
		TranslationModel: "google/gemini-2.5-flash",
		TargetLanguage:   "Ukrainian",
		SubtitleStyle:    "CapCut",
	}
}

func getConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".opensubtitles")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func LoadConfig() (Config, error) {
	path, err := getConfigPath()
	if err != nil {
		return DefaultConfig(), err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		cfg := DefaultConfig()
		_ = SaveConfig(cfg)
		return cfg, nil
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig(), nil
	}

	if cfg.WhisperModel == "" {
		cfg.WhisperModel = "openai/whisper-large-v3-turbo"
	}
	if cfg.TranslationModel == "" {
		cfg.TranslationModel = "google/gemini-2.5-flash"
	}
	if cfg.TargetLanguage == "" {
		cfg.TargetLanguage = "Ukrainian"
	}

	return cfg, nil
}

func SaveConfig(cfg Config) error {
	path, err := getConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}
