package auth

import (
	"time"

	"github.com/globalsign/mgo/bson"
	"golang.org/x/crypto/bcrypt"
)

type UserStatus string

const (
	UserStatusInActive UserStatus = "inactive"
	UserStatusActive   UserStatus = "active"
	UserStatusLocked   UserStatus = "locked"
)

func (st UserStatus) GetBSON() (interface{}, error) {
	return string(st), nil
}

func (st *UserStatus) SetBSON(raw bson.Raw) error {
	var status string
	err := raw.Unmarshal(&status)
	if err != nil {
		return err
	}

	*st = UserStatus(status)
	return nil
}

type Profile struct {
	Name    string `bson:"name,omitempty" json:"name,omitempty"`
	Picture string `bson:"picture,omitempty" json:"picture"`
	Bio     string `bson:"bio,omitempty" json:"bio"`
	Web     string `bson:"web,omitempty" json:"web,omitempty"`
}

type User struct {
	Id        bson.ObjectId `bson:"_id,omitempty" json:"id"`
	Profile   Profile       `bson:"profile" json:"profile,omitempty"`
	Email     string        `bson:"email" json:"email"`
	Password  []byte        `bson:"password" json:"-"`
	Status    UserStatus    `bson:"status" json:"status"`
	Superuser bool          `bson:"is_superuser" json:"is_superuser"`
	Staff     bool          `bson:"is_staff" json:"is_staff"`
	CreatedAt time.Time     `bson:"created_at" json:"created_at"`
}

type OAuthProvider struct {
	Id   bson.ObjectId `bson:"_id,omitempty"`
	Name string        `bson:"name"`
	Key  string        `bson:"key"`
	User bson.ObjectId `bson:"user"`
}

func NewUser(email, password string) (*User, error) {
	user := &User{
		Email:     email,
		Superuser: false,
		Staff:     false,
		CreatedAt: time.Now().UTC(),
	}
	if err := user.SetPassword([]byte(password)); err != nil {
		return nil, err
	}
	return user, nil
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
	if u.CreatedAt.IsZero() {
		u.CreatedAt = time.Now().UTC()
	}

	if len(u.Status) == 0 {
		u.Status = UserStatusActive
	}
}

func (user *User) SetPassword(pswd []byte) error {
	b, err := bcrypt.GenerateFromPassword(pswd, 11)
	if err != nil {
		return err
	}
	user.Password = b
	return nil
}

func (user *User) VerifyPassword(pswd string) bool {
	if err := bcrypt.CompareHashAndPassword(user.Password, []byte(pswd)); err != nil {
		return false
	}

	return true
}

func (user *User) IsActive() bool {
	return user.Status == UserStatusActive
}

func (p *OAuthProvider) CollectionName() string {
	return "oauth_providers"
}

func (p *OAuthProvider) Unique() bson.M {
	if len(p.Id) > 0 {
		return bson.M{"_id": p.Id}
	}

	return bson.M{"name": p.Name, "key": p.Key}
}

func (p *OAuthProvider) Presave() {
}
