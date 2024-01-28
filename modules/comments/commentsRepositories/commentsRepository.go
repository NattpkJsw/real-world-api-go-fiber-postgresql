package commentsrepositories

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/NattpkJsw/real-world-api-go/modules/comments"
	"github.com/jmoiron/sqlx"
)

type ICommentsRepository interface {
	FindComments(aritcleID, userID int) ([]*comments.Comment, error)
	InsertComment(req *comments.CommentCredential) (*comments.Comment, error)
}

type commentRepository struct {
	db *sqlx.DB
}

func CommentRepository(db *sqlx.DB) ICommentsRepository {
	return &commentRepository{
		db: db,
	}
}

func (r *commentRepository) FindComments(aritcleID, userID int) ([]*comments.Comment, error) {
	query := `
	SELECT
		array_to_json(array_agg("cs"))
	FROM (
		SELECT
			"c"."id",
			"c"."created_at",
			"c"."updated_at",
			"c"."body",
			(
				SELECT 
					json_build_object(
						'username', "u"."username",
						'bio', "u"."bio",
						'image', "u"."image",
						'following',
						CASE 
							WHEN EXISTS (
								SELECT 1
								FROM "user_follows" "uf"
								WHERE "c"."author_id" = "uf"."following_id"  AND "uf"."follower_id" = $2
							) THEN TRUE 
							ELSE FALSE 
						END
					)
				FROM "users" "u"
				WHERE "c"."author_id" = "u"."id"
			) AS "author"
		FROM "comments" "c"
		WHERE "article_id" = $1
	) AS "cs";`

	bytes := make([]byte, 0)
	comments := make([]*comments.Comment, 0)

	if err := r.db.Get(&bytes, query, aritcleID, userID); err != nil {
		return nil, fmt.Errorf("get comments failed: %v", err)
	}

	if err := json.Unmarshal(bytes, &comments); err != nil {
		return nil, fmt.Errorf("unmarshal failed: %v", err)
	}
	return comments, nil
}

func (r *commentRepository) FindSingleComment(commentID, userID int) (*comments.Comment, error) {
	query := `
	SELECT
		to_jsonb("cmt")
	FROM
	(
		SELECT
			"id",
			"created_at",
			"updated_at",
			"body",
			(
				SELECT 
					json_build_object(
						'username', "u"."username",
						'bio', "u"."bio",
						'image', "u"."image",
						'following',
						CASE 
							WHEN EXISTS (
								SELECT 1
								FROM "user_follows" "uf"
								WHERE "c"."author_id" = "uf"."following_id"  AND "uf"."follower_id" = $2
							) THEN TRUE 
							ELSE FALSE 
						END
					)
				FROM "users" "u"
				WHERE "c"."author_id" = "u"."id"
			) AS "author"
		FROM "comments" "c"
		WHERE "id" = $1
	) AS "cmt";`

	bytes := make([]byte, 0)
	comment := new(comments.Comment)
	if err := r.db.Get(&bytes, query, commentID, userID); err != nil {
		return nil, fmt.Errorf("get comment failed: %v", err)
	}
	if err := json.Unmarshal(bytes, &comment); err != nil {
		return nil, fmt.Errorf("unmarshal failed: %v", err)
	}

	return comment, nil
}

func (r *commentRepository) InsertComment(req *comments.CommentCredential) (*comments.Comment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var commentID int
	query := `
	INSERT INTO "comments"(
		"body",
		"article_id",
		"author_id"
	)
	VALUES
	($1, $2, $3)
	RETURNING "id";`

	fmt.Println("body === ", req.Body)
	if err := r.db.QueryRowxContext(ctx, query, req.Body, req.ArticleID, req.AuthorID).Scan(&commentID); err != nil {
		return nil, fmt.Errorf("insert comment failed: %v", err)
	}

	return r.FindSingleComment(commentID, req.AuthorID)
}
