package bar_test

import (
	"database/sql"
	"testing"

	"github.com/nilproj/bar"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	"github.com/uptrace/bun/extra/bundebug"
)

type User struct {
	ID   int64 `bun:",pk,autoincrement"`
	Name string

	Profile *Profile `bun:"rel:has-one,join:id=user_id"`
	Posts   []*Post  `bun:"rel:has-many,join:id=user_id"`
}

type Profile struct {
	ID     int64 `bun:",pk,autoincrement"`
	Lang   string
	UserID int64

	User *User `bun:"rel:belongs-to,join:user_id=id"`
}

type Post struct {
	ID     int64 `bun:",pk,autoincrement"`
	Name   string
	UserID int64

	User     *User      `bun:"rel:belongs-to,join:user_id=id"`
	Comments []*Comment `bun:"rel:has-many,join:id=trackable_id,join:type=trackable_type,polymorphic"`
	Articles []*Article `bun:"m2m:article_posts,join:Post=Article"`
}

type Article struct {
	ID   int64 `bun:",pk,autoincrement"`
	Name string

	Comments []*Comment `bun:"rel:has-many,join:id=trackable_id,join:type=trackable_type,polymorphic"`
	Posts    []*Post    `bun:"m2m:article_posts,join:Article=Post"`
}

type Comment struct {
	TrackableID   int64
	TrackableType string
	Text          string
}

type ArticlePost struct {
	ArticleID int64 `bun:",pk"`
	PostID    int64 `bun:",pk"`

	Article *Article `bun:"rel:belongs-to,join:article_id=id"`
	Post    *Post    `bun:"rel:belongs-to,join:post_id=id"`
}

func TestCRUD(t *testing.T) {
	sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db := bun.NewDB(sqldb, sqlitedialect.New()).
		WithQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(testing.Verbose())))
	defer db.Close()

	if err := db.ResetModel(t.Context(),
		(*User)(nil),
		(*Profile)(nil),
		(*ArticlePost)(nil),
		(*Post)(nil),
		(*Article)(nil),
		(*Comment)(nil),
	); err != nil {
		t.Fatal(err)
	}

	ctx := bar.ContextWithDB(t.Context(), db)

	users := []*User{
		{Name: "user 1"},
		{
			Name: "user 2",
			Profile: &Profile{
				Lang: "vi",
			},
		},
		{
			Name: "user 3",
			Posts: []*Post{
				{
					Name: "post 1",
					Articles: []*Article{
						{Name: "article 1"},
						{
							Name: "article 2",
							Comments: []*Comment{
								{Text: "comment 1"},
							},
						},
					},
				},
				{
					Name: "post 2",
					Comments: []*Comment{
						{Text: "comment 2"},
						{Text: "comment 3"},
					},
				},
			},
		},
	}
	if err := bar.Model(&users).Create(ctx); err != nil {
		t.Error(err)
	}
}
