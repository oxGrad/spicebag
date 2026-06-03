package config

import (
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	DashboardPort int    `toml:"dashboard_port"`
	GotenbergURL  string `toml:"gotenberg_url"`
}

func defaults() Config {
	return Config{
		DashboardPort: 8080,
		GotenbergURL:  "http://localhost:3000",
	}
}

func Load(path string) (Config, error) {
	cfg := defaults()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	_, err = toml.Decode(string(data), &cfg)
	return cfg, err
}

func Save(path string, cfg Config) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}
