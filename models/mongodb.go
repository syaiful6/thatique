package models

import (
	"context"

	"github.com/globalsign/mgo"
)

type MongoConn struct {
	DBName string
	Session *mgo.Session
	DB *mgo.Database // default db
}

// Session
func Dial(uri string, db string) (*MongoConn, error) {
	session, err := mgo.Dial(uri)
	if err != nil {
		return nil, err
	}

	conn := &MongoConn{
		DBName: db,
		Session: session,
		DB: session.DB(db),
	}

	return conn, err
}

//
func (conn *MongoConn) WithContext(ctx context.Context, f func(*mgo.Database) error) error {
	sess := conn.Session.Copy()
	defer sess.Close()

	c := make(chan error, 1)
	go func() { c <- f(sess.DB(conn.DBName)) }()

	select {
	case <-ctx.Done():
		<-c // Wait for f to return
		return ctx.Err()
	case err := <-c:
		return err 
	}
}