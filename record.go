package bar

import (
	"context"
	"reflect"

	"github.com/uptrace/bun"
)

func RecordOf(model any) Record {
	return Record{model}
}

type Record struct {
	model any
}

func (r Record) Create(ctx context.Context, attrs ...Attribute) error {
	fns := []func(*bun.InsertQuery) *bun.InsertQuery{}
	relAttrs := map[string][]Attribute{}

	for _, attr := range attrs {
		fns = append(fns, attr.insertQuery())

		for rel, attr := range attr.relationAttributes() {
			relAttrs[rel] = append(relAttrs[rel], attr)
		}
	}

	if len(relAttrs) > 0 {
		return RunInTx(ctx, nil, func(ctx context.Context) error {
			if _, err := DBFromContext(ctx).NewInsert().Model(r.model).Apply(fns...).Exec(ctx); err != nil {
				return err
			}
			return r.createRelations(ctx, relAttrs)
		})
	}

	_, err := DBFromContext(ctx).NewInsert().Model(r.model).Apply(fns...).Exec(ctx)
	return err
}

func (r Record) createRelations(ctx context.Context, relAttrs map[string][]Attribute) error {
	mv := reflect.ValueOf(r.model)
	me := mv.Elem()
	table := DBFromContext(ctx).Dialect().Tables().Get(mv.Type())
	for name, attrs := range relAttrs {
		rel := table.Relations[name]
		v := reflect.New(rel.JoinTable.Type)
		e := v.Elem()
		for i, fk := range rel.JoinPKs {
			pk := rel.BasePKs[i]
			e.FieldByName(fk.GoName).Set(me.FieldByName(pk.GoName))
		}
		if err := RecordOf(v.Interface()).Create(ctx, attrs...); err != nil {
			return err
		}
		me.FieldByName(name).Set(v)
	}
	return nil
}
