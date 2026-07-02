package shared

import (
	"encoding/json"
	"os"
)

type AppConfig struct {
	LLMProviders struct {
		FrontModel ModelConfig `json:"front_model"`
	} `json:"llm_providers"`
}

type ModelConfig struct {
	BaseURL string `json:"base_url"`
	ApiKey  string `json:"api_key"`
	Model   string `json:"model"`

	ContextWindow int `json:"context_window"`
}

func LoadAppConfig(path string) (AppConfig, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return AppConfig{}, err
	}
	var config AppConfig
	err = json.Unmarshal(content, &config)
	if err != nil {
		return AppConfig{}, err
	}
	return config, nil
}
