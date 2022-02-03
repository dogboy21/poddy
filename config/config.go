package config

import (
	"encoding/hex"
	"log"
	"net/url"
	"strings"

	"github.com/spf13/viper"
)

const (
	keyServerListenAddress = "server.listenAddress"
	keyServerUrl           = "server.url"
	keyServerCookieSecret  = "server.cookieSecret"
	keyServerSecureCookies = "server.secureCookies"

	keyDeploymentNamespace    = "deployment.namespace"
	keyDeploymentBaseDomain   = "deployment.baseDomain"
	keyDeploymentIngressClass = "deployment.ingressClass"
)

func setDefaults() {
	viper.SetDefault(keyServerListenAddress, ":8080")
	viper.SetDefault(keyServerUrl, "http://poddy.127.0.0.1.nip.io:8080")
	viper.SetDefault(keyServerCookieSecret, "abcdef")
	viper.SetDefault(keyServerSecureCookies, false)

	viper.SetDefault(keyDeploymentNamespace, "poddy-workspaces")
	viper.SetDefault(keyDeploymentBaseDomain, "poddy.127.0.0.1.nip.io")
	viper.SetDefault(keyDeploymentIngressClass, "")
}

func ReadConfig() error {
	setDefaults()

	viper.SetConfigName("poddy")
	viper.SetConfigType("yaml")

	viper.AddConfigPath("config")
	viper.AddConfigPath("data")
	viper.AddConfigPath("/etc/poddy")

	viper.SetEnvPrefix("poddy")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	//TODO Validate config entries

	return nil
}

func mustParseHex(val []byte, err error) []byte {
	if err != nil {
		log.Fatalf("failed to parse config value as hex: %v\n", err)
	}

	return val
}

func ServerListenAddress() string {
	return viper.GetString(keyServerListenAddress)
}

func ServerUrl() *url.URL {
	urlObj, err := url.Parse(viper.GetString(keyServerUrl))
	if err != nil {
		log.Fatalf("failed to parse app url: %v\n", err)
	}

	return urlObj
}

func ServerCookieSecret() []byte {
	return mustParseHex(hex.DecodeString(viper.GetString(keyServerCookieSecret)))
}

func ServerSecureCookies() bool {
	return viper.GetBool(keyServerSecureCookies)
}

func DeploymentNamespace() string {
	return viper.GetString(keyDeploymentNamespace)
}

func DeploymentBaseDomain() string {
	return viper.GetString(keyDeploymentBaseDomain)
}

func DeploymentIngressClass() string {
	return viper.GetString(keyDeploymentIngressClass)
}
