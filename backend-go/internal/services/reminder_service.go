package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/axfinn/todoIng/backend-go/internal/models"
)

// ReminderService 提醒服务
type ReminderService struct {
	db           *mongo.Database
	reminderColl *mongo.Collection
	eventColl    *mongo.Collection
}

// NewReminderService 创建提醒服务
func NewReminderService(db *mongo.Database) *ReminderService {
	return &ReminderService{
		db:           db,
		reminderColl: db.Collection("reminders"),
		eventColl:    db.Collection("events"),
	}
}

// CreateReminder 创建提醒
func (s *ReminderService) CreateReminder(ctx context.Context, userID primitive.ObjectID, req models.CreateReminderRequest) (*models.Reminder, error) {
	// 验证事件是否存在且属于当前用户
	var event models.Event
	err := s.eventColl.FindOne(ctx, bson.M{
		"_id":     req.EventID,
		"user_id": userID,
	}).Decode(&event)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("event not found")
		}
		return nil, fmt.Errorf("failed to find event: %w", err)
	}

	now := time.Now()
	
	reminder := &models.Reminder{
		ID:            primitive.NewObjectID(),
		EventID:       req.EventID,
		UserID:        userID,
		AdvanceDays:   req.AdvanceDays,
		ReminderTimes: req.ReminderTimes,
		ReminderType:  req.ReminderType,
		CustomMessage: req.CustomMessage,
		IsActive:      true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// 计算下次发送时间
	nextSend := reminder.CalculateNextSendTime(event)
	reminder.NextSend = nextSend

	_, err = s.reminderColl.InsertOne(ctx, reminder)
	if err != nil {
		return nil, fmt.Errorf("failed to create reminder: %w", err)
	}

	return reminder, nil
}

// GetReminder 获取单个提醒
func (s *ReminderService) GetReminder(ctx context.Context, userID, reminderID primitive.ObjectID) (*models.ReminderWithEvent, error) {
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"_id":     reminderID,
				"user_id": userID,
			},
		},
		{
			"$lookup": bson.M{
				"from":         "events",
				"localField":   "event_id",
				"foreignField": "_id",
				"as":           "event",
			},
		},
		{
			"$unwind": "$event",
		},
	}

	cursor, err := s.reminderColl.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get reminder: %w", err)
	}
	defer cursor.Close(ctx)

	var results []models.ReminderWithEvent
	if err = cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode reminder: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("reminder not found")
	}

	return &results[0], nil
}

// UpdateReminder 更新提醒
func (s *ReminderService) UpdateReminder(ctx context.Context, userID, reminderID primitive.ObjectID, req models.UpdateReminderRequest) (*models.Reminder, error) {
	filter := bson.M{
		"_id":     reminderID,
		"user_id": userID,
	}

	update := bson.M{
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}

	// 只更新提供的字段
	setFields := update["$set"].(bson.M)
	if req.AdvanceDays != nil {
		setFields["advance_days"] = *req.AdvanceDays
	}
	if req.ReminderTimes != nil {
		setFields["reminder_times"] = req.ReminderTimes
	}
	if req.ReminderType != nil {
		setFields["reminder_type"] = *req.ReminderType
	}
	if req.CustomMessage != nil {
		setFields["custom_message"] = *req.CustomMessage
	}
	if req.IsActive != nil {
		setFields["is_active"] = *req.IsActive
	}

	result, err := s.reminderColl.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, fmt.Errorf("failed to update reminder: %w", err)
	}

	if result.MatchedCount == 0 {
		return nil, fmt.Errorf("reminder not found")
	}

	// 重新计算下次发送时间
	var reminder models.Reminder
	err = s.reminderColl.FindOne(ctx, filter).Decode(&reminder)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated reminder: %w", err)
	}

	// 获取关联的事件
	var event models.Event
	err = s.eventColl.FindOne(ctx, bson.M{"_id": reminder.EventID}).Decode(&event)
	if err == nil {
		nextSend := reminder.CalculateNextSendTime(event)
		_, err = s.reminderColl.UpdateOne(ctx, 
			bson.M{"_id": reminderID},
			bson.M{"$set": bson.M{"next_send": nextSend}},
		)
		if err == nil {
			reminder.NextSend = nextSend
		}
	}

	return &reminder, nil
}

// DeleteReminder 删除提醒
func (s *ReminderService) DeleteReminder(ctx context.Context, userID, reminderID primitive.ObjectID) error {
	filter := bson.M{
		"_id":     reminderID,
		"user_id": userID,
	}

	result, err := s.reminderColl.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete reminder: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("reminder not found")
	}

	return nil
}

