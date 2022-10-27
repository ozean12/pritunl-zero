package sync

import (
	"time"

	"github.com/ozean12/pritunl-zero/database"
	"github.com/ozean12/pritunl-zero/settings"
	"github.com/ozean12/pritunl-zero/user"
	"github.com/pritunl/mongo-go-driver/bson"
	"github.com/pritunl/mongo-go-driver/mongo/options"
	"github.com/sirupsen/logrus"
)

func authSync() (err error) {
	db := database.GetDatabase()
	defer db.Close()

	coll := db.Users()
	opts := &options.CountOptions{}
	opts.SetLimit(1)

	count, err := coll.CountDocuments(
		db,
		&bson.M{
			"type": user.Local,
		},
		opts,
	)
	if err != nil {
		err = database.ParseError(err)
		return
	}

	settings.Local.NoLocalAuth = count == 0

	return
}

func authRunner() {
	time.Sleep(1 * time.Second)

	for {
		time.Sleep(10 * time.Second)

		err := authSync()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Error("sync: Failed to sync authentication status")
		}
	}
}

func initAuth() {
	go authRunner()
}
