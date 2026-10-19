package bar

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"
)

func RunInTx(ctx context.Context, opts *sql.TxOptions, fn func(ctx context.Context) error) error {
	return DBFromContext(ctx).RunInTx(ctx, opts, func(ctx context.Context, tx bun.Tx) error {
		return fn(ContextWithDB(ctx, tx))
	})
}
