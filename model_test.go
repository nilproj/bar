package bar_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/nilproj/bar"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	"github.com/uptrace/bun/extra/bundebug"
)

type User struct {
	ID       int64 `bun:",pk,autoincrement"`
	Name     string
	Username string `bun:",notnull,unique"`

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

type logger struct {
	text string
}

var _ bun.QueryHook = (*logger)(nil)

func (*logger) BeforeQuery(ctx context.Context, e *bun.QueryEvent) context.Context {
	return ctx
}

func (l *logger) AfterQuery(ctx context.Context, e *bun.QueryEvent) {
	switch e.Operation() {
	case "SAVEPOINT":
		l.text = l.text + "\n" + "SAVEPOINT"
	case "RELEASE":
		l.text = l.text + "\n" + "RELEASE"
	default:
		l.text = l.text + "\n" + e.Query
	}
}

func (l *logger) test(t *testing.T, query string) {
	if !strings.HasSuffix(l.text, query) {
		t.Error("not expected query")
	}
}

func TestCRUD(t *testing.T) {
	l := &logger{}
	sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db := bun.NewDB(sqldb, sqlitedialect.New()).
		WithQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(testing.Verbose()))).
		WithQueryHook(l)
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
		{
			Name:     "user 1",
			Username: "u1",
		},
		{
			Name:     "user 2",
			Username: "u2",
			Profile: &Profile{
				Lang: "vi",
			},
		},
		{
			Name:     "user 3",
			Username: "u3",
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
	l.test(t, `BEGIN
INSERT INTO "users" ("name", "username") VALUES ('user 1', 'u1'), ('user 2', 'u2'), ('user 3', 'u3') RETURNING "id"
SAVEPOINT
INSERT INTO "profiles" ("lang", "user_id") VALUES ('vi', 2) RETURNING "id"
RELEASE
SAVEPOINT
INSERT INTO "posts" ("name", "user_id") VALUES ('post 1', 3), ('post 2', 3) RETURNING "id"
SAVEPOINT
INSERT INTO "articles" ("name") VALUES ('article 1'), ('article 2') RETURNING "id"
SAVEPOINT
INSERT INTO "article_posts" ("article_id", "post_id") VALUES (1, 1), (2, 1)
RELEASE
SAVEPOINT
INSERT INTO "comments" ("trackable_id", "trackable_type", "text") VALUES (2, 'article', 'comment 1')
RELEASE
RELEASE
SAVEPOINT
INSERT INTO "comments" ("trackable_id", "trackable_type", "text") VALUES (2, 'post', 'comment 2'), (2, 'post', 'comment 3')
RELEASE
RELEASE
COMMIT`)

	u3 := &User{}
	if err := bar.Model(u3).Take(ctx, 1, 2); err != nil {
		t.Error(err)
	}
	l.test(t, `SELECT "user"."id", "user"."name", "user"."username" FROM "users" AS "user" LIMIT 1 OFFSET 2`)

	var u3Posts []*Post
	if err := bar.Model(u3).Relation("Posts", &u3Posts).Take(ctx); err != nil {
		t.Error(err)
	}
	l.test(t, `SELECT "post"."id", "post"."name", "post"."user_id" FROM "posts" AS "post" WHERE ("user_id" = 3)`)

	p1, p2 := u3Posts[0], u3Posts[1]

	var p1Articles []*Article
	if err := bar.Model(p1).Relation("Articles", &p1Articles).Take(ctx); err != nil {
		t.Error(err)
	}
	l.test(t, `SELECT "article"."id", "article"."name" FROM "articles" AS "article" JOIN "article_posts" AS "article_post" ON ("article_post"."article_id" = "article".'id') WHERE ("article_post"."post_id" = 1)`)

	c2 := &Comment{}
	if err := bar.Model(p2).Relation("Comments", c2).Take(ctx, 1); err != nil {
		t.Error(err)
	}
	l.test(t, `SELECT "comment"."trackable_id", "comment"."trackable_type", "comment"."text" FROM "comments" AS "comment" WHERE ("trackable_id" = 2) AND ("trackable_type" = 'post') LIMIT 1`)

	a2 := &Article{ID: 2}
	if err := bar.Model(a2).Includes("Comments").Find(ctx); err != nil {
		t.Error(err)
	}
	l.test(t, `SELECT "article"."id", "article"."name" FROM "articles" AS "article" WHERE ("article"."id" = 2)
SELECT "comment"."trackable_id", "comment"."trackable_type", "comment"."text" FROM "comments" AS "comment" WHERE ("comment"."trackable_id" IN (2)) AND ("trackable_type" = 'article')`)

	p3 := new(Post)
	p3Input := PostInput{
		UserID: 1,
		Name:   bar.MakeOptional("post 3"),
	}
	if err := bar.Model(p3).With(p3Input).Create(ctx); err != nil {
		t.Error(err)
	}
	l.test(t, `BEGIN
INSERT INTO "posts" ("name", "user_id") VALUES ('post 3', 1) RETURNING "id"
COMMIT`)
}

type PostInput struct {
	UserID int64
	Name   bar.Optional[string]
}
