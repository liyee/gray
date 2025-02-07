package gconf

import (
	"os"
	"path/filepath"
)

const (
	EnvConfigFilePathKey     = "GTCP_CONFIG_FILE_PATH"
	EnvDefaultConfigFilePath = "conf/gtcp.json"
)

type gEnv struct {
	configFilePath string
}

var env = new(gEnv)

func init() {
	configFilePath := os.Getenv(EnvConfigFilePathKey)
	if configFilePath == "" {
		pwd, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		configFilePath = filepath.Join(pwd, EnvDefaultConfigFilePath)
	}

	var err error
	configFilePath, err = filepath.Abs(configFilePath)
	if err != nil {
		panic(err)
	}
	env.configFilePath = configFilePath
}

func GetConfigFilePath() string {
	return env.configFilePath
}
