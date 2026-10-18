package bar

import "github.com/uptrace/bun"

type Attribute map[string]any

func (attr Attribute) InsertQuery() func(*bun.InsertQuery) *bun.InsertQuery {
	return func(iq *bun.InsertQuery) *bun.InsertQuery {
		for k, v := range attr {
			iq = iq.Value(k, "?", v)
		}
		return iq
	}
}
