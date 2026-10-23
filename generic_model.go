package bar

func ModelFor[T any]() GenericModel[T] {
	return GenericModel[T]{}
}

type GenericModel[T any] struct{}
