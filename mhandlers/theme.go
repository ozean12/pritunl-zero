package mhandlers

import (
	"github.com/dropbox/godropbox/container/set"
	"github.com/dropbox/godropbox/errors"
	"github.com/gin-gonic/gin"
	"github.com/ozean12/pritunl-zero/authorizer"
	"github.com/ozean12/pritunl-zero/database"
	"github.com/ozean12/pritunl-zero/demo"
	"github.com/ozean12/pritunl-zero/errortypes"
	"github.com/ozean12/pritunl-zero/utils"
)

type themeData struct {
	Theme string `json:"theme"`
}

func themePut(c *gin.Context) {
	if demo.IsDemo() {
		c.JSON(200, nil)
		return
	}

	db := c.MustGet("db").(*database.Database)
	authr := c.MustGet("authorizer").(*authorizer.Authorizer)
	data := &themeData{}

	err := c.Bind(&data)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "handler: Bind error"),
		}
		utils.AbortWithError(c, 500, err)
		return
	}

	usr, err := authr.GetUser(db)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	usr.Theme = data.Theme

	err = usr.CommitFields(db, set.NewSet("theme"))
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	c.JSON(200, data)
	return
}
