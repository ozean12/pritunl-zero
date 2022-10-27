package mhandlers

import (
	"strings"

	"github.com/dropbox/godropbox/container/set"
	"github.com/dropbox/godropbox/errors"
	"github.com/gin-gonic/gin"
	"github.com/ozean12/pritunl-zero/database"
	"github.com/ozean12/pritunl-zero/demo"
	"github.com/ozean12/pritunl-zero/errortypes"
	"github.com/ozean12/pritunl-zero/event"
	"github.com/ozean12/pritunl-zero/settings"
	"github.com/ozean12/pritunl-zero/subscription"
	"github.com/ozean12/pritunl-zero/utils"
)

type subscriptionPostData struct {
	License string `json:"license"`
}

func subscriptionGet(c *gin.Context) {
	if demo.IsDemo() {
		c.JSON(200, demo.Subscription)
		return
	}
	c.JSON(200, subscription.Sub)
}

func subscriptionUpdateGet(c *gin.Context) {
	if demo.IsDemo() {
		c.JSON(200, demo.Subscription)
		return
	}

	errData, err := subscription.Update()
	if err != nil {
		if errData != nil {
			c.JSON(400, errData)
		} else {
			utils.AbortWithError(c, 500, err)
		}
		return
	}

	c.JSON(200, subscription.Sub)
}

func subscriptionPost(c *gin.Context) {
	if demo.Blocked(c) {
		return
	}

	db := c.MustGet("db").(*database.Database)
	data := &subscriptionPostData{}

	err := c.Bind(data)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "handler: Bind error"),
		}
		utils.AbortWithError(c, 500, err)
		return
	}

	license := strings.TrimSpace(data.License)
	license = strings.Replace(license, "BEGIN LICENSE", "", 1)
	license = strings.Replace(license, "END LICENSE", "", 1)
	license = strings.Replace(license, "-", "", -1)
	license = strings.Replace(license, " ", "", -1)
	license = strings.Replace(license, "\n", "", -1)

	settings.System.License = license

	errData, err := subscription.Update()
	if err != nil {
		settings.System.License = ""
		if errData != nil {
			c.JSON(400, errData)
		} else {
			utils.AbortWithError(c, 500, err)
		}
		return
	}

	err = settings.Commit(db, settings.System, set.NewSet(
		"license",
	))
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	_ = event.PublishDispatch(db, "subscription.change")
	_ = event.PublishDispatch(db, "settings.change")

	c.JSON(200, subscription.Sub)
}
