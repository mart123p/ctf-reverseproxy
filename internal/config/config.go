package config

import (
	"fmt"

	"github.com/spf13/viper"
)

func Init() {
	setupFile()
	setupDefault()
	validate()
}

func setupFile() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/ctf-reverseproxy/")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			panic("Error: The config file was not found. Make sure that a config file is available")
		} else {
			// Config file was found but another error was produced
			panic(err)
		}
	}
}

func setupDefault() {
	viper.SetDefault(CReverseProxyHost, "")
	viper.SetDefault(CReverseProxyPort, "8000")
	viper.SetDefault(CReverseProxySessionHeader, "X-Session-Id")
	viper.SetDefault(CReverseProxySessionTimeout, "300")

	viper.SetDefault(CMgmtHost, "")
	viper.SetDefault(CMgmtPort, "8080")

	viper.SetDefault(CDockerHost, "unix:///var/run/docker.sock")

	viper.SetDefault(CDockerNetwork, "ctf-bridge")
	viper.SetDefault(CDockerComposeWorkdir, ".")
	viper.SetDefault(CDockerComposeFile, "docker-compose.yml")

}

func GetString(key string) string {
	return viper.GetString(key)
}

func GetInt(key string) int {
	return viper.GetInt(key)
}

func GetInt64(key string) int64 {
	return viper.GetInt64(key)
}

func GetAddr(hostname string, portname string) string {
	return fmt.Sprintf("%s:%d", GetString(hostname), GetInt(portname))
}

func validate() {
	//Check if salt and key are set
	if viper.GetString(CReverseProxySessionSalt) == "" {
		panic("Error: The session salt is not set. Please set it in the config file")
	}

	if viper.GetString(CMgmtKey) == "" {
		panic("Error: The management key is not set. Please set it in the config file")
	}
}
