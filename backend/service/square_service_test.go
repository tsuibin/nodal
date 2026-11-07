package service

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/nodal/nodal-server/apperr"
	"github.com/nodal/nodal-server/model"
	"github.com/nodal/nodal-server/token"
)

func newSquareTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:square_" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Canvas{}, &model.CanvasLike{}, &model.CanvasComment{}, &model.CanvasShareLink{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func seedPublicCanvas(t *testing.T, db *gorm.DB, ownerKey string) *model.Canvas {
	t.Helper()
	now := time.Now().UTC()
	canvas := &model.Canvas{
		OwnerKey:    ownerKey,
		Title:       "测试画布",
		Nodes:       []byte("[]"),
		Edges:       []byte("[]"),
		Viewport:    []byte(`{"x":0,"y":0,"zoom":1}`),
		Tags:        []byte(`["AI"]`),
		IsPublic:    true,
		ShareMode:   model.ShareModePublic,
		PublishedAt: &now,
	}
	if err := db.Create(canvas).Error; err != nil {
		t.Fatalf("create canvas: %v", err)
	}
	return canvas
}

func TestLikeToggleKeepsCountConsistent(t *testing.T) {
	db := newSquareTestDB(t)
	canvas := seedPublicCanvas(t, db, "u:1")
	square := NewSquareService(db)
	id := &token.Identity{Role: token.RoleUser, UserID: 2}

	result, err := square.Like(context.Background(), id, canvas.ID)
	if err != nil {
		t.Fatalf("like: %v", err)
	}
	if !result.Liked || result.LikeCount != 1 {
		t.Fatalf("first like = %+v", result)
	}

	result, err = square.Like(context.Background(), id, canvas.ID)
	if err != nil {
		t.Fatalf("unlike: %v", err)
	}
	if result.Liked || result.LikeCount != 0 {
		t.Fatalf("second like = %+v", result)
	}

	// 未发布画布不能点赞
	unpublished := &model.Canvas{OwnerKey: "u:1", Title: "未发布", Nodes: []byte("[]"),
		Edges: []byte("[]"), Viewport: []byte(`{}`), Tags: []byte("[]"), IsPublic: false,
		ShareMode: model.ShareModePrivate}
	if err := db.Create(unpublished).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := square.Like(context.Background(), id, unpublished.ID); err == nil {
		t.Fatalf("期望未发布画布点赞失败")
	} else if apperr.From(err).Code != apperr.CodeCanvasNotPublished {
		t.Fatalf("错误码 = %d", apperr.From(err).Code)
	}
}

func TestCommentCreateAndDeleteUpdatesCount(t *testing.T) {
	db := newSquareTestDB(t)
	canvas := seedPublicCanvas(t, db, "u:1")
	square := NewSquareService(db)
	id := &token.Identity{Role: token.RoleUser, UserID: 2}

	comment, err := square.CreateComment(context.Background(), id, canvas.ID, "  很有启发  ")
	if err != nil {
		t.Fatalf("comment: %v", err)
	}
	if comment.Content != "很有启发" || !comment.Mine {
		t.Fatalf("comment view = %+v", comment)
	}

	var after model.Canvas
	_ = db.First(&after, canvas.ID).Error
	if after.CommentCount != 1 {
		t.Fatalf("comment_count = %d", after.CommentCount)
	}

	if err := square.DeleteComment(context.Background(), id, canvas.ID, comment.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_ = db.First(&after, canvas.ID).Error
	if after.CommentCount != 0 {
		t.Fatalf("delete 后 comment_count = %d", after.CommentCount)
	}
	if _, err := square.ListComments(context.Background(), id, canvas.ID, 1, 20); err != nil {
		t.Fatalf("list: %v", err)
	}
}

func TestShareLinkPermissionAndRevoke(t *testing.T) {
	db := newSquareTestDB(t)
	canvas := seedPublicCanvas(t, db, "u:1")
	canvases := NewCanvasService(db)
	shares := NewShareService(db, canvases)
	owner := &token.Identity{Role: token.RoleUser, UserID: 1}

	readLink, err := shares.Create(context.Background(), owner, canvas.ID, CreateShareLinkInput{Permission: model.SharePermissionRead})
	if err != nil {
		t.Fatalf("create read link: %v", err)
	}
	view, err := shares.Resolve(context.Background(), readLink.Token)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if view.CanFork {
		t.Fatalf("read 链接不应允许 fork")
	}

	forkLink, err := shares.Create(context.Background(), owner, canvas.ID, CreateShareLinkInput{Permission: model.SharePermissionFork})
	if err != nil {
		t.Fatalf("create fork link: %v", err)
	}
	view, err = shares.Resolve(context.Background(), forkLink.Token)
	if err != nil || !view.CanFork {
		t.Fatalf("fork link resolve = %+v, %v", view, err)
	}

	if err := shares.Revoke(context.Background(), owner, canvas.ID, forkLink.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, err := shares.Resolve(context.Background(), forkLink.Token); err == nil {
		t.Fatalf("停用后应失效")
	} else if apperr.From(err).Code != apperr.CodeShareInvalid {
		t.Fatalf("错误码 = %d", apperr.From(err).Code)
	}

	// 过期链接
	expiredLink, err := shares.Create(context.Background(), owner, canvas.ID, CreateShareLinkInput{Permission: model.SharePermissionFork, ExpiresHours: 1})
	if err != nil {
		t.Fatalf("create expired link: %v", err)
	}
	past := time.Now().Add(-time.Hour)
	if err := db.Model(&model.CanvasShareLink{}).Where("id = ?", expiredLink.ID).Update("expires_at", past).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := shares.Resolve(context.Background(), expiredLink.Token); err == nil {
		t.Fatalf("过期链接应失效")
	} else if apperr.From(err).Code != apperr.CodeShareExpired {
		t.Fatalf("错误码 = %d", apperr.From(err).Code)
	}
}

func TestSquarePublishVisibility(t *testing.T) {
	db := newSquareTestDB(t)
	square := NewSquareService(db)
	owner := &token.Identity{Role: token.RoleUser, UserID: 1}
	canvas := &model.Canvas{OwnerKey: "u:1", Title: "待发布", Nodes: []byte("[]"),
		Edges: []byte("[]"), Viewport: []byte(`{}`), Tags: []byte(`["AI"]`), ShareMode: model.ShareModePrivate}
	if err := db.Create(canvas).Error; err != nil {
		t.Fatal(err)
	}

	if _, err := square.Publish(context.Background(), owner, canvas.ID); err != nil {
		t.Fatalf("publish: %v", err)
	}
	list, err := square.List(context.Background(), nil, SquareQuery{Tag: "AI"})
	if err != nil {
		t.Fatal(err)
	}
	if list.Total != 1 || len(list.Items) != 1 || list.Items[0].Author.Nickname == "" {
		t.Fatalf("square list = %+v", list)
	}

	if _, err := square.Unpublish(context.Background(), owner, canvas.ID); err != nil {
		t.Fatalf("unpublish: %v", err)
	}
	list, _ = square.List(context.Background(), nil, SquareQuery{Tag: "AI"})
	if list.Total != 0 {
		t.Fatalf("下架后仍出现在广场: %+v", list)
	}
}
