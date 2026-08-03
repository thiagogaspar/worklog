package google

import (
	"encoding/json"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
)

type Config struct {
	Enabled      bool   `json:"enabled"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	TokenFile    string `json:"token_file"`
	Email        string `json:"email"`
}

func cfgDir() string { h, _ := os.UserHomeDir(); return filepath.Join(h, ".config/worklog") }

func LoadConfig() (*Config, error) {
	p := filepath.Join(cfgDir(), "google.json")
	d, err := os.ReadFile(p)
	if err != nil { if os.IsNotExist(err) { return &Config{}, nil }; return nil, err }
	var c Config
	return &c, json.Unmarshal(d, &c)
}

func SaveConfig(c *Config) error {
	d, _ := json.MarshalIndent(c, "", "  ")
	return os.WriteFile(filepath.Join(cfgDir(), "google.json"), d, 0600)
}

func tokenPath() string { return filepath.Join(cfgDir(), "google-token.json") }

func LoadToken(p string) (*oauth2.Token, error) {
	d, err := os.ReadFile(p); if err != nil { return nil, err }
	var t oauth2.Token; return &t, json.Unmarshal(d, &t)
}

func SaveToken(p string, t *oauth2.Token) error {
	d, _ := json.MarshalIndent(t, "", "  "); return os.WriteFile(p, d, 0600)
}
