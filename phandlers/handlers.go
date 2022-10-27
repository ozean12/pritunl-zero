package phandlers

import (
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/ozean12/pritunl-zero/config"
	"github.com/ozean12/pritunl-zero/constants"
	"github.com/ozean12/pritunl-zero/middlewear"
	"github.com/ozean12/pritunl-zero/proxy"
	"github.com/ozean12/pritunl-zero/requires"
	"github.com/ozean12/pritunl-zero/service"
	"github.com/ozean12/pritunl-zero/static"
	"github.com/ozean12/pritunl-zero/utils"
)

var (
	index *static.File
	logo  *static.File
)

func Register(prxy *proxy.Proxy, engine *gin.Engine) {
	engine.Use(middlewear.Limiter)
	engine.Use(middlewear.Counter)
	engine.Use(middlewear.Recovery)
	engine.Use(middlewear.Headers)

	engine.Use(func(c *gin.Context) {
		var srvc *service.Service
		host, _ := prxy.MatchHost(utils.StripPort(c.Request.Host))
		if host != nil {
			srvc = host.Service
		}
		c.Set("service", srvc)
	})

	engine.NoRoute(redirect)

	dbGroup := engine.Group("")
	dbGroup.Use(middlewear.Database)

	sessGroup := dbGroup.Group("")
	sessGroup.Use(middlewear.SessionProxy)

	engine.GET("/auth/state", authStateGet)
	dbGroup.POST("/auth/session", authSessionPost)
	dbGroup.POST("/auth/secondary", authSecondaryPost)
	dbGroup.GET("/auth/request", authRequestGet)
	dbGroup.GET("/auth/callback", authCallbackGet)
	dbGroup.GET("/auth/webauthn/request", authWanRequestGet)
	dbGroup.POST("/auth/webauthn/respond", authWanRespondPost)
	dbGroup.GET("/auth/webauthn/register", authWanRegisterGet)
	dbGroup.POST("/auth/webauthn/register", authWanRegisterPost)
	sessGroup.GET("/logout", logoutGet)

	engine.GET("/check", checkGet)

	engine.GET("/", staticIndexGet)
	engine.GET("/login", staticIndexGet)
	engine.GET("/logo.png", staticLogoGet)
	engine.GET("/robots.txt", middlewear.RobotsGet)
}

func init() {
	module := requires.New("phandlers")
	module.After("settings")

	module.Handler = func() (err error) {
		root := ""
		if constants.Production {
			root = config.StaticRoot
		} else {
			root = config.StaticTestingRoot
		}

		index, err = static.NewFile(filepath.Join(root, "login.html"))
		if err != nil {
			return
		}

		logo, err = static.NewFile(filepath.Join(root, "logo.png"))
		if err != nil {
			return
		}

		return
	}
}
