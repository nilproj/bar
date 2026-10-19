package bar

import (
	"context"

	"github.com/uptrace/bun"
)

type dbCtxKey struct{}

func ContextWithDB(ctx context.Context, db bun.IDB) context.Context {
	return context.WithValue(ctx, dbCtxKey{}, db)
}

func DBFromContext(ctx context.Context) bun.IDB {
	return ctx.Value(dbCtxKey{}).(bun.IDB)
}
