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
			setFKs(v, r, fv.Elem())
			if err := Create(ctx, fv.Interface()); err != nil {
				return err
			}
		case schema.HasManyRelation:
			for i := range fv.Len() {
				setFKs(v, r, fv.Index(i).Elem())
			}
			sv := reflect.New(fv.Type())
			sv.Elem().Set(fv)
			if err := Create(ctx, sv.Interface()); err != nil {
				return err
			}
		}
	}
	return nil
}

func setFKs(bv reflect.Value, r *schema.Relation, jv reflect.Value) {
	for i, fk := range r.JoinPKs {
		jv.FieldByName(fk.GoName).Set(bv.FieldByName(r.BasePKs[i].GoName))
	}
}
