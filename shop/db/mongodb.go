package db

import (
	"context"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

type Model interface {
	CollectionName() string
}

type OrderedModel interface {
	Model
	SortBy() string
}

type Updatable interface {
	Model
	Unique() bson.M
	PreSave()
}

type MongoConn struct {
	DBName  string
	Session *mgo.Session
	DB      *mgo.Database // default db
}

// Session
func Dial(uri string, db string) (*MongoConn, error) {
	session, err := mgo.Dial(uri)
	if err != nil {
		return nil, err
	}

	conn := &MongoConn{
		DBName:  db,
		Session: session,
		DB:      session.DB(db),
	}

	return conn, err
}

func (conn *MongoConn) Copy() *MongoConn {
	sess := conn.Session.Copy()
	return &MongoConn{
		DBName:  conn.DBName,
		Session: sess,
		DB:      sess.DB(conn.DBName),
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
	u.PreSave()
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
