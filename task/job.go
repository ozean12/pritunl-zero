package task

import (
	"time"

	"github.com/ozean12/pritunl-zero/database"
	"github.com/pritunl/mongo-go-driver/bson"
	"github.com/pritunl/mongo-go-driver/bson/primitive"
)

type Job struct {
	Id        string             `bson:"_id"`
	Name      string             `bson:"name"`
	State     string             `bson:"state"`
	Retry     bool               `bson:"retry"`
	Node      primitive.ObjectID `bson:"node"`
	Timestamp time.Time          `bson:"timestamp"`
}

func (j *Job) Reserve(db *database.Database) (reserved bool, err error) {
	coll := db.Tasks()

	_, err = coll.InsertOne(db, j)
	if err != nil {
		err = database.ParseError(err)

		switch err.(type) {
		case *database.DuplicateKeyError:
			err = nil
			break
		}

		return
	}

	reserved = true
	return
}

func (j *Job) Failed(db *database.Database) (err error) {
	coll := db.Tasks()

	err = coll.UpdateId(j.Id, &bson.M{
		"$set": &bson.M{
			"state": Failed,
		},
	})

	return
}

func (j *Job) Finished(db *database.Database) (err error) {
	coll := db.Tasks()

	err = coll.UpdateId(j.Id, &bson.M{
		"$set": &bson.M{
			"state": Finished,
		},
	})

	return
}
