/*
 * @Author: error: error: git config user.name & please set dead value or install git && error: git config user.email & please set dead value or install git & please set dead value or install git
 * @Date: 2025-08-10 17:52:53
 * @LastEditors: error: error: git config user.name & please set dead value or install git && error: git config user.email & please set dead value or install git & please set dead value or install git
 * @LastEditTime: 2025-08-10 20:44:01
 * @FilePath: /todoIngPlus/backend-go/internal/services/notification_service.go
 * @Description: 这是默认设置,请设置`customMade`, 打开koroFileHeader查看配置 进行设置: https://github.com/OBKoro1/koro1FileHeader/wiki/%E9%85%8D%E7%BD%AE
 */
package services

import (
	"context"
	"errors"
	"time"

	"github.com/axfinn/todoIngPlus/backend-go/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type NotificationService struct{ db *mongo.Database }

func NewNotificationService(db *mongo.Database) *NotificationService {
	return &NotificationService{db: db}
}

func (s *NotificationService) collection() *mongo.Collection { return s.db.Collection("notifications") }

// Create 插入通知
func (s *NotificationService) Create(ctx context.Context, in models.NotificationCreate) (models.Notification, error) {
	n := models.Notification{
		ID:        primitive.NewObjectID(),
		UserID:    in.UserID,
		Type:      in.Type,
		Message:   in.Message,
		EventID:   in.EventID,
		CreatedAt: time.Now(),
		Metadata:  in.Metadata,
	}
	if _, err := s.collection().InsertOne(ctx, n); err != nil {
		return models.Notification{}, err
	}
	return n, nil
}

// List 列出通知 (倒序) 支持未读过滤
func (s *NotificationService) List(ctx context.Context, userID primitive.ObjectID, unreadOnly bool, limit int) ([]models.Notification, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	filter := bson.M{"user_id": userID}
	if unreadOnly {
		filter["read_at"] = bson.M{"$exists": false}
	}
	findOpts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(int64(limit))
	cur, err := s.collection().Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []models.Notification
	for cur.Next(ctx) {
		var n models.Notification
		if err := cur.Decode(&n); err == nil {
			out = append(out, n)
		}
	}
	return out, cur.Err()
}

// MarkRead 标记某条通知已读
func (s *NotificationService) MarkRead(ctx context.Context, userID, id primitive.ObjectID) error {
	now := time.Now()
	res, err := s.collection().UpdateOne(ctx, bson.M{"_id": id, "user_id": userID, "read_at": bson.M{"$exists": false}}, bson.M{"$set": bson.M{"read_at": now}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return errors.New("not_found")
	}
	return nil
}

// MarkAllRead 标记用户所有未读为已读
func (s *NotificationService) MarkAllRead(ctx context.Context, userID primitive.ObjectID) (int64, error) {
	now := time.Now()
	res, err := s.collection().UpdateMany(ctx, bson.M{"user_id": userID, "read_at": bson.M{"$exists": false}}, bson.M{"$set": bson.M{"read_at": now}})
	if err != nil {
		return 0, err
	}
	return res.ModifiedCount, nil
}

func int64Ptr(v int64) *int64 { return &v }
