package cmd

import (
	"github.com/ozean12/pritunl-zero/database"
	"github.com/ozean12/pritunl-zero/log"
	"github.com/sirupsen/logrus"
)

func ClearLogs() (err error) {
	db := database.GetDatabase()
	defer db.Close()

	err = log.Clear(db)
	if err != nil {
		return
	}

	logrus.Info("cmd.log: Logs cleared")

	return
}
