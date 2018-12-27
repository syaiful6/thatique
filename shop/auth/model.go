package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/globalsign/mgo/bson"
	"golang.org/x/crypto/bcrypt"
)

type Profile struct {
	Name    string `bson:"name,omitempty" json:"name,omitempty"`
	Picture string `bson:"picture,omitempty" json:"picture"`
	Bio     string `bson:"bio,omitempty" json:"bio"`
	Web     string `bson:"web,omitempty" json:"web,omitempty"`
}

type User struct {
	Id        bson.ObjectId `bson:"_id,omitempty"`
	Profile   Profile       `bson:"profile"`
	Email     string        `bson:"email"`
	Password  string        `bson:"password"`
	Superuser bool          `bson:"is_superuser"`
	Staff     bool          `bson:"is_staff"`
	CreatedAt time.Time     `bson:"created_at"`
}

type SerializeUser struct {
	Id        string    `json:"id"`
	Profile   Profile   `json:"profile"`
	Email     string    `json:"email"`
	Superuser bool      `json:"is_superuser"`
	Staff     bool      `json:"is_staff"`
	CreatedAt time.Time `json:"created_at"`
}

func Create(email, password string) (*User, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), 11)
	if err != nil {
		return nil, fmt.Errorf("error bcrypting password: %v", err)
	}

	str := base64.URLEncoding.EncodeToString(b)

	return &User{
		Email:     email,
		Password:  str,
		Superuser: false,
		Staff:     false,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (u *User) CollectionName() string {
	return "users"
}

func (u *User) SortBy() string {
	return "-created_at"
}

func (u *User) Unique() bson.M {
	if len(u.Id) > 0 {
		return bson.M{"_id": u.Id}
	}

	return bson.M{"email": u.Email}
}

func (u *User) Presave() {
}

func (user *User) VerifyPassword(pswd string) bool {
	b, err := base64.URLEncoding.DecodeString(user.Password)
	if err != nil {
		return false
	}
	if err := bcrypt.CompareHashAndPassword(b, []byte(pswd)); err != nil {
		return false
	}

	return true
}

func (user *User) IsAnonymous() bool {
	return len(user.Id) == 0
}

func (user *User) MarshalJSON() ([]byte, error) {
	return json.Marshal(&SerializeUser{
		Id:        user.Id.Hex(),
		Profile:   user.Profile,
		Email:     user.Email,
		Superuser: user.Superuser,
		Staff:     user.Staff,
		CreatedAt: user.CreatedAt,
	})
}
