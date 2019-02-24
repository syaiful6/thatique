package buntdbtoken

import (
	"testing"

	"github.com/globalsign/mgo/bson"
	"github.com/syaiful6/thatique/shop/auth"
)

func TestGenerateToken(t *testing.T) {
	id := bson.NewObjectId()
	user, _ := auth.NewUser("foo@baz.com", "secret")
	user.Id = id

	generator, err := NewMemoryTokenGenerator()
	token, err := generator.Generate(user)
	if err != nil {
		t.Errorf("Failed to create token %s", err.Error())
		return
	}

	valid := generator.IsValid(user, token)
	if !valid {
		t.Errorf("expected generator.IsValid to return true but it's return %v", valid)
	}

	// delete it
	err = generator.Delete(token)
	if err != nil {
		t.Errorf("generator delete return error: %v", err)
		return
	}
	valid = generator.IsValid(user, token)
	if valid {
		t.Errorf("expected generator.IsValid to return false but it's return %v", valid)
	}
}
