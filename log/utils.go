package log

import (
	"github.com/ozean12/pritunl-zero/database"
	"github.com/ozean12/pritunl-zero/event"
	"github.com/ozean12/pritunl-zero/utils"
	"github.com/pritunl/mongo-go-driver/bson"
	"github.com/pritunl/mongo-go-driver/bson/primitive"
	"github.com/pritunl/mongo-go-driver/mongo/options"
)

func Get(db *database.Database, logId primitive.ObjectID) (
	entry *Entry, err error) {

	coll := db.Logs()
	entry = &Entry{}

	err = coll.FindOneId(logId, entry)
	if err != nil {
		return
	}

	return
}

func GetAll(db *database.Database, query *bson.M, page, pageCount int64) (
	entries []*Entry, count int64, err error) {

	coll := db.Logs()
	entries = []*Entry{}

	if len(*query) == 0 {
		count, err = coll.EstimatedDocumentCount(db)
		if err != nil {
			err = database.ParseError(err)
			return
		}
	} else {
		count, err = coll.CountDocuments(db, query)
		if err != nil {
			err = database.ParseError(err)
			return
		}
	}

	opts := options.FindOptions{
		Sort: &bson.D{
			{"$natural", -1},
		},
	}

	if pageCount != 0 {
		maxPage := count / pageCount
		if count == pageCount {
			maxPage = 0
		}
		page = utils.Min64(page, maxPage)
		skip := utils.Min64(page*pageCount, count)
		opts.Skip = &skip
		opts.Limit = &pageCount
	}

	cursor, err := coll.Find(
		db,
		query,
		&opts,
	)
	if err != nil {
		err = database.ParseError(err)
		return
	}
	defer cursor.Close(db)

	for cursor.Next(db) {
		entry := &Entry{}
		err = cursor.Decode(entry)
		if err != nil {
			err = database.ParseError(err)
			return
		}

		entries = append(entries, entry)
	}

	err = cursor.Err()
	if err != nil {
		err = database.ParseError(err)
		return
	}

	return
}

func Clear(db *database.Database) (err error) {
	coll := db.Logs()

	_, err = coll.DeleteMany(db, nil)
	if err != nil {
		err = database.ParseError(err)
		return
	}

	_ = event.PublishDispatch(db, "log.change")

	return
}
