package main

import (
	_ "embed"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"time"
)

type Config map[string]Session

type Session struct {
	Dir              string        `json:"dir" yaml:"dir"`
	Files            []string      `json:"files" yaml:"files"`
	CopyTo           string        `json:"copyTo" yaml:"copyTo"`
	MinimumInterval  time.Duration `json:"minimumInterval" yaml:"minimumInterval"`
	TimeToDeleteOld  time.Duration `json:"timeToDeleteOld" yaml:"timeToDeleteOld"`
	CountTodeleteOld int           `json:"countTodeleteOld" yaml:"countTodeleteOld"`

	lastBackupTime time.Time
}

const (
	DefaultConfigName = "config.yaml"
)

//go:embed default_config.yaml
var DefaultConfigContent string

var errs = struct {
	UnsupportedFileType error
}{
	errors.New("unsuppoted file type"),
}

var (
	homeDir, _        = filepath.Abs(filepath.Dir(os.Args[0]))
	defaultConfigPath = homeDir + "/" + DefaultConfigName
)

func initConfig(configPath string) (err error) {
	homeDir, err = filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return fmt.Errorf("failed to get home dir: %w", err)
	}
	var configContent []byte

	switch configPath {
	case "":
		configPath = defaultConfigPath
		configContent = initDefaultConfig()
	default:
		data, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", configPath, err)
		}
		configContent = data
	}

	err = config.parse(configContent)
	if err != nil {
		return err
	}
	return nil
}

func initDefaultConfig() []byte {
	configContent, err := os.ReadFile(defaultConfigPath)
	switch {
	case os.IsNotExist(err):
		file, err := os.Create(defaultConfigPath)
		defer file.Close()
		if err != nil {
			log.Fatal("failed to create default config: ", err)
		}
		_, err = file.WriteString(DefaultConfigContent)
		if err != nil {
			log.Fatal("failed to write default config: ", err)
		}
		log.Info("default config created: ", defaultConfigPath)
		os.Exit(0)

	case err != nil:
		log.Fatal("failed to init default config: ", err)
	}
	return configContent
}

func (cfg *Config) parse(content []byte) (err error) {
	err = yaml.Unmarshal(content, cfg)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return nil
}
