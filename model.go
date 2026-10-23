package bar

import (
	"context"
	"reflect"

	"github.com/uptrace/bun/schema"
)

func Model(mod any) *model {
	return &model{mod: mod}
}

type model struct {
	mod any

	el reflect.Value
}

func (m *model) Relation(name string) *relation {
	return &relation{m, name}
}

func (m *model) Create(ctx context.Context) error {
	return RunInTx(ctx, nil, func(ctx context.Context) error {
		if _, err := DBFromContext(ctx).NewInsert().Model(m.mod).Exec(ctx); err != nil {
			return err
		}

		el := m.elem()
		switch el.Kind() {
		case reflect.Struct:
			if err := m.createRelations(ctx); err != nil {
				return err
			}
		case reflect.Slice:
			for i := range el.Len() {
				if err := Model(el.Index(i).Interface()).createRelations(ctx); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func (m *model) createRelations(ctx context.Context) error {
	el := m.elem()

	for name := range m.table(ctx).Relations {
		fv := el.FieldByName(name)
		if fv.IsZero() {
			continue
		}

		if err := m.Relation(name).Create(ctx, fv.Interface()); err != nil {
			return err
		}
	}

	return nil
}

func (m *model) table(ctx context.Context) *schema.Table {
	return DBFromContext(ctx).Dialect().Tables().Get(m.elem().Type())
}

func (m *model) elem() reflect.Value {
	if !m.el.IsValid() {
		m.el = reflect.ValueOf(m.mod).Elem()
	}
	return m.el
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
