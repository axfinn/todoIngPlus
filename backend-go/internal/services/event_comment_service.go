package services

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/axfinn/todoIngPlus/backend-go/internal/models"
)

// EventCommentService 管理事件评论/时间线
type EventCommentService struct {
	db      *mongo.Database
	coll    *mongo.Collection
	eventCo *mongo.Collection
}

func NewEventCommentService(db *mongo.Database) *EventCommentService {
	return &EventCommentService{db: db, coll: db.Collection("event_comments"), eventCo: db.Collection("events")}
}

// AddComment 创建评论
type AddCommentOptions struct {
	AllowEmpty bool
}

func (s *EventCommentService) AddComment(ctx context.Context, userID, eventID primitive.ObjectID, req models.CreateEventCommentRequest) (*models.EventComment, error) {
	if req.Content == "" {
		return nil, errors.New("content empty")
	}
	// 校验事件存在
	if err := s.eventCo.FindOne(ctx, bson.M{"_id": eventID, "user_id": userID}).Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("event not found or no permission")
		}
		return nil, err
	}
	now := time.Now()
	c := &models.EventComment{ID: primitive.NewObjectID(), EventID: eventID, UserID: userID, Type: req.Type}
	if c.Type == "" {
		c.Type = "comment"
	}
	c.Content = req.Content
	c.Meta = req.Meta
	c.CreatedAt = now
	c.UpdatedAt = now
	if _, err := s.coll.InsertOne(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// UpdateComment 仅作者且在创建后一定窗口内允许编辑（例如 10 分钟）
func (s *EventCommentService) UpdateComment(ctx context.Context, userID, commentID primitive.ObjectID, req models.UpdateEventCommentRequest) (*models.EventComment, error) {
	var ec models.EventComment
	if err := s.coll.FindOne(ctx, bson.M{"_id": commentID, "user_id": userID}).Decode(&ec); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("comment not found or no permission")
		}
		return nil, err
	}
	if ec.Type != "comment" {
		return nil, errors.New("only normal comment editable")
	}
	if time.Since(ec.CreatedAt) > 10*time.Minute {
		return nil, errors.New("edit window expired")
	}
	upd := bson.M{}
	if req.Content != nil && *req.Content != "" {
		upd["content"] = *req.Content
	}
	if len(upd) == 0 {
		return &ec, nil
	}
	upd["updated_at"] = time.Now()
	if _, err := s.coll.UpdateByID(ctx, ec.ID, bson.M{"$set": upd}); err != nil {
		return nil, err
	}
	if err := s.coll.FindOne(ctx, bson.M{"_id": ec.ID}).Decode(&ec); err != nil {
		return nil, err
	}
	return &ec, nil
}

// DeleteComment 仅作者或事件拥有者可删除
func (s *EventCommentService) DeleteComment(ctx context.Context, userID primitive.ObjectID, commentID primitive.ObjectID) error {
	var ec models.EventComment
	if err := s.coll.FindOne(ctx, bson.M{"_id": commentID}).Decode(&ec); err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.New("comment not found")
		}
		return err
	}
	var ev models.Event
	if err := s.eventCo.FindOne(ctx, bson.M{"_id": ec.EventID}).Decode(&ev); err != nil {
		return err
	}
	if ec.UserID != userID && ev.UserID != userID {
		return errors.New("no permission")
	}
	_, err := s.coll.DeleteOne(ctx, bson.M{"_id": ec.ID})
	return err
}

// ListTimeline 列出事件时间线（按创建时间升序）
func (s *EventCommentService) ListTimeline(ctx context.Context, userID, eventID primitive.ObjectID, limit int, before *primitive.ObjectID) ([]models.EventTimelineItem, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	filter := bson.M{"event_id": eventID}
	if before != nil {
		filter["_id"] = bson.M{"$lt": *before}
	}
	opts := options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}).SetLimit(int64(limit))
	cur, err := s.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []models.EventTimelineItem
	for cur.Next(ctx) {
		var ec models.EventComment
		if err := cur.Decode(&ec); err == nil {
			out = append(out, models.EventTimelineItem{ID: ec.ID, EventID: ec.EventID, UserID: ec.UserID, Type: ec.Type, Content: ec.Content, Meta: ec.Meta, CreatedAt: ec.CreatedAt, UpdatedAt: ec.UpdatedAt})
		}
	}
	return out, nil
}
