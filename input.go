package bar

import "reflect"

type IOptional interface {
	Nullable()
}

var ioptn = reflect.TypeFor[IOptional]()

type Optional[T any] struct {
	Value *T
	Set   bool
}

var _ IOptional = (*Optional[any])(nil)

func (o *Optional[T]) Nullable() {}

func MakeOptional[T any](v T) Optional[T] {
	return Optional[T]{
		Set:   true,
		Value: &v,
	}
}
