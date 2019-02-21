package mongostore

import (
	"github.com/globalsign/mgo/bson"
	"github.com/syaiful6/thatique/shop/auth"
	"github.com/syaiful6/thatique/shop/db"
)

type MongoStore struct {
	c *db.MongoConn
}

func NewMongoStore(conn *db.MongoConn) *MongoStore {
	return &MongoStore{c: conn}
}

func (s *MongoStore) FindUserById(id bson.ObjectId) (user *auth.User, err error) {
	err = s.c.C(user).FindId(id).One(&user)
	return
}

func (s *MongoStore) FindUserByEmail(email string) (user *auth.User, err error) {
	err = s.c.C(user).Find(bson.M{"email": email}).One(&user)
	return
}
