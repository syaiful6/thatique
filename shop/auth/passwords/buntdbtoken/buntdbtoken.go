package buntdbtoken

import (
	"crypto/subtle"
	"encoding/json"
	"time"

	"github.com/syaiful6/thatique/shop/auth"
	"github.com/syaiful6/thatique/shop/auth/passwords"
	"github.com/tidwall/buntdb"
)

type BuntDbToken struct {
	Token     string
	Email     string
	Pass      []byte
	CreatedAt int64
}

type BuntDbTokenGenerator struct {
	db     *buntdb.DB
	Expire int // default two hours
}

// NewBuntDbTokenGenerator create a token store instance based on memory
func NewMemoryTokenGenerator() (generator *BuntDbTokenGenerator, err error) {
	generator, err = NewFileTokenGenerator(":memory:")
	return
}

// NewBuntDbTokenGenerator create a token store instance based on file
func NewFileTokenGenerator(filename string) (generator *BuntDbTokenGenerator, err error) {
	db, err := buntdb.Open(filename)
	if err != nil {
		return
	}
	generator = &BuntDbTokenGenerator{db: db, Expire: 7200}
	return
}

func (t *BuntDbTokenGenerator) Generate(user *auth.User) (token string, err error) {
	token, err = passwords.GenerateToken()
	if err != nil {
		return "", err
	}

	tv, err := json.Marshal(&BuntDbToken{
		Token:     token,
		Email:     user.Email,
		Pass:      user.Password,
		CreatedAt: time.Now().UTC().Unix(),
	})
	if err != nil {
		return "", err
	}

	err = t.db.Update(func(tx *buntdb.Tx) (err error) {
		expire := time.Duration(t.Expire) * time.Second
		hextId := user.Id.Hex()
		_, _, err = tx.Set(hextId, string(tv), &buntdb.SetOptions{Expires: true, TTL: expire})
		if err != nil {
			return
		}
		_, _, err = tx.Set(token, hextId, &buntdb.SetOptions{Expires: true, TTL: expire})
		return
	})
	return
}

func (t *BuntDbTokenGenerator) Delete(token string) (err error) {
	verr := t.db.Update(func(tx *buntdb.Tx) (err error) {
		pk, err := tx.Get(token)
		if err != nil {
			return
		}
		_, err = tx.Delete(token)
		if err != nil {

		}
		_, err = tx.Delete(pk)
		return
	})
	if verr == buntdb.ErrNotFound {
		return
	}
	err = verr
	return
}

func (t *BuntDbTokenGenerator) IsValid(user *auth.User, token string) (valid bool) {
	err := t.db.View(func(tx *buntdb.Tx) (err error) {
		jv, err := tx.Get(user.Id.Hex())
		if err != nil {
			return
		}
		var s *BuntDbToken
		err = json.Unmarshal([]byte(jv), &s)
		if err != nil {
			return
		}

		if s.Token == token && s.Email == user.Email && subtle.ConstantTimeCompare(s.Pass, user.Password) == 1 {
			valid = true
		}
		return
	})
	if err != nil {
		return false
	}

	return
}
