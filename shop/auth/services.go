package auth

import (
	"github.com/globalsign/mgo/bson"
	"github.com/syaiful6/thatique/shop/db"
)

// Find user by Id
func FindUserById(id bson.ObjectId) (*User, error) {
	var user *User
	if err := db.Conn.Find(user, bson.M{"_id": id}).One(&user); err != nil {
		return nil, err
	}
	return user, nil
}

// Find user by Email
func FindUserByEmail(email string) (*User, error) {
	var user *User
	if err := db.Conn.Find(user, bson.M{"email": email}).One(&user); err != nil {
		return nil, err
	}
	return user, nil
}

func FindUserBySlug(slug string) (*User, error) {
	var user *User
	if err := db.Conn.Find(user, bson.M{"slug": slug}).One(&user); err != nil {
		return nil, err
	}
	return user, nil
}

// Add the user if one doesn't exist for this identity and set the data for
// that provider for the user whether the user is new or not.
func FindOrCreateUserForProvider(user *User, provider OAuthProvider) (*User, bool, error) {
	query := bson.M{
		"identities": bson.M{
			"$elemMatch": bson.M{
				"name": provider.Name,
				"key":  provider.Key,
			},
		},
	}
	var userData *User
	*userData = *user
	user.Identities = []OAuthProvider{provider}

	userData.Presave()
	info, err := db.Conn.Cursor(userData).Upsert(
		query,
		bson.M{
			"$setOnInsert": userData,
		},
	)
	if err != nil {
		return nil, false, err
	}

	if info.UpsertedId != nil {
		userData.Id = info.UpsertedId.(bson.ObjectId)
		return userData, true, err
	}

	var existingUser *User
	err = db.Conn.Find(user, query).One(&existingUser)
	return existingUser, false, err
}

// Push a provider to an existing user. Maybe because user connect their account
// with different provider
func PushProviderForUser(user *User, provider OAuthProvider) error {
	return db.Conn.Cursor(user).Update(user.Unique(), bson.M{
		"$addToSet": bson.M{
			"identities": provider,
		},
	})
}
