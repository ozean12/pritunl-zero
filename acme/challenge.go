package acme

import (
	"time"

	"github.com/ozean12/pritunl-zero/database"
	"github.com/pritunl/mongo-go-driver/bson"
)

type Challenge struct {
	Id        string    `bson:"_id"`
	Resource  string    `bson:"resource"`
	Timestamp time.Time `bson:"timestamp"`
}

func (c *Challenge) Insert(db *database.Database) (err error) {
	coll := db.AcmeChallenges()

	_, err = coll.InsertOne(db, c)
	if err != nil {
		err = database.ParseError(err)
		return
	}

	return
}

func (c *Challenge) Remove(db *database.Database) (err error) {
	coll := db.AcmeChallenges()

	_, err = coll.DeleteOne(db, &bson.M{
		"_id": c.Id,
	})
	if err != nil {
		err = database.ParseError(err)
		switch err.(type) {
		case *database.NotFoundError:
			err = nil
		default:
			return
		}
	}

	return
}
