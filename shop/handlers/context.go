package handlers

import (
	"context"

	scontext "github.com/syaiful6/thatique/context"
)

// Context should contain the request specific context for use in across
// handlers. Resources that don't need to be shared across handlers should not
// be on this object.
type Context struct {
	*App
	context.Context
}

// Value overrides context.Context.Value to ensure that calls are routed to
// correct context.
func (ctx *Context) Value(key interface{}) interface{} {
	return ctx.Context.Value(key)
}

func getName(ctx context.Context) (name string) {
	return scontext.GetStringValue(ctx, "vars.name")
}
