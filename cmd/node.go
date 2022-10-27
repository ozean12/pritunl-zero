package cmd

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dropbox/godropbox/errors"
	"github.com/ozean12/pritunl-zero/config"
	"github.com/ozean12/pritunl-zero/constants"
	"github.com/ozean12/pritunl-zero/errortypes"
	"github.com/ozean12/pritunl-zero/node"
	"github.com/ozean12/pritunl-zero/router"
	"github.com/ozean12/pritunl-zero/sync"
	"github.com/pritunl/mongo-go-driver/bson/primitive"
	"github.com/sirupsen/logrus"
)

func Node(testing bool) (err error) {
	objId, err := primitive.ObjectIDFromHex(config.Config.NodeId)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "cmd: Failed to parse ObjectId"),
		}
		return
	}

	nde := &node.Node{
		Id: objId,
	}
	err = nde.Init()
	if err != nil {
		return
	}

	sync.Init()

	routr := &router.Router{}

	routr.Init()

	go func() {
		err = routr.Run()
		if err != nil && !constants.Interrupt {
			panic(err)
		}
	}()

	if testing {
		time.Sleep(180 * time.Second)
	} else {
		sig := make(chan os.Signal, 2)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig
	}

	constants.Interrupt = true

	logrus.Info("cmd.node: Shutting down")
	go routr.Shutdown()
	if constants.Production {
		time.Sleep(300 * time.Millisecond)
	} else {
		time.Sleep(300 * time.Millisecond)
	}

	return
}
