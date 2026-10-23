package bar

import (
	"context"
	"reflect"

	"github.com/uptrace/bun/schema"
)

func ModelFor[T any]() Model[T] {
	return Model[T]{}
}

type Model[T any] struct{}

func (m Model[T]) Create(ctx context.Context, attrs ...Attribute) (*T, error) {
	model := new(T)
	return model, RecordOf(model).Create(ctx, attrs...)
}

func Create(ctx context.Context, model any) error {
	return RunInTx(ctx, nil, func(ctx context.Context) error {
		db := DBFromContext(ctx)

		if _, err := db.NewInsert().Model(model).Exec(ctx); err != nil {
			return err
		}

		mv := reflect.ValueOf(model).Elem()
		switch mv.Kind() {
		case reflect.Struct:
			if err := createRelations(ctx, mv); err != nil {
				return err
			}
		case reflect.Slice:
			for i := range mv.Len() {
				if err := createRelations(ctx, mv.Index(i).Elem()); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func createRelations(ctx context.Context, v reflect.Value) error {
	table := DBFromContext(ctx).Dialect().Tables().Get(v.Type())
	for name, r := range table.Relations {
		fv := v.FieldByName(name)
		if fv.IsZero() {
			continue
		}
		switch r.Type {
		case schema.HasOneRelation:
			setFKs(r, v, fv.Elem())
			if err := Create(ctx, fv.Interface()); err != nil {
				return err
			}
		case schema.HasManyRelation:
			for i := range fv.Len() {
				setFKs(r, v, fv.Index(i).Elem())
			}
			if err := Create(ctx, newSlice(fv).Interface()); err != nil {
				return err
			}
		case schema.ManyToManyRelation:
			if err := Create(ctx, newSlice(fv).Interface()); err != nil {
				return err
			}
			size := fv.Len()
			s := reflect.MakeSlice(reflect.SliceOf(reflect.PointerTo(r.M2MTable.Type)), 0, size)
			for i := range size {
				v := newM2M(r, v, fv.Index(i).Elem())
				s = reflect.Append(s, v)
			}
			if err := Create(ctx, newSlice(s).Interface()); err != nil {
				return err
			}
		}
	}
	return nil
}

func setFKs(r *schema.Relation, bv reflect.Value, jv reflect.Value) {
	for i, fk := range r.JoinPKs {
		jv.FieldByName(fk.GoName).Set(bv.FieldByName(r.BasePKs[i].GoName))
	}
	if pf := r.PolymorphicField; pf != nil {
		jv.FieldByName(pf.GoName).Set(reflect.ValueOf(r.PolymorphicValue))
	}
}

func newM2M(r *schema.Relation, bv reflect.Value, jv reflect.Value) reflect.Value {
	v := reflect.New(r.M2MTable.Type)
	el := v.Elem()
	for i, fk := range r.M2MJoinPKs {
		el.FieldByName(fk.GoName).Set(jv.FieldByName(r.JoinPKs[i].GoName))
	}
	for i, fk := range r.M2MBasePKs {
		el.FieldByName(fk.GoName).Set(bv.FieldByName(r.BasePKs[i].GoName))
	}
	return v
}

func newSlice(v reflect.Value) reflect.Value {
	r := reflect.New(v.Type())
	r.Elem().Set(v)
	return r
}
