package uhandlers

import (
	"regexp"
	"time"

	"github.com/dropbox/godropbox/errors"
	"github.com/gin-gonic/gin"
	"github.com/ozean12/pritunl-zero/audit"
	"github.com/ozean12/pritunl-zero/authorizer"
	"github.com/ozean12/pritunl-zero/challenge"
	"github.com/ozean12/pritunl-zero/database"
	"github.com/ozean12/pritunl-zero/device"
	"github.com/ozean12/pritunl-zero/errortypes"
	"github.com/ozean12/pritunl-zero/event"
	"github.com/ozean12/pritunl-zero/secondary"
	"github.com/ozean12/pritunl-zero/ssh"
	"github.com/ozean12/pritunl-zero/utils"
	"github.com/ozean12/pritunl-zero/validator"
)

var (
	domainRe = regexp.MustCompile(`[^a-zA-Z0-9-_.]+`)
)

type sshValidateData struct {
	Token     string `json:"token"`
	PublicKey string `json:"public_key,omitempty"`
}

type sshCertificateData struct {
	Token                  string      `json:"token"`
	Certificates           []string    `json:"certificates"`
	CertificateAuthorities []string    `json:"certificate_authorities"`
	Hosts                  []*ssh.Host `json:"hosts"`
}

func sshGet(c *gin.Context) {
	authr := c.MustGet("authorizer").(*authorizer.Authorizer)

	redirect := ""

	if authr.IsValid() {
		if c.Request.URL.RawQuery == "" {
			redirect = "/"
		} else {
			query := c.Request.URL.Query()
			redirect = "/?" + query.Encode()
		}
	} else {
		if c.Request.URL.RawQuery == "" {
			redirect = "/login"
		} else {
			query := c.Request.URL.Query()
			redirect = "/login?" + query.Encode()
		}
	}

	c.Redirect(302, redirect)
}

func sshValidatePut(c *gin.Context) {
	db := c.MustGet("db").(*database.Database)
	authr := c.MustGet("authorizer").(*authorizer.Authorizer)

	sshToken := c.Param("ssh_token")

	usr, err := authr.GetUser(db)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	chal, err := challenge.GetChallenge(db, sshToken)
	if err != nil {
		switch err.(type) {
		case *database.NotFoundError:
			utils.AbortWithStatus(c, 404)
			break
		default:
			utils.AbortWithError(c, 500, err)
		}
		return
	}

	deviceAuth, secProviderId, err, errData := chal.Approve(
		db, usr, c.Request, false, false)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	if errData != nil {
		c.JSON(400, errData)
		return
	}

	if deviceAuth {
		deviceCount, err := device.CountSecondary(db, usr.Id)
		if err != nil {
			utils.AbortWithError(c, 500, err)
			return
		}

		if deviceCount == 0 {
			errData := &errortypes.ErrorData{
				Error:   "secondary_device_unavailable",
				Message: "Secondary authentication device not available",
			}
			c.JSON(400, errData)
			return
		}

		secd, err := secondary.NewChallenge(db, usr.Id,
			secondary.AuthorityDevice, chal.Id, secondary.DeviceProvider)
		if err != nil {
			utils.AbortWithError(c, 500, err)
			return
		}

		data, err := secd.GetData()
		if err != nil {
			utils.AbortWithError(c, 500, err)
			return
		}

		c.JSON(201, data)
		return
	} else if !secProviderId.IsZero() {
		secd, err := secondary.NewChallenge(
			db, usr.Id, secondary.Authority, chal.Id, secProviderId)
		if err != nil {
			utils.AbortWithError(c, 500, err)
			return
		}

		data, err := secd.GetData()
		if err != nil {
			utils.AbortWithError(c, 500, err)
			return
		}

		c.JSON(201, data)
		return
	}

	err = audit.New(
		db,
		c.Request,
		usr.Id,
		audit.SshApprove,
		audit.Fields{
			"ssh_key": chal.PubKey,
		},
	)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	_ = event.Publish(db, "ssh_challenge", chal.Id)

	c.Status(200)
}

func sshValidateDelete(c *gin.Context) {
	db := c.MustGet("db").(*database.Database)
	authr := c.MustGet("authorizer").(*authorizer.Authorizer)

	sshToken := c.Param("ssh_token")

	usr, err := authr.GetUser(db)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	chal, err := challenge.GetChallenge(db, sshToken)
	if err != nil {
		switch err.(type) {
		case *database.NotFoundError:
			utils.AbortWithStatus(c, 404)
			break
		default:
			utils.AbortWithError(c, 500, err)
		}
		return
	}

	err = audit.New(
		db,
		c.Request,
		usr.Id,
		audit.SshDeny,
		audit.Fields{
			"ssh_key": chal.PubKey,
		},
	)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	err = chal.Deny(db, usr)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	_ = event.Publish(db, "ssh_challenge", chal.Id)

	c.Status(200)
}

