package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	HTTP     HTTPConfig     `yaml:"http"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	Files    FilesConfig    `yaml:"files"`
	JWT      JWTConfig      `yaml:"jwt"`
}

type HTTPConfig struct {
	Port int `yaml:"port"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Name     string `yaml:"name"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type RedisConfig struct {
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	Password  string `yaml:"password"`
	DB        int    `yaml:"db"`
	MaxActive int    `yaml:"max_active"`
}

type FilesConfig struct {
	MaxSizeMB int              `yaml:"max_size_mb"`
	User      UserFilesConfig  `yaml:"user"`
	Video     VideoFilesConfig `yaml:"video"`
	Waste     WasteFilesConfig `yaml:"waste"`
	Photo     PhotoFilesConfig `yaml:"photo"`
}

type UserFilesConfig struct {
	AvatarDir string `yaml:"avatar_dir"`
}

type VideoFilesConfig struct {
	SaveDir  string `yaml:"save_dir"`
	TempDir  string `yaml:"temp_dir"`
	CoverDir string `yaml:"cover_dir"`
}

type WasteFilesConfig struct {
	Dir string `yaml:"dir"`
}

type PhotoFilesConfig struct {
	Dir string `yaml:"dir"`
}

type JWTConfig struct {
	Key         string `yaml:"key"`
	Issuer      string `yaml:"issuer"`
	Subject     string `yaml:"subject"`
	ExpireHours int    `yaml:"expire_hours"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
