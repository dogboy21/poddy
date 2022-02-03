package poddy

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dogboy21/poddy/config"
	"github.com/gin-contrib/sessions"
	"golang.org/x/oauth2"
)

func SaveTokenToSession(session sessions.Session, providerConfig *config.OauthRepositoryProviderConfig, source oauth2.TokenSource) error {
	token, err := source.Token()
	if err != nil {
		return err
	}

	jsonToken, err := json.Marshal(token)
	if err != nil {
		return err
	}

	session.Set(fmt.Sprintf("%s_token", providerConfig.ID), string(jsonToken))

	return nil
}

func ReadTokenFromSession(session sessions.Session, providerConfig *config.OauthRepositoryProviderConfig) (oauth2.TokenSource, error) {
	sessionValue := session.Get(fmt.Sprintf("%s_token", providerConfig.ID))
	if sessionValue == nil {
		return nil, nil
	}

	jsonToken := sessionValue.(string)

	token := oauth2.Token{}
	if err := json.Unmarshal([]byte(jsonToken), &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %v", err)
	}

	return providerConfig.OauthConfig.TokenSource(context.Background(), &token), nil
}

func RemoveTokenFromSession(session sessions.Session, providerConfig *config.OauthRepositoryProviderConfig) {
	session.Delete(fmt.Sprintf("%s_token", providerConfig.ID))
}