type sshSecondaryData struct {
	Token    string `json:"token"`
	Factor   string `json:"factor"`
	Passcode string `json:"passcode"`
}

func sshSecondaryPut(c *gin.Context) {
	db := c.MustGet("db").(*database.Database)
	authr := c.MustGet("authorizer").(*authorizer.Authorizer)
	data := &sshSecondaryData{}

	err := c.Bind(data)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "handler: Bind error"),
		}
		utils.AbortWithError(c, 500, err)
		return
	}

	secd, err := secondary.Get(db, data.Token, secondary.Authority)
	if err != nil {
		if _, ok := err.(*database.NotFoundError); ok {
			errData := &errortypes.ErrorData{
				Error:   "secondary_expired",
				Message: "Secondary authentication has expired",
			}
			c.JSON(400, errData)
		} else {
			utils.AbortWithError(c, 500, err)
		}
		return
	}

	errData, err := secd.Handle(db, c.Request, data.Factor, data.Passcode)
	if err != nil {
		if _, ok := err.(*secondary.IncompleteError); ok {
			c.Status(206)
		} else {
			utils.AbortWithError(c, 500, err)
		}
		return
	}

	if errData != nil {
		c.JSON(400, errData)
		return
	}

	usr, err := authr.GetUser(db)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	chal, err := challenge.GetChallenge(db, secd.ChallengeId)
	if err != nil {
		switch err.(type) {
		case *database.NotFoundError:
			utils.AbortWithStatus(c, 404)
			break
		default:
			utils.AbortWithError(c, 500, err)
		}
		return
	}

	_, _, err, errData = chal.Approve(db, usr, c.Request, true, true)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	if errData != nil {
		c.JSON(400, errData)
		return
	}

	err = audit.New(
		db,
		c.Request,
		usr.Id,
		audit.SshApprove,
		audit.Fields{
			"ssh_key": chal.PubKey,
		},
	)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	_ = event.Publish(db, "ssh_challenge", chal.Id)

	c.Status(200)
}

func sshWanRequestGet(c *gin.Context) {
	db := c.MustGet("db").(*database.Database)
	token := c.Query("token")

	secd, err := secondary.Get(db, token, secondary.AuthorityDevice)
	if err != nil {
		if _, ok := err.(*database.NotFoundError); ok {
			errData := &errortypes.ErrorData{
				Error:   "secondary_expired",
				Message: "Secondary authentication has expired",
			}
			c.JSON(400, errData)
		} else {
			utils.AbortWithError(c, 500, err)
		}
		return
	}

	resp, errData, err := secd.DeviceRequest(
		db, utils.GetOrigin(c.Request))
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	if errData != nil {
		c.JSON(400, errData)
		return
	}

	c.JSON(200, resp)
}

type sshWanRespondData struct {
	Token string `json:"token"`
}

func sshWanRespondPost(c *gin.Context) {
	db := c.MustGet("db").(*database.Database)
	authr := c.MustGet("authorizer").(*authorizer.Authorizer)
	data := &sshWanRespondData{}

	body, err := utils.CopyBody(c.Request)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	err = c.Bind(data)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "handler: Bind error"),
		}
		utils.AbortWithError(c, 500, err)
		return
	}

	secd, err := secondary.Get(db, data.Token, secondary.AuthorityDevice)
	if err != nil {
		if _, ok := err.(*database.NotFoundError); ok {
			errData := &errortypes.ErrorData{
				Error:   "secondary_expired",
				Message: "Secondary authentication has expired",
			}
			c.JSON(400, errData)
		} else {
			utils.AbortWithError(c, 500, err)
		}
		return
	}

	usr, err := authr.GetUser(db)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	_, secProviderId, errAudit, errData, err := validator.ValidateUser(
		db, usr, false, c.Request)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	if errData != nil {
		c.JSON(400, errData)
		return
	}

	errData, err = secd.DeviceRespond(
		db, utils.GetOrigin(c.Request), body)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	if errData != nil {
		if errAudit == nil {
			errAudit = audit.Fields{
				"error":   errData.Error,
				"message": errData.Message,
			}
		}
		errAudit["method"] = "add_device_register"

		err = audit.New(
			db,
			c.Request,
			usr.Id,
			audit.UserAuthFailed,
			errAudit,
		)
		if err != nil {
			utils.AbortWithError(c, 500, err)
			return
		}

		c.JSON(400, errData)
		return
	}

	chal, err := challenge.GetChallenge(db, secd.ChallengeId)
	if err != nil {
		switch err.(type) {
		case *database.NotFoundError:
			utils.AbortWithStatus(c, 404)
			break
		default:
			utils.AbortWithError(c, 500, err)
		}
		return
	}

	_, secProviderId, err, errData = chal.Approve(
		db, usr, c.Request, true, false)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	if errData != nil {
		c.JSON(400, errData)
		return
	}

	if !secProviderId.IsZero() {
		secd, err := secondary.NewChallenge(db, usr.Id,
			secondary.Authority, chal.Id, secProviderId)
		if err != nil {
			utils.AbortWithError(c, 500, err)
			return
		}

		data, err := secd.GetData()
		if err != nil {
			utils.AbortWithError(c, 500, err)
			return
		}

		c.JSON(201, data)
		return
	}

	err = audit.New(
		db,
		c.Request,
		usr.Id,
		audit.SshApprove,
		audit.Fields{
			"ssh_key": chal.PubKey,
		},
	)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	_ = event.Publish(db, "ssh_challenge", chal.Id)

	c.Status(200)
}

