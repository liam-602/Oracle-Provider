package tokentracker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const DefaultConfigPath = "./token-config.json"
const DefaultBlockTimeout = int64(180) // 3 minutes

type Config struct {
	CustomTokens    []CustomToken `json:"tokens"`
	CalcCustomToken []CustomToken `json:"custom_token_price"`
}

type CustomToken struct {
	Symbol      string `json:"symbol"`
	Decimals    int    `json:"decimals"`
	Name        string `json:"name"`
	CoinGeckoId string `json:"coingeckoId"`
}

func (c *Config) ToJSON(file string) *os.File {
	var (
		newFile *os.File
		err     error
	)

	var raw []byte
	if raw, err = json.Marshal(*c); err != nil {
		log.Warn("error marshalling json", "err", err)
		os.Exit(1)
	}

	newFile, err = os.Create(file)
	if err != nil {
		log.Warn("error creating config file", "err", err)
	}
	_, err = newFile.Write(raw)
	if err != nil {
		log.Warn("error writing to config file", "err", err)
	}

	if err := newFile.Close(); err != nil {
		log.Warn("error closing file", "err", err)
	}
	return newFile
}

func GetConfig(filePath string) (*Config, error) {
	var fig Config
	path := DefaultConfigPath
	if file := filePath; file != "" {
		path = file
	}
	err := loadConfig(path, &fig)
	if err != nil {
		log.Info("err loading json file", "err", err.Error())
		return &fig, err
	}

	log.Info("Loaded config", "path", path)

	return &fig, nil
}

func loadConfig(file string, config *Config) error {
	ext := filepath.Ext(file)
	fp, err := filepath.Abs(file)
	if err != nil {
		log.Error("error filepath: ", err)
		return err
	}

	f, err := os.Open(filepath.Clean(fp))
	if err != nil {
		return err
	}

	if ext == ".json" {
		if err = json.NewDecoder(f).Decode(&config); err != nil {
			log.Error("error decoder: ", err)
			return err
		}
	} else {
		return fmt.Errorf("unrecognized extention: %s", ext)
	}
	return nil
}
