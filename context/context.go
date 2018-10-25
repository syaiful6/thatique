package context

import (
	"context"
	"sync"

	"github.com/syaiful6/thatique/uuid"
)

// instanceContext is a context that provides only an instance id. It is
// provided as the main background context.
type instanceContext struct {
	context.Context
	id   string    // id of context, logged as "instance.id"
	once sync.Once // once protect generation of the id
}

func (ic *instanceContext) Value(key interface{}) interface{} {
	if key == "instance.id" {
		ic.once.Do(func() {
			// We want to lazy initialize the UUID such that we don't
			// call a random generator from the package initialization
			// code. For various reasons random could not be available
			ic.id = uuid.Generate().String()
		})
		return ic.id
	}

	return ic.Context.Value(key)
}

var background = &instanceContext{
	Context: context.Background(),
}

// Background returns a non-nil, empty Context. The background context
// provides a single key, "instance.id" that is globally unique to the
// process.
func Background() context.Context {
	return background
}