func sshChallengePut(c *gin.Context) {
	db := c.MustGet("db").(*database.Database)
	data := &sshValidateData{}

	err := c.Bind(data)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "handler: Bind error"),
		}
		utils.AbortWithError(c, 500, err)
		return
	}

	chal, err := challenge.GetChallenge(db, data.Token)
	if err != nil {
		switch err.(type) {
		case *database.NotFoundError:
			utils.AbortWithStatus(c, 404)
			break
		default:
			utils.AbortWithError(c, 500, err)
		}
		return
	}
	token := chal.Id

	sync := func() {
		chal, err = challenge.GetChallenge(db, data.Token)
		if err != nil {
			switch err.(type) {
			case *database.NotFoundError:
				utils.AbortWithStatus(c, 404)
				break
			default:
				utils.AbortWithError(c, 500, err)
			}
			return
		}
	}

	update := func() bool {
		switch chal.State {
		case ssh.Approved:
			cert, err := ssh.GetCertificate(db, chal.CertificateId)
			if err != nil {
				switch err.(type) {
				case *database.NotFoundError:
					utils.AbortWithStatus(c, 404)
					break
				default:
					utils.AbortWithError(c, 500, err)
				}
				return true
			}

			resp := &sshCertificateData{
				Token:                  token,
				Hosts:                  cert.Hosts,
				Certificates:           cert.Certificates,
				CertificateAuthorities: cert.CertificateAuthorities,
			}

			c.JSON(200, resp)

			return true
		case ssh.Unavailable:
			errData := &errortypes.ErrorData{
				Error: "certificate_unavailable",
				Message: "Cerification was approved but no " +
					"certificates are available",
			}
			c.JSON(412, errData)
			return true
		case ssh.Denied:
			c.Status(401)
			return true
		}

		return false
	}

	if update() {
		return
	}

	start := time.Now()
	ticker := time.NewTicker(3 * time.Second)
	notify := make(chan bool, 3)

	listenerId := challenge.Register(token, func() {
		defer func() {
			recover()
		}()
		notify <- true
	})
	defer challenge.Unregister(token, listenerId)

	for {
		select {
		case <-ticker.C:
			if time.Since(start) > 29*time.Second {
				c.Status(205)
				return
			}

			sync()
			if update() {
				return
			}
		case <-notify:
			sync()
			if update() {
				return
			}
		}
	}
}

func sshChallengePost(c *gin.Context) {
	db := c.MustGet("db").(*database.Database)
	data := &sshValidateData{}

	err := c.Bind(data)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "handler: Bind error"),
		}
		utils.AbortWithError(c, 500, err)
		return
	}

	chal, err := challenge.NewChallenge(db, data.PublicKey)
	if err != nil {
		switch err.(type) {
		case *database.NotFoundError:
			utils.AbortWithStatus(c, 404)
			break
		default:
			utils.AbortWithError(c, 500, err)
		}
		return
	}

	resp := &sshValidateData{
		Token: chal.Id,
	}

	c.JSON(200, resp)
}

type sshHostData struct {
	Hostname  string   `json:"hostname"`
	Port      int      `json:"port"`
	Tokens    []string `json:"tokens"`
	PublicKey string   `json:"public_key"`
}

type sshHostCertificateData struct {
	Certificates []string `json:"certificates"`
}

func sshHostPost(c *gin.Context) {
	db := c.MustGet("db").(*database.Database)
	data := &sshHostData{}

	err := c.Bind(data)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "handler: Bind error"),
		}
		utils.AbortWithError(c, 500, err)
		return
	}

	hostname := domainRe.ReplaceAllString(data.Hostname, "")

	cert, errData, err := ssh.NewHostCertificate(db, hostname,
		data.Port, data.Tokens, c.Request, data.PublicKey)
	if err != nil {
		switch err.(type) {
		case *database.NotFoundError:
			utils.AbortWithStatus(c, 404)
			break
		default:
			utils.AbortWithError(c, 500, err)
		}
		return
	}

	if errData != nil {
		c.JSON(400, errData)
		return
	}

	resp := &sshHostCertificateData{
		Certificates: cert.Certificates,
	}

	c.JSON(200, resp)
}
