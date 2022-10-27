package auth

import (
	"net/http"
	"time"

	"github.com/ozean12/pritunl-zero/database"
	"github.com/pritunl/mongo-go-driver/bson"
	"github.com/pritunl/mongo-go-driver/bson/primitive"
)

var (
	client = &http.Client{
		Timeout: 20 * time.Second,
	}
)

type authData struct {
	Url string `json:"url"`
}

type Token struct {
	Id        string             `bson:"_id"`
	Type      string             `bson:"type"`
	Secret    string             `bson:"secret"`
	Timestamp time.Time          `bson:"timestamp"`
	Provider  primitive.ObjectID `bson:"provider,omitempty"`
	Query     string             `bson:"query"`
}

func (t *Token) Remove(db *database.Database) (err error) {
	coll := db.Tokens()

	_, err = coll.DeleteOne(db, &bson.M{
		"_id": t.Id,
	})
	if err != nil {
		err = database.ParseError(err)
		if _, ok := err.(*database.NotFoundError); ok {
			err = nil
		} else {
			return
		}
	}

	return
}
