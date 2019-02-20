package db

import (
	"fmt"
	"context"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/syaiful6/thatique/pkg/text"
)

var (
	Conn   = new(MongoConn)
	models = []Model{}
)

type Model interface {
	CollectionName() string
	Indexes() []mgo.Index
}

type Slugable interface {
	Model
	// This is used to generate slug
	SlugQuery(slug string) bson.M
}

type OrderedModel interface {
	Model
	SortBy() string
}

type Updatable interface {
	Model
	Unique() bson.M
	Presave()
}

type MongoConn struct {
	Session *mgo.Session
	DB      *mgo.Database // default db
}

func Connect(uri string, db string) error {
	conn, err := dial(uri, db)
	if err != nil {
		return err
	}

	Conn = conn

	for _, m := range models {
		err = registerIndexes(m)
		if err != nil {
			return err
		}
	}

	return nil
}

func Register(m Model) {
	models = append(models, m)
}

func registerIndexes(m Model) error {
	collection := Conn.DB.C(m.CollectionName())
	indexes := m.Indexes()
	for _, index := range indexes {
		err := collection.EnsureIndex(index)
		if err != nil {
			return err
		}
	}
	return nil
}

func GenerateSlug(m Slugable, base) (string, error) {
	var (
		slugTryCount = 0
		slug 		 = text.Slugify(base)
		collection   = Conn.DB.C(m.CollectionName())
		maxretries   = 20
		retries int
		count int
		err error
	)
	slugToTry := slug
	for {
		count, err = collection.Find(m.SlugQuery(slugToTry)).Count()
		if count == 0 {
			return slugToTry, nil
		}
		retries += 1
		if retries > maxretries {
			return "", fmt.Errorf("generateslug: maximum retries reached. max: %d", maxretries)
		}
		slugToTry = fmt.Sprintf("%s-%d", slug, retries)
	}
}

// Session
func dial(uri string, db string) (*MongoConn, error) {
	session, err := mgo.Dial(uri)
	if err != nil {
		return nil, err
	}

	conn := &MongoConn{
		Session: session,
		DB:      session.DB(db),
	}

	return conn, err
}

func (conn *MongoConn) Copy() *MongoConn {
	sess := conn.Session.Copy()
	return &MongoConn{
		Session: sess,
		DB:      sess.DB(conn.DB.Name),
	}
}

func (conn *MongoConn) Close() {
	conn.Session.Close()
}

//
func (conn *MongoConn) Cursor(m Model) *mgo.Collection {
	return conn.DB.C(m.CollectionName())
}

func (conn *MongoConn) Find(m Model, query interface{}) *mgo.Query {
	return conn.Cursor(m).Find(query)
}

func (conn *MongoConn) Latest(ord OrderedModel, query interface{}) *mgo.Query {
	return conn.Find(ord.(Model), query).Sort(ord.SortBy())
}

func (conn *MongoConn) Exists(u Updatable) bool {
	var data interface{}
	err := conn.Cursor(u.(Model)).Find(u.Unique()).One(&data)
	if err != nil {
		return false
	}
	return true
}

func (conn *MongoConn) Upsert(u Updatable) (info *mgo.ChangeInfo, err error) {
	u.Presave()
	return conn.Cursor(u.(Model)).Upsert(u.Unique(), u)
}

//
func (conn *MongoConn) WithContext(ctx context.Context, f func(*MongoConn) error) error {
	sess := conn.Copy()
	defer sess.Close()

	c := make(chan error, 1)
	go func() {
		c <- f(sess)
	}()

	select {
	case <-ctx.Done():
		<-c // Wait for f to return
		return ctx.Err()
	case err := <-c:
		return err
	}
}
