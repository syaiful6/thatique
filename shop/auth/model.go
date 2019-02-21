package auth

import (
	"fmt"
	"strings"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/syaiful6/thatique/pkg/emailparser"
	"github.com/syaiful6/thatique/shop/db"
	"golang.org/x/crypto/bcrypt"
)

type UserStatus string

const (
	UserStatusInactive UserStatus = "inactive"
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

	status = strings.ToLower(status)
	switch status {
	case "inactive", "active", "locked":
	default:
		return fmt.Errorf("Invalid user status %s must be one of [inactive, active, locked]", status)
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
	Id         bson.ObjectId   `bson:"_id,omitempty" json:"id"`
	Slug       string          `bson:"slug"`
	Profile    Profile         `bson:"profile" json:"profile,omitempty"`
	Email      string          `bson:"email" json:"email"`
	Password   []byte          `bson:"password" json:"-"`
	Status     UserStatus      `bson:"status" json:"status"`
	Superuser  bool            `bson:"is_superuser" json:"is_superuser"`
	Staff      bool            `bson:"is_staff" json:"is_staff"`
	CreatedAt  time.Time       `bson:"created_at" json:"created_at"`
	Identities []OAuthProvider `bson:"identities,omitempty" json:"-"`
}

type OAuthProvider struct {
	Name string `bson:"name"`
	Key  string `bson:"key"`
}

type UserFinder interface {
	FindUserById(id bson.ObjectId) (*User, error)
	FindUserByEmail(email string) (*User, error)
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

func (u *User) Indexes() []mgo.Index {
	return []mgo.Index{
		mgo.Index{Key: []string{"email"}, Unique: true,},
		mgo.Index{Key: []string{"slug"}, Unique: true,},
		mgo.Index{Key: []string{"identities.name", "identities.key"}, Unique: true, Sparse: true,},
	}
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

func (u *User) SlugQuery(slug string) bson.M {
	return bson.M{"slug": slug,}
}

func (u *User) Presave(conn *db.MongoConn) {
	if u.CreatedAt.IsZero() {
		u.CreatedAt = time.Now().UTC()
	}

	if u.Slug == "" {
		if u.Profile.Name != "" {
			slug, err := conn.GenerateSlug(u, u.Profile.Name)
			if err != nil {
				panic(err)
			}
			u.Slug = slug
		} else {
			email, _ := emailparser.NewEmail(u.Email)
			slug, err := conn.GenerateSlug(u, email.Local())
			if err != nil {
				panic(err)
			}
			u.Slug = slug
		}
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

func init() {
	db.Register(&User{})
}
