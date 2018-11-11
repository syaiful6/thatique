package auth

import (
	"context"
	"fmt"

	"github.com/gorilla/mux"
)

const (
	// UserKey is used to get the user object from
	// a user context
	UserKey = "auth.user"

	userIdKey = "auth.user.id"
)

type UserInfo struct {
	Id   string
}

func WithUser(ctx context.Context, user UserInfo) context.Context {
	return userInfoContext{
		Context: ctx,
		user: user,
	}
}

type userInfoContext struct {
	context.Context

	user UserInfo
}

func (uic userInfoContext) Value(key interface{}) interface{} {
	switch key {
	case UserKey:
		return uic.user
	case userIdKey:
		return uic.user.Id
	}

	return uic.Context.Value(key)
}

// auth strategies is function that take options map and return `mux.MiddlewareFunc`
type AuthStrategy func(options map[string]interface{}) (mux.MiddlewareFunc, error)

var authStrategies map[string]AuthStrategy

func init() {
	authStrategies = make(map[string]AuthStrategy)
}

func Register(name string, strategy AuthStrategy) error {
	if _, exists := authStrategies[name]; exists {
		return fmt.Errorf("name already registered: %s", name)
	}

	authStrategies[name] = strategy

	return nil
}

func GetStrategy(name string, options map[string]interface{}) (mux.MiddlewareFunc, error) {
	if strategy, exists := authStrategies[name]; exists {
		return strategy(options)
	}

	return nil, fmt.Errorf("no auth strategy registered with name: %s", name)
}
