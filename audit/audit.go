package audit

import (
	"time"

	"github.com/dropbox/godropbox/errors"
	"github.com/ozean12/pritunl-zero/agent"
	"github.com/ozean12/pritunl-zero/database"
	"github.com/ozean12/pritunl-zero/errortypes"
	"github.com/pritunl/mongo-go-driver/bson/primitive"
)

type Fields map[string]interface{}

type Audit struct {
	Id        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	User      primitive.ObjectID `bson:"u" json:"user"`
	Timestamp time.Time          `bson:"t" json:"timestamp"`
	Type      string             `bson:"y" json:"type"`
	Fields    Fields             `bson:"f" json:"fields"`
	Agent     *agent.Agent       `bson:"a" json:"agent"`
}

func (a *Audit) Insert(db *database.Database) (err error) {
	coll := db.Audits()

	if !a.Id.IsZero() {
		err = &errortypes.DatabaseError{
			errors.New("audit: Entry already exists"),
		}
		return
	}

	_, err = coll.InsertOne(db, a)
	if err != nil {
		err = database.ParseError(err)
		return
	}

	return
}
