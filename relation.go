package bar

import (
	"context"
	"reflect"

	"github.com/uptrace/bun/schema"
)

type relation struct {
	m    *model
	name string
}

func (rel *relation) Create(ctx context.Context, values any) error {
	r := rel.m.table(ctx).Relations[rel.name]
	el := rel.m.elem()
	v := reflect.ValueOf(values)

	switch r.Type {
	case schema.HasOneRelation:
		setFKs(r, el, v.Elem())
		if err := Model(v.Interface()).Create(ctx); err != nil {
			return err
		}
	case schema.HasManyRelation:
		for i := range v.Len() {
			setFKs(r, el, v.Index(i).Elem())
		}
		if err := Model(newModels(v)).Create(ctx); err != nil {
			return err
		}
	case schema.ManyToManyRelation:
		if err := Model(newModels(v)).Create(ctx); err != nil {
			return err
		}
		size := v.Len()
		s := reflect.MakeSlice(reflect.SliceOf(reflect.PointerTo(r.M2MTable.Type)), 0, size)
		for i := range size {
			v := newM2M(r, el, v.Index(i).Elem())
			s = reflect.Append(s, v)
		}
		if err := Model(newModels(s)).Create(ctx); err != nil {
			return err
		}
	}
	return nil
}

func newModels(v reflect.Value) any {
	r := reflect.New(v.Type())
	r.Elem().Set(v)
	return r.Interface()
}
