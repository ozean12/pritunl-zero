package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/dropbox/godropbox/errors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/ozean12/pritunl-zero/database"
	"github.com/ozean12/pritunl-zero/demo"
	"github.com/ozean12/pritunl-zero/endpoint"
	"github.com/ozean12/pritunl-zero/errortypes"
	"github.com/ozean12/pritunl-zero/event"
	"github.com/ozean12/pritunl-zero/utils"
	"github.com/sirupsen/logrus"
)

const (
	endpointWriteTimeout = 10 * time.Second
	endpointPingInterval = 20 * time.Second
	endpointPingWait     = 40 * time.Second
)

func EndpointRegisterPut(c *gin.Context) {
	if demo.Blocked(c) {
		return
	}

	db := c.MustGet("db").(*database.Database)
	data := &endpoint.RegisterData{}

	endpointId, ok := utils.ParseObjectId(c.Param("endpoint_id"))
	if !ok {
		utils.AbortWithStatus(c, 400)
		return
	}

	err := c.Bind(data)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "handler: Bind error"),
		}
		utils.AbortWithError(c, 500, err)
		return
	}

	endpt, err := endpoint.Get(db, endpointId)
	if err != nil {
		if _, ok := err.(*database.NotFoundError); ok {
			utils.AbortWithError(c, 404, err)
		} else {
			utils.AbortWithError(c, 500, err)
		}
		return
	}

	resData, errData, err := endpt.Register(db, data)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}
	if errData != nil {
		c.JSON(400, errData)
		return
	}

	_ = event.PublishDispatch(db, "endpoint.change")

	c.JSON(200, resData)
}

func EndpointCommGet(c *gin.Context) {
	db := c.MustGet("db").(*database.Database)
	socket := &endpoint.WebSocket{}

	endpointId, ok := utils.ParseObjectId(c.Param("endpoint_id"))
	if !ok {
		utils.AbortWithStatus(c, 400)
		return
	}

	timestamp := c.Request.Header.Get("Pritunl-Endpoint-Timestamp")
	nonce := c.Request.Header.Get("Pritunl-Endpoint-Nonce")
	sig := c.Request.Header.Get("Pritunl-Endpoint-Signature")
	endptUpdate := time.Now()

	endpt, err := endpoint.Get(db, endpointId)
	if err != nil {
		if _, ok := err.(*database.NotFoundError); ok {
			utils.AbortWithError(c, 404, err)
		} else {
			utils.AbortWithError(c, 500, err)
		}
		return
	}

	errData, err := endpt.ValidateSignature(
		db, timestamp, nonce, sig, "communicate")
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}
	if errData != nil {
		c.JSON(401, errData)
		return
	}

	defer func() {
		socket.Close()
		endpoint.WebSocketsLock.Lock()
		endpoint.WebSockets.Remove(socket)
		endpoint.WebSocketsLock.Unlock()
	}()

	endpoint.WebSocketsLock.Lock()
	endpoint.WebSockets.Add(socket)
	endpoint.WebSocketsLock.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	socket.Cancel = cancel

	conn, err := endpoint.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "mhandlers: Failed to upgrade request"),
		}
		utils.AbortWithError(c, 500, err)
		return
	}
	socket.Conn = conn

	err = conn.SetReadDeadline(time.Now().Add(endpointPingWait))
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "mhandlers: Failed to set read deadline"),
		}
		utils.AbortWithError(c, 500, err)
		return
	}

	conn.SetPongHandler(func(x string) (err error) {
		err = conn.SetReadDeadline(time.Now().Add(endpointPingWait))
		if err != nil {
			err = &errortypes.RequestError{
				errors.Wrap(err, "mhandlers: Failed to set read deadline"),
			}
			utils.AbortWithError(c, 500, err)
			return
		}

		return
	})

	ticker := time.NewTicker(endpointPingInterval)
	socket.Ticker = ticker

	go func() {
		defer func() {
			r := recover()
			if r != nil && !socket.Closed {
				logrus.WithFields(logrus.Fields{
					"error": errors.New(fmt.Sprintf("%s", r)),
				}).Error("mhandlers: Endpoint comm panic")
			}
		}()
		for {
			msgType, msgByte, err := conn.ReadMessage()
			if err != nil {
				_ = conn.Close()
				return
			}

			if msgType != websocket.TextMessage {
				continue
			}

			err = endpt.InsertDoc(db, msgByte)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"error": err,
				}).Error("mhandlers: Failed to insert doc")

				_ = conn.Close()
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err = conn.WriteControl(websocket.PingMessage, []byte{},
				time.Now().Add(endpointWriteTimeout))
			if err != nil {
				err = &errortypes.RequestError{
					errors.Wrap(err,
						"mhandlers: Failed to set write control"),
				}
				_ = conn.Close()
				return
			}

			if time.Since(endptUpdate) > 1*time.Minute {
				newEndpt, e := endpoint.Get(db, endpointId)
				if e != nil {
					logrus.WithFields(logrus.Fields{
						"error": e,
					}).Error("mhandlers: Failed to update endpoint")

					_ = conn.Close()
					return
				}

				endpt = newEndpt
				endptUpdate = time.Now()

				encConf, e := endpt.GetConf(db)
				if e != nil {
					logrus.WithFields(logrus.Fields{
						"error": e,
					}).Error("mhandlers: Failed to update endpoint conf")

					_ = conn.Close()
					return
				}

				err = conn.SetWriteDeadline(
					time.Now().Add(endpointWriteTimeout))
				if err != nil {
					_ = conn.Close()
					return
				}

				if encConf != nil {
					err = conn.WriteMessage(websocket.TextMessage, encConf)
					if err != nil {
						err = &errortypes.RequestError{
							errors.Wrap(err,
								"mhandlers: Failed to write endpoint conf"),
						}
						_ = conn.Close()
						return
					}
				}
			}
		}
	}
}
