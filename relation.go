package bar

import (
	"context"
	"reflect"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
)

type relation struct {
	base *model
	name string
}

func (rel *relation) selectQuery(ctx context.Context) func(*bun.SelectQuery) *bun.SelectQuery {
	return func(sq *bun.SelectQuery) *bun.SelectQuery {
		r := rel.base.table(ctx).Relations[rel.name]
		el := rel.base.elem()

		switch r.Type {
		case schema.ManyToManyRelation:
			ba, ja := bun.Ident(r.JoinTable.Alias), bun.Ident(r.M2MTable.Alias)
			sq = sq.Join("JOIN ? AS ?", bun.Ident(r.M2MTable.Name), ja)

			for i, fk := range r.M2MJoinPKs {
				sq = sq.JoinOn("?.? = ?.?", ja, bun.Ident(fk.Name), ba, r.JoinPKs[i].Name)
			}
			for i, fk := range r.M2MBasePKs {
				sq = sq.Where("?.? = ?", ja, bun.Ident(fk.Name), el.FieldByName(r.BasePKs[i].GoName).Interface())
			}
		default:
			for i, fk := range r.JoinPKs {
				sq = sq.Where("? = ?", bun.Ident(fk.Name), el.FieldByName(r.BasePKs[i].GoName).Interface())
			}
			if f := r.PolymorphicField; f != nil {
				sq = sq.Where("? = ?", bun.Ident(f.Name), r.PolymorphicValue)
			}
		}

		return sq
	}
}

func (rel *relation) beforeCreate(ctx context.Context, model any) func() error {
	r := rel.base.table(ctx).Relations[rel.name]
	el := rel.base.elem()
	v := reflect.ValueOf(model).Elem()

	switch r.Type {
	case schema.HasOneRelation:
		setFKs(r, el, v)
	case schema.HasManyRelation:
		for i := range v.Len() {
			setFKs(r, el, v.Index(i).Elem())
		}
	case schema.ManyToManyRelation:
		return func() error {
			size := v.Len()
			s := reflect.MakeSlice(reflect.SliceOf(reflect.PointerTo(r.M2MTable.Type)), 0, size)
			for i := range size {
				v := newM2M(r, el, v.Index(i).Elem())
				s = reflect.Append(s, v)
			}
			if err := Model(newModels(s)).Create(ctx); err != nil {
				return err
			}
			return nil
		}
	}
	return nil
}

func newModels(v reflect.Value) any {
	r := reflect.New(v.Type())
	r.Elem().Set(v)
	return r.Interface()
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
