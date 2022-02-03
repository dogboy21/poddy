package poddy

import (
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/dogboy21/poddy/config"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

type poddy struct {
	r                              *gin.Engine
	oauthRepositoryProviderConfigs []config.OauthRepositoryProviderConfig
}

func (p *poddy) getProviderForId(id string) *config.OauthRepositoryProviderConfig {
	for _, provider := range p.oauthRepositoryProviderConfigs {
		if provider.ID == id {
			return &provider
		}
	}

	return nil
}

func (p *poddy) getProviderForHost(host string) *config.OauthRepositoryProviderConfig {
	for _, provider := range p.oauthRepositoryProviderConfigs {
		if provider.Host == host {
			return &provider
		}
	}

	return nil
}

func Start() {
	rand.Seed(time.Now().UnixNano())

	err := config.ReadConfig()
	if err != nil {
		log.Fatalf("failed to read config: %v\n", err)
	}

	oauthRepositoryProviderConfigs, err := config.GetOauthConfigs()
	if err != nil {
		log.Fatalf("failed to read oauth repository provider configs: %v\n", err)
	}

	app := poddy{
		r:                              gin.New(),
		oauthRepositoryProviderConfigs: oauthRepositoryProviderConfigs,
	}

	sessionStore := cookie.NewStore(config.ServerCookieSecret())
	sessionStore.Options(sessions.Options{
		Domain:   config.ServerUrl().Host,
		Secure:   config.ServerSecureCookies(),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   24 * 60 * 60,
		Path:     "/",
	})

	app.r.Use(
		gin.Logger(), gin.Recovery(),
		sessions.Sessions("poddy", sessionStore),
	)

	app.r.GET("/oauth/providers", app.listOauthProvidersHandler)
	app.r.GET("/oauth/auth/:id", app.oauthAuthHandler)
	app.r.GET("/oauth/redirect/:id", app.oauthRedirectHandler)
	app.r.GET("/oauth/logout/:id", app.oauthLogoutHandler)

	app.r.GET("/api/v1/self", app.selfHandler)

	app.r.POST("/api/v1/workspaces", app.openWorkspaceHandler)
	app.r.GET("/api/v1/workspaces", app.listWorkspacesHandler)
	app.r.DELETE("/api/v1/workspaces/:provider/:name", app.deleteWorkspaceHandler)

	app.r.Static("/assets", "./frontend/dist/assets")
	app.r.StaticFile("/", "./frontend/dist/index.html")
	app.r.StaticFile("/favicon.ico", "./frontend/dist/favicon.ico")

	if err := app.r.Run(config.ServerListenAddress()); err != nil {
		log.Fatalf("failed to start webserver: %v\n", err)
	}
}
