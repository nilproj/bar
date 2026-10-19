package bar

import "github.com/uptrace/bun"

type Attribute map[string]any

func (attr Attribute) insertQuery() func(*bun.InsertQuery) *bun.InsertQuery {
	return func(iq *bun.InsertQuery) *bun.InsertQuery {
		for k, v := range attr {
			if _, ok := v.(Attribute); !ok {
				iq = iq.Value(k, "?", v)
			}
		}
		return iq
	}
}

func (attr Attribute) relationAttributes() map[string]Attribute {
	m := map[string]Attribute{}
	for k, v := range attr {
		if a, ok := v.(Attribute); ok {
			m[k] = a
		}
	}
	return m
}
