package bar

import (
	"context"
)

func ModelFor[T any]() Model[T] {
	return Model[T]{}
}

type Model[T any] struct{}

func (m Model[T]) Create(ctx context.Context, attrs ...Attribute) (*T, error) {
	model := new(T)
	return model, RecordOf(model).Create(ctx, attrs...)
}
