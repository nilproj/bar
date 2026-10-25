package bar

import (
	"context"
	"reflect"

	"github.com/graph-gophers/dataloader/v7"
)

type PKs map[string]any

func (pks PKs) equal(v reflect.Value) bool {
	el := v.Elem()
	for k, v := range pks {
		if !el.FieldByName(k).Equal(reflect.ValueOf(v)) {
			return false
		}
	}
	return true
}

func LoaderFor[T any]() *modelLoader[T] {
	return &modelLoader[T]{}
}

type modelLoader[T any] struct{}

func (*modelLoader[T]) Load(ctx context.Context, pks PKs) (*T, error) {
	v, err := modLdr.Load(ctx, &modelKey{reflect.TypeFor[T](), pks})()
	if err != nil {
		return nil, err
	}
	return v.(*T), nil
}

type modelKey struct {
	typ reflect.Type
	pks PKs
}

func (k modelKey) newModel() reflect.Value {
	v := reflect.New(k.typ)
	el := v.Elem()
	for k, v := range k.pks {
		el.FieldByName(k).Set(reflect.ValueOf(v))
	}
	return v
}

var modLdr = dataloader.NewBatchedLoader(func(ctx context.Context, keys []*modelKey) (results []*dataloader.Result[any]) {
	grData, grErr := map[reflect.Type]reflect.Value{}, map[reflect.Type]error{}

	for _, k := range keys {
		if !grData[k.typ].IsValid() {
			grData[k.typ] = reflect.MakeSlice(reflect.SliceOf(reflect.PointerTo(k.typ)), 0, 0)
		}
		grData[k.typ] = reflect.Append(grData[k.typ], k.newModel())
	}

	for t, s := range grData {
		v := reflect.New(s.Type())
		v.Elem().Set(s)
		grErr[t] = Model(v.Interface()).Find(ctx)
	}

	for _, k := range keys {
		var data any
		mods := grData[k.typ]
		for i := range mods.Len() {
			if mod := mods.Index(i); k.pks.equal(mod) {
				data = mod.Interface()
			}
		}
		results = append(results, &dataloader.Result[any]{
			Data:  data,
			Error: grErr[k.typ],
		})
	}

	return
})
