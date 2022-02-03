package config

import (
	"fmt"
	"net/url"
	"reflect"

	gitlab2 "github.com/dogboy21/poddy/gitlab"
	"github.com/dogboy21/poddy/models"

	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

type OauthRepositoryProviderConfig struct {
	ID            string   `mapstructure:"id"`
	Type          string   `mapstructure:"type"`
	ClientID      string   `mapstructure:"client_id"`
	ClientSecret  string   `mapstructure:"client_secret"`
	BaseUrl       string   `mapstructure:"base_url"`
	AuthEndpoint  string   `mapstructure:"auth_endpoint"`
	TokenEndpoint string   `mapstructure:"token_endpoint"`
	Scopes        []string `mapstructure:"scopes"`

	parsedBaseUrl *url.URL
	OauthConfig   *oauth2.Config
	Host          string
}

func (c *OauthRepositoryProviderConfig) GetOauthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  c.parsedBaseUrl.ResolveReference(&url.URL{Path: c.AuthEndpoint}).String(),
			TokenURL: c.parsedBaseUrl.ResolveReference(&url.URL{Path: c.TokenEndpoint}).String(),
		},
		RedirectURL: ServerUrl().ResolveReference(&url.URL{Path: fmt.Sprintf("/oauth/redirect/%s", c.ID)}).String(),
		Scopes:      c.Scopes,
	}
}

func (c *OauthRepositoryProviderConfig) GetRepositoryProvider(source oauth2.TokenSource) (models.RepositoryProvider, error) {
	if c.Type == "gitlab" {
		return gitlab2.GitlabApi(c.parsedBaseUrl, source), nil
	}

	return nil, fmt.Errorf("invalid provider type: %s", c.Type)
}

func GetOauthConfigs() ([]OauthRepositoryProviderConfig, error) {
	providersSlice := viper.Get("providers")
	sliceLen := reflect.ValueOf(providersSlice).Len()

	configSlice := make([]OauthRepositoryProviderConfig, sliceLen)

	for i := 0; i < sliceLen; i++ {
		var cfg OauthRepositoryProviderConfig
		if err := viper.Sub(fmt.Sprintf("providers.%d", i)).Unmarshal(&cfg); err != nil {
			return nil, err
		}

		parsedUrl, err := url.Parse(cfg.BaseUrl)
		if err != nil {
			return nil, err
		}

		cfg.parsedBaseUrl = parsedUrl

		cfg.OauthConfig = cfg.GetOauthConfig()
		cfg.Host = parsedUrl.Host
		configSlice[i] = cfg
	}

	return configSlice, nil
}
