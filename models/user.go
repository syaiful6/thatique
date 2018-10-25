package models

import (
	"time"

	"github.com/globalsign/mgo/bson"
)

type Profile struct {
	Name		string		`bson:"name,omitempty" json:"name,omitempty"`
	Picture		string		`bson:"picture,omitempty" json:"picture"`
	Bio			string		`bson:"bio,omitempty" json:"bio"`
	Web			string		`bson:"web,omitempty" json:"web,omitempty"`
}

type User struct {
	Id			bson.ObjectId			`bson:"_id,omitempty" json:"id,omitempty"`
	Profile		Profile					`bson:"profile" json:"profile"`
	Email		string					`bson:"email" json:"email"`
	Password	string					`bson:"password" json:"password"`
	Superuser	bool					`bson:"is_superuser" json:"is_superuser"`
	Staff		bool					`bson:"is_staff" json:"is_staff"`
	CreatedAt	time.Time				`bson:"created_at" json:"created_at"`
}

type SerializeUser struct {
	Id			string		`json:"id"`
	Profile		Profile		`json:"profile"`
	Email		string		`json:"email"`
	Superuser	bool		`json:"is_superuser"`
	Staff		bool		`json:"is_staff"`
	CreatedAt	time.Time	`json:"created_at"`
}