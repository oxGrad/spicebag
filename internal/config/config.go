package config

import (
	"os"

	"github.com/BurntSushi/toml"
)

// GotenbergRuntime controls how the dashboard starts and stops Gotenberg.
// Valid values: "docker-compose" (default), "docker", "podman-compose", "podman".
type GotenbergRuntime string

const (
	RuntimeDockerCompose GotenbergRuntime = "docker-compose"
	RuntimeDocker        GotenbergRuntime = "docker"
	RuntimePodmanCompose GotenbergRuntime = "podman-compose"
	RuntimePodman        GotenbergRuntime = "podman"
)

type Config struct {
	DashboardPort     int              `toml:"dashboard_port"`
	GotenbergURL      string           `toml:"gotenberg_url"`
	GotenbergRuntime  GotenbergRuntime `toml:"gotenberg_runtime"`
}

func defaults() Config {
	return Config{
		DashboardPort:    8080,
		GotenbergURL:     "http://localhost:3000",
		GotenbergRuntime: RuntimeDockerCompose,
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
