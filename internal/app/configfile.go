package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const defaultConfigPath = ".workflow-lock-config.yaml"

type fileConfig struct {
	WorkflowDir string `yaml:"workflows"`
	Lockfile    string `yaml:"lockfile"`
	DefaultHost string `yaml:"default_host"`
	Format      string `yaml:"format"`
}

func resolveConfig(args []string) (string, error) {
	for i := range args {
		arg := args[i]
		if arg == "-config" {
			if i+1 >= len(args) {
				return "", errors.New("missing value for -config")
			}
			return args[i+1], nil
		}
		if len(arg) > len("-config=") && arg[:len("-config=")] == "-config=" {
			return arg[len("-config="):], nil
		}
	}

	_, err := os.Stat(defaultConfigPath)
	if err == nil {
		return defaultConfigPath, nil
	}
	if !os.IsNotExist(err) {
		return "", fmt.Errorf("stat config %s: %w", defaultConfigPath, err)
	}
	return "", nil
}

func loadConfigFile(path string) (fileConfig, error) {
	if path == "" {
		return fileConfig{}, nil
	}

	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return fileConfig{}, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg fileConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fileConfig{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
}
