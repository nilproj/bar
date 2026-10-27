package bar

import (
	"context"
	"database/sql"
	"reflect"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
)

func Model(mod any) *model {
	return &model{mod: mod}
}

type model struct {
	mod          any
	rel          *relation
	includedRels []string

	el reflect.Value
}

func (m *model) Relation(name string, mod any) *model {
	return &model{
		mod: mod,
		rel: &relation{m, name},
	}
}

func (m *model) Includes(rels ...string) *model {
	m.includedRels = append(m.includedRels, rels...)
	return m
}

func (m *model) With(data any) *model {
	el := m.elem()
	v := reflect.ValueOf(data)

	for f := range v.Fields() {
		fv := v.FieldByName(f.Name)
		if fv.IsZero() {
			continue
		}

		fel := el.FieldByName(f.Name)

		if fv.Type().Implements(ioptn) {
			if fv.FieldByName("Set").Bool() {
				fel.Set(fv.FieldByName("Value").Elem())
			}
			continue
		}

		fel.Set(indirect(fv))

	}

	return m
}

func (m *model) Take(ctx context.Context, lo ...int64) error {
	var l, o int64
	if len(lo) > 0 {
		l = lo[0]
	}
	if len(lo) > 1 {
		o = lo[1]
	}

	return m.newSelect(ctx).Limit(l).Offset(o).Scan(ctx)
}

func (m *model) FindOrCreate(ctx context.Context, cols ...string) error {
	err := m.Find(ctx, cols...)
	if err == sql.ErrNoRows {
		return m.Create(ctx)
	}
	return err
}

func (m *model) Find(ctx context.Context, cols ...string) error {
	return m.newSelect(ctx).WherePK(cols...).Scan(ctx)
}

func (m *model) Create(ctx context.Context) error {
	return RunInTx(ctx, nil, func(ctx context.Context) error {
		var afterCreate func() error
		if m.rel != nil {
			afterCreate = m.rel.beforeCreate(ctx, m.mod)
		}
		if _, err := DBFromContext(ctx).NewInsert().Model(m.mod).Exec(ctx); err != nil {
			return err
		}
		if fn := afterCreate; fn != nil {
			if err := fn(); err != nil {
				return err
			}
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

func (m *model) newSelect(ctx context.Context) *bun.SelectQuery {
	fns := []func(*bun.SelectQuery) *bun.SelectQuery{
		func(sq *bun.SelectQuery) *bun.SelectQuery {
			for _, rel := range m.includedRels {
				sq = sq.Relation(rel)
			}
			return sq
		},
	}
	if m.rel != nil {
		fns = append(fns, m.rel.selectQuery(ctx))
	}
	return DBFromContext(ctx).NewSelect().Model(m.mod).Apply(fns...)
}

func (m *model) createRelations(ctx context.Context) error {
	el := m.elem()

	for name := range m.table(ctx).Relations {
		fv := el.FieldByName(name)
		if fv.IsZero() {
			continue
		}

		rm := fv.Interface()
		if fv.Kind() == reflect.Slice {
			rm = newModels(fv)
		}
		if err := m.Relation(name, rm).Create(ctx); err != nil {
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

func indirect(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Pointer {
		return v.Elem()
	}
	return v
}
