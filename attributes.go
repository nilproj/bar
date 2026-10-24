package bar

import "github.com/uptrace/bun"

type Attributes map[string]any

func (attrs Attributes) selectQuery(sq *bun.SelectQuery) *bun.SelectQuery {
	for k, v := range attrs {
		sq = sq.Where("? = ?", bun.Ident(k), v)
	}
	return sq
}
