package bar

import (
	"context"

	"github.com/uptrace/bun"
)

func ModelFor[T any]() Model[T] {
	return Model[T]{}
}

type Model[T any] struct{}

func (m Model[T]) Create(ctx context.Context, attrs ...Attribute) (*T, error) {
	model := new(T)
	return model, Create(ctx, model, attrs...)
}

func Create(ctx context.Context, model any, attrs ...Attribute) error {
	fns := []func(*bun.InsertQuery) *bun.InsertQuery{}
	for _, attr := range attrs {
		fns = append(fns, attr.InsertQuery())
	}

	_, err := DBFromContext(ctx).NewInsert().Model(model).Apply(fns...).Exec(ctx)
	return err
}
