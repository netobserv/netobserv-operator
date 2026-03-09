package config

import (
	_ "embed"
)

//go:embed local-config.yaml
var rawLocalConfig []byte

func GetLokiConfigStr() string {
	return string(rawLocalConfig)
}
