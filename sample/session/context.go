package session

import (
	"context"
	"multisig/durable"
	"net/http"

	"github.com/unrolled/render"
)

type contextValueKey int

const (
	keyRequest       contextValueKey = 0
	keyDatabase      contextValueKey = 1
	keyLogger        contextValueKey = 2
	keyRender        contextValueKey = 3
	keyRemoteAddress contextValueKey = 11
)

func Logger(ctx context.Context) *durable.Logger {
	v, _ := ctx.Value(keyLogger).(*durable.Logger)
	return v
}

func Database(ctx context.Context) *durable.Database {
	v, _ := ctx.Value(keyDatabase).(*durable.Database)
	return v
}

func Render(ctx context.Context) *render.Render {
	v, _ := ctx.Value(keyRender).(*render.Render)
	return v
}

func Request(ctx context.Context) *http.Request {
	v, _ := ctx.Value(keyRequest).(*http.Request)
	return v
}

func RemoteAddress(ctx context.Context) string {
	v, _ := ctx.Value(keyRemoteAddress).(string)
	return v
}

func WithLogger(ctx context.Context, logger *durable.Logger) context.Context {
	return context.WithValue(ctx, keyLogger, logger)
}

func WithDatabase(ctx context.Context, database *durable.Database) context.Context {
	return context.WithValue(ctx, keyDatabase, database)
}

func WithRender(ctx context.Context, render *render.Render) context.Context {
	return context.WithValue(ctx, keyRender, render)
}

func WithRequest(ctx context.Context, r *http.Request) context.Context {
	rCopy := new(http.Request)
	*rCopy = *r
	return context.WithValue(ctx, keyRequest, rCopy)
}

func WithRemoteAddress(ctx context.Context, remoteAddr string) context.Context {
	return context.WithValue(ctx, keyRemoteAddress, remoteAddr)
}