// ListReminders 获取提醒列表
func (s *ReminderService) ListReminders(ctx context.Context, userID primitive.ObjectID, page, pageSize int, activeOnly bool) (*models.ReminderListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	matchFilter := bson.M{
		"user_id": userID,
	}
	if activeOnly {
		matchFilter["is_active"] = true
	}

	// 聚合查询，包含事件信息
	pipeline := []bson.M{
		{"$match": matchFilter},
		{
			"$lookup": bson.M{
				"from":         "events",
				"localField":   "event_id",
				"foreignField": "_id",
				"as":           "event",
			},
		},
		{"$unwind": "$event"},
	}

	// 计算总数
	countPipeline := append(pipeline, bson.M{"$count": "total"})
	countCursor, err := s.reminderColl.Aggregate(ctx, countPipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to count reminders: %w", err)
	}
	defer countCursor.Close(ctx)

	var countResult []bson.M
	if err = countCursor.All(ctx, &countResult); err != nil {
		return nil, fmt.Errorf("failed to get count: %w", err)
	}

	var total int64
	if len(countResult) > 0 {
		total = int64(countResult[0]["total"].(int32))
	}

	// 分页查询
	skip := (page - 1) * pageSize
	pipeline = append(pipeline, 
		bson.M{"$sort": bson.M{"created_at": -1}},
		bson.M{"$skip": skip},
		bson.M{"$limit": pageSize},
	)

	cursor, err := s.reminderColl.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to find reminders: %w", err)
	}
	defer cursor.Close(ctx)

	var reminders []models.ReminderWithEvent
	if err = cursor.All(ctx, &reminders); err != nil {
		return nil, fmt.Errorf("failed to decode reminders: %w", err)
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	return &models.ReminderListResponse{
		Reminders:  reminders,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// GetUpcomingReminders 获取即将到来的提醒
func (s *ReminderService) GetUpcomingReminders(ctx context.Context, userID primitive.ObjectID, hours int) ([]models.UpcomingReminder, error) {
	if hours <= 0 {
		hours = 24 // 默认24小时
	}

	now := time.Now()
	endTime := now.Add(time.Duration(hours) * time.Hour)

	pipeline := []bson.M{
		{
			"$match": bson.M{
				"user_id":   userID,
				"is_active": true,
				"next_send": bson.M{
					"$gte": now,
					"$lte": endTime,
				},
			},
		},
		{
			"$lookup": bson.M{
				"from":         "events",
				"localField":   "event_id",
				"foreignField": "_id",
				"as":           "event",
			},
		},
		{"$unwind": "$event"},
		{
			"$project": bson.M{
				"id":          "$_id",
				"event_title": "$event.title",
				"event_date":  "$event.event_date",
				"message":     bson.M{
					"$cond": bson.A{
						bson.M{"$ne": []interface{}{"$custom_message", ""}},
						"$custom_message",
						bson.M{"$concat": []string{"提醒: ", "$event.title"}},
					},
				},
				"days_left": bson.M{
					"$toInt": bson.M{
						"$divide": []interface{}{
							bson.M{"$subtract": []interface{}{"$event.event_date", now}},
							86400000, // 毫秒转天数
						},
					},
				},
				"importance":  "$event.importance_level",
				"reminder_at": "$next_send",
			},
		},
		{"$sort": bson.M{"reminder_at": 1}},
	}

	cursor, err := s.reminderColl.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to find upcoming reminders: %w", err)
	}
	defer cursor.Close(ctx)

	var reminders []models.UpcomingReminder
	if err = cursor.All(ctx, &reminders); err != nil {
		return nil, fmt.Errorf("failed to decode upcoming reminders: %w", err)
	}

	return reminders, nil
}

// GetPendingReminders 获取待发送的提醒
func (s *ReminderService) GetPendingReminders(ctx context.Context) ([]models.ReminderWithEvent, error) {
	now := time.Now()

	pipeline := []bson.M{
		{
			"$match": bson.M{
				"is_active": true,
				"next_send": bson.M{
					"$lte": now,
				},
			},
		},
		{
			"$lookup": bson.M{
				"from":         "events",
				"localField":   "event_id",
				"foreignField": "_id",
				"as":           "event",
			},
		},
		{"$unwind": "$event"},
		{"$sort": bson.M{"next_send": 1}},
	}

	cursor, err := s.reminderColl.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to find pending reminders: %w", err)
	}
	defer cursor.Close(ctx)

	var reminders []models.ReminderWithEvent
	if err = cursor.All(ctx, &reminders); err != nil {
		return nil, fmt.Errorf("failed to decode pending reminders: %w", err)
	}

	return reminders, nil
}

// MarkReminderSent 标记提醒已发送
func (s *ReminderService) MarkReminderSent(ctx context.Context, reminderID primitive.ObjectID) error {
	now := time.Now()
	
	// 获取提醒和事件信息
	var reminder models.Reminder
	err := s.reminderColl.FindOne(ctx, bson.M{"_id": reminderID}).Decode(&reminder)
	if err != nil {
		return fmt.Errorf("failed to find reminder: %w", err)
	}

	var event models.Event
	err = s.eventColl.FindOne(ctx, bson.M{"_id": reminder.EventID}).Decode(&event)
	if err != nil {
		return fmt.Errorf("failed to find event: %w", err)
	}

	// 计算下次发送时间
	nextSend := reminder.CalculateNextSendTime(event)

	update := bson.M{
		"$set": bson.M{
			"last_sent":  now,
			"next_send":  nextSend,
			"updated_at": now,
		},
	}

	_, err = s.reminderColl.UpdateOne(ctx, bson.M{"_id": reminderID}, update)
	if err != nil {
		return fmt.Errorf("failed to mark reminder as sent: %w", err)
	}

	return nil
}

// SnoozeReminder 暂停提醒（推迟指定时间）
func (s *ReminderService) SnoozeReminder(ctx context.Context, userID, reminderID primitive.ObjectID, snoozeMinutes int) error {
	if snoozeMinutes <= 0 {
		snoozeMinutes = 60 // 默认推迟1小时
	}

	filter := bson.M{
		"_id":     reminderID,
		"user_id": userID,
	}

	newNextSend := time.Now().Add(time.Duration(snoozeMinutes) * time.Minute)
	
	update := bson.M{
		"$set": bson.M{
			"next_send":  newNextSend,
			"updated_at": time.Now(),
		},
	}

	result, err := s.reminderColl.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to snooze reminder: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("reminder not found")
	}

	return nil
}
