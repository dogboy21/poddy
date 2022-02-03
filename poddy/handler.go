package poddy

import (
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func (p *poddy) listOauthProvidersHandler(c *gin.Context) {
	providers := make([]map[string]interface{}, len(p.oauthRepositoryProviderConfigs))
	for i := 0; i < len(providers); i++ {
		providers[i] = map[string]interface{}{
			"id":   p.oauthRepositoryProviderConfigs[i].ID,
			"host": p.oauthRepositoryProviderConfigs[i].Host,
		}
	}

	c.JSON(http.StatusOK, providers)
}

func (p *poddy) oauthAuthHandler(c *gin.Context) {
	provider := p.getProviderForId(c.Param("id"))
	if provider == nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	session := sessions.Default(c)

	randomBytes := make([]byte, 16)
	crand.Read(randomBytes)
	state := hex.EncodeToString(randomBytes)

	session.Set(fmt.Sprintf("%s_state", provider.ID), state)
	session.Save()

	c.Redirect(http.StatusFound, provider.OauthConfig.AuthCodeURL(state))
}

func (p *poddy) oauthRedirectHandler(c *gin.Context) {
	provider := p.getProviderForId(c.Param("id"))
	if provider == nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	stateKey := fmt.Sprintf("%s_state", provider.ID)

	session := sessions.Default(c)
	if c.Query("state") == "" || c.Query("state") != session.Get(stateKey) {
		c.AbortWithStatus(http.StatusBadRequest)
	}

	session.Delete(stateKey)

	token, err := provider.OauthConfig.Exchange(context.Background(), c.Query("code"))
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("failed to exchange code for token: %v\n", err))
		return
	}

	tokenSource := provider.OauthConfig.TokenSource(context.Background(), token)
	if err := SaveTokenToSession(session, provider, tokenSource); err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("failed to save exchanged token to session: %v\n", err))
		return
	}

	session.Save()

	c.Redirect(http.StatusFound, "/")
}

func (p *poddy) oauthLogoutHandler(c *gin.Context) {
	provider := p.getProviderForId(c.Param("id"))
	if provider == nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	session := sessions.Default(c)
	RemoveTokenFromSession(session, provider)
	session.Save()

	c.Redirect(http.StatusFound, "/")
}

func (p *poddy) selfHandler(c *gin.Context) {
	session := sessions.Default(c)

	currentSessions := make([]interface{}, 0)

	for _, providerConfig := range p.oauthRepositoryProviderConfigs {
		tokenSource, err := ReadTokenFromSession(session, &providerConfig)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		if tokenSource == nil {
			continue
		}

		repositoryProvider, err := providerConfig.GetRepositoryProvider(tokenSource)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		selfUser, err := repositoryProvider.GetSelfUser()
		if err != nil {
			continue
		}

		currentSessions = append(currentSessions, map[string]interface{}{
			"host":     providerConfig.Host,
			"provider": providerConfig.ID,
			"user": map[string]interface{}{
				"username":     selfUser.GetUsername(),
				"display_name": selfUser.GetDisplayName(),
				"email":        selfUser.GetEmail(),
				"avatar_url":   selfUser.GetAvatarUrl(),
			},
		})
	}

	c.JSON(http.StatusOK, currentSessions)
}

type openWorkspaceBody struct {
	Host    string `json:"host" binding:"required"`
	Project string `json:"project" binding:"required"`
	Branch  string `json:"branch" binding:"required"`
}

func (p *poddy) openWorkspaceHandler(c *gin.Context) {
	var body openWorkspaceBody

	if err := c.ShouldBind(&body); err != nil {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("failed to bind request body: %v", err))
		return
	}

	repositoryProviderConfig := p.getProviderForHost(body.Host)
	if repositoryProviderConfig == nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	session := sessions.Default(c)
	tokenSource, err := ReadTokenFromSession(session, repositoryProviderConfig)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("failed to read token from session: %v", err))
		return
	}

	if tokenSource == nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	repositoryProvider, err := repositoryProviderConfig.GetRepositoryProvider(tokenSource)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("failed to get repository provider: %v", err))
		return
	}

	currentUser, err := repositoryProvider.GetSelfUser()
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	token, err := tokenSource.Token()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("failed to get access token: %v", err))
		return
	}

	workspaceName, workspaceUrl, err := createWorkspace(repositoryProvider, body.Project, body.Branch, currentUser, token.AccessToken)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("failed to create workspace: %v", err))
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"name": workspaceName,
		"url":  workspaceUrl,
	})
}

func (p *poddy) listWorkspacesHandler(c *gin.Context) {
	workspaces := make(map[string][]map[string]string)

	for _, providerConfig := range p.oauthRepositoryProviderConfigs {
		session := sessions.Default(c)
		tokenSource, err := ReadTokenFromSession(session, &providerConfig)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("failed to read token from session: %v", err))
			return
		}

		if tokenSource == nil {
			continue
		}

		repositoryProvider, err := providerConfig.GetRepositoryProvider(tokenSource)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("failed to get repository provider: %v", err))
			return
		}

		currentUser, err := repositoryProvider.GetSelfUser()
		if err != nil {
			continue
		}

		list, err := listWorkspaces(currentUser)
		if err != nil {
			continue
		}

		if len(list) > 0 {
			workspaces[providerConfig.ID] = list
		}
	}

	c.JSON(http.StatusOK, workspaces)
}

func (p *poddy) deleteWorkspaceHandler(c *gin.Context) {
	repositoryProviderConfig := p.getProviderForId(c.Param("provider"))
	if repositoryProviderConfig == nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	session := sessions.Default(c)
	tokenSource, err := ReadTokenFromSession(session, repositoryProviderConfig)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("failed to read token from session: %v", err))
		return
	}

	if tokenSource == nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	repositoryProvider, err := repositoryProviderConfig.GetRepositoryProvider(tokenSource)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("failed to get repository provider: %v", err))
		return
	}

	currentUser, err := repositoryProvider.GetSelfUser()
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	if err := deleteWorkspace(c.Param("name"), currentUser); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusNoContent)
}
