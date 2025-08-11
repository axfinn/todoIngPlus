package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/axfinn/todoIngPlus/backend-go/internal/models"
)

// EventService 事件服务
type EventService struct {
	db             *mongo.Database
	eventColl      *mongo.Collection
	reminderColl   *mongo.Collection
	taskColl       *mongo.Collection
	sortConfigColl *mongo.Collection
}

// NewEventService 创建事件服务
func NewEventService(db *mongo.Database) *EventService {
	return &EventService{
		db:             db,
		eventColl:      db.Collection("events"),
		reminderColl:   db.Collection("reminders"),
		taskColl:       db.Collection("tasks"),
		sortConfigColl: db.Collection("task_sort_configs"),
	}
}

// CreateEvent 创建事件
func (s *EventService) CreateEvent(ctx context.Context, userID primitive.ObjectID, req models.CreateEventRequest) (*models.Event, error) {
	now := time.Now()

	event := &models.Event{
		ID:               primitive.NewObjectID(),
		UserID:           userID,
		Title:            req.Title,
		Description:      req.Description,
		EventType:        req.EventType,
		EventDate:        req.EventDate,
		RecurrenceType:   req.RecurrenceType,
		RecurrenceConfig: req.RecurrenceConfig,
		ImportanceLevel:  req.ImportanceLevel,
		Tags:             req.Tags,
		Location:         req.Location,
		IsAllDay:         req.IsAllDay,
		CreatedAt:        now,
		UpdatedAt:        now,
		IsActive:         true,
	}

	// 设置默认值
	if event.RecurrenceType == "" {
		event.RecurrenceType = "none"
	}
	if event.ImportanceLevel == 0 {
		event.ImportanceLevel = 3
	}
	if event.Tags == nil {
		event.Tags = []string{}
	}

	_, err := s.eventColl.InsertOne(ctx, event)
	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	// 同步写入创建系统时间线条目，便于前端立即看到
	func() {
		defer func() { recover() }()
		_, _ = s.db.Collection("event_comments").InsertOne(ctx, bson.M{
			"_id":        primitive.NewObjectID(),
			"event_id":   event.ID,
			"user_id":    event.UserID,
			"type":       "system",
			"content":    "created",
			"meta":       bson.M{"action": "create", "date": event.EventDate.Format(time.RFC3339)},
			"created_at": time.Now(),
			"updated_at": time.Now(),
		})
	}()

	return event, nil
}

// GetEvent 获取单个事件
func (s *EventService) GetEvent(ctx context.Context, userID, eventID primitive.ObjectID) (*models.Event, error) {
	var event models.Event

	filter := bson.M{
		"_id":     eventID,
		"user_id": userID,
	}

	err := s.eventColl.FindOne(ctx, filter).Decode(&event)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("event not found")
		}
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	return &event, nil
}

// UpdateEvent 更新事件
func (s *EventService) UpdateEvent(ctx context.Context, userID, eventID primitive.ObjectID, req models.UpdateEventRequest) (*models.Event, error) {
	filter := bson.M{
		"_id":     eventID,
		"user_id": userID,
	}

	update := bson.M{
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}

	// 只更新提供的字段
	setFields := update["$set"].(bson.M)
	if req.Title != nil {
		setFields["title"] = *req.Title
	}
	if req.Description != nil {
		setFields["description"] = *req.Description
	}
	if req.EventType != nil {
		setFields["event_type"] = *req.EventType
	}
	if req.EventDate != nil {
		setFields["event_date"] = *req.EventDate
	}
	if req.RecurrenceType != nil {
		setFields["recurrence_type"] = *req.RecurrenceType
	}
	if req.RecurrenceConfig != nil {
		setFields["recurrence_config"] = req.RecurrenceConfig
	}
	if req.ImportanceLevel != nil {
		setFields["importance_level"] = *req.ImportanceLevel
	}
	if req.Tags != nil {
		setFields["tags"] = req.Tags
	}
	if req.Location != nil {
		setFields["location"] = *req.Location
	}
	if req.IsAllDay != nil {
		setFields["is_all_day"] = *req.IsAllDay
	}
	if req.IsActive != nil {
		setFields["is_active"] = *req.IsActive
	}

	result, err := s.eventColl.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, fmt.Errorf("failed to update event: %w", err)
	}

	if result.MatchedCount == 0 {
		return nil, fmt.Errorf("event not found")
	}

	// 返回更新后的事件
	return s.GetEvent(ctx, userID, eventID)
}

// DeleteEvent 删除事件
func (s *EventService) DeleteEvent(ctx context.Context, userID, eventID primitive.ObjectID) error {
	filter := bson.M{
		"_id":     eventID,
		"user_id": userID,
	}

	result, err := s.eventColl.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("event not found")
	}

	// 同时删除相关的提醒
	_, err = s.reminderColl.DeleteMany(ctx, bson.M{"event_id": eventID})
	if err != nil {
		// 记录错误但不阻止事件删除
		fmt.Printf("Warning: failed to delete related reminders: %v\n", err)
	}

	return nil
}

// AdvanceEvent 推进事件到下一周期（若为非重复则标记为不活跃）
func (s *EventService) AdvanceEvent(ctx context.Context, userID, eventID primitive.ObjectID, reason string) (*models.Event, error) {
	// 获取当前事件
	event, err := s.GetEvent(ctx, userID, eventID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	update := bson.M{"updated_at": now}
	var oldDate = event.EventDate

	if event.RecurrenceType == "none" { // 一次性事件：标记完成
		update["is_active"] = false
		// 可选：记录 last_triggered_at
		update["last_triggered_at"] = now
	} else {
		// 手动推进：始终在当前 event_date 基础上 +1 周期（符合用户“下一次”预期）
		var next time.Time
		switch event.RecurrenceType {
		case "daily":
			next = event.EventDate.AddDate(0, 0, 1)
		case "weekly":
			next = event.EventDate.AddDate(0, 0, 7)
		case "monthly":
			next = event.EventDate.AddDate(0, 1, 0)
		case "yearly":
			next = event.EventDate.AddDate(1, 0, 0)
		default:
			// 未知类型，保守：标记 inactive
			update["is_active"] = false
		}
		if !next.IsZero() {
			// 若用户重复快速点击，确保 next 在现在之后；若仍在过去，循环推进到未来
			for next.Before(now) { // 理论上只在过去日期数据异常时出现
				switch event.RecurrenceType {
				case "daily":
					next = next.AddDate(0, 0, 1)
				case "weekly":
					next = next.AddDate(0, 0, 7)
				case "monthly":
					next = next.AddDate(0, 1, 0)
				case "yearly":
					next = next.AddDate(1, 0, 0)
				default:
					break
				}
			}
			update["event_date"] = next
			update["last_triggered_at"] = now
		}
	}

	_, err = s.eventColl.UpdateOne(ctx, bson.M{"_id": event.ID, "user_id": userID}, bson.M{"$set": update})
	if err != nil {
		return nil, fmt.Errorf("failed to advance event: %w", err)
	}
	updated, err := s.GetEvent(ctx, userID, eventID)
	if err != nil {
		return nil, err
	}

	// 重新计算相关提醒 next_send（非阻断，错误忽略）
	go func(ev *models.Event) {
		defer func() { recover() }()
		cur, err2 := s.reminderColl.Find(context.Background(), bson.M{"event_id": ev.ID, "is_active": true})
		if err2 != nil {
			return
		}
		defer cur.Close(context.Background())
		for cur.Next(context.Background()) {
			var r models.Reminder
			if err := cur.Decode(&r); err == nil {
				// 仅当未配置绝对时间时才基于事件时间重新计算
				if len(r.AbsoluteTimes) == 0 {
					nx := r.CalculateNextSendTime(*ev)
					upd := bson.M{"updated_at": time.Now()}
					if nx != nil {
						upd["next_send"] = *nx
					} else {
						upd["next_send"] = nil
					}
					_, _ = s.reminderColl.UpdateByID(context.Background(), r.ID, bson.M{"$set": upd})
				}
			}
		}
	}(updated)

	// 同步写入系统时间线条目，确保前端立即可见
	func(oldT time.Time, newEv *models.Event, reason string) {
		defer func() { recover() }()
		content := "advanced"
		meta := map[string]string{
			"action":     "advance",
			"old":        oldT.Format(time.RFC3339),
			"recurrence": newEv.RecurrenceType,
			"active":     fmt.Sprintf("%v", newEv.IsActive),
		}
		if reason != "" {
			meta["reason"] = reason
		}
		if newEv.EventDate.After(oldT) {
			meta["new"] = newEv.EventDate.Format(time.RFC3339)
		}
		joined := content
		if reason != "" {
			joined += ": " + reason
		}
		_, _ = s.db.Collection("event_comments").InsertOne(ctx, bson.M{
			"_id":        primitive.NewObjectID(),
			"event_id":   newEv.ID,
			"user_id":    newEv.UserID,
			"type":       "system",
			"content":    joined,
			"meta":       meta,
			"created_at": time.Now(),
			"updated_at": time.Now(),
		})
	}(oldDate, updated, reason)

	return updated, nil
}

// ListEvents 获取事件列表
func (s *EventService) ListEvents(ctx context.Context, userID primitive.ObjectID, page, pageSize int, eventType string, startDate, endDate *time.Time) (*models.EventListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	filter := bson.M{
		"user_id":   userID,
		"is_active": true,
	}

	// 事件类型过滤
	if eventType != "" {
		filter["event_type"] = eventType
	}

	// 日期范围过滤
	if startDate != nil || endDate != nil {
		dateFilter := bson.M{}
		if startDate != nil {
			dateFilter["$gte"] = *startDate
		}
		if endDate != nil {
			dateFilter["$lte"] = *endDate
		}
		filter["event_date"] = dateFilter
	}

	// 计算总数
	total, err := s.eventColl.CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count events: %w", err)
	}

	// 分页查询
	skip := (page - 1) * pageSize
	opts := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize)).
		SetSort(bson.D{{Key: "event_date", Value: 1}})

	cursor, err := s.eventColl.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find events: %w", err)
	}
	defer cursor.Close(ctx)

	var events []models.Event
	if err = cursor.All(ctx, &events); err != nil {
		return nil, fmt.Errorf("failed to decode events: %w", err)
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	return &models.EventListResponse{
		Events:     events,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// GetUpcomingEvents 获取即将到来的事件
func (s *EventService) GetUpcomingEvents(ctx context.Context, userID primitive.ObjectID, days int) ([]models.Event, error) {
	if days <= 0 {
		days = 7 // 默认7天
	}

	now := time.Now()
	endDate := now.AddDate(0, 0, days)

	filter := bson.M{
		"user_id":   userID,
		"is_active": true,
		"event_date": bson.M{
			"$gte": now,
			"$lte": endDate,
		},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "event_date", Value: 1}}).
		SetLimit(50) // 限制最多50个

	cursor, err := s.eventColl.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find upcoming events: %w", err)
	}
	defer cursor.Close(ctx)

	var events []models.Event
	if err = cursor.All(ctx, &events); err != nil {
		return nil, fmt.Errorf("failed to decode upcoming events: %w", err)
	}

	// 处理循环事件
	var allEvents []models.Event
	for _, event := range events {
		if event.RecurrenceType != "none" {
			nextOccurrence := event.GetNextOccurrence(now)
			if nextOccurrence != nil && nextOccurrence.Before(endDate) {
				// 创建下次发生的事件实例
				eventCopy := event
				eventCopy.EventDate = *nextOccurrence
				allEvents = append(allEvents, eventCopy)
			}
		} else {
			allEvents = append(allEvents, event)
		}
	}

	return allEvents, nil
}

// GetCalendarEvents 获取日历视图事件
func (s *EventService) GetCalendarEvents(ctx context.Context, userID primitive.ObjectID, year, month int) (map[string][]models.Event, error) {
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, -1).Add(24*time.Hour - time.Nanosecond)

	filter := bson.M{
		"user_id":   userID,
		"is_active": true,
		"event_date": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
	}

	cursor, err := s.eventColl.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find calendar events: %w", err)
	}
	defer cursor.Close(ctx)

	var events []models.Event
	if err = cursor.All(ctx, &events); err != nil {
		return nil, fmt.Errorf("failed to decode calendar events: %w", err)
	}

	// 按日期分组
	calendar := make(map[string][]models.Event)
	for _, event := range events {
		dateKey := event.EventDate.Format("2006-01-02")
		calendar[dateKey] = append(calendar[dateKey], event)
	}

	return calendar, nil
}

// SearchEvents 搜索事件
func (s *EventService) SearchEvents(ctx context.Context, userID primitive.ObjectID, keyword string, limit int) ([]models.Event, error) {
	if keyword == "" {
		return []models.Event{}, nil
	}

	if limit <= 0 {
		limit = 20
	}

	filter := bson.M{
		"user_id":   userID,
		"is_active": true,
		"$or": []bson.M{
			{"title": bson.M{"$regex": keyword, "$options": "i"}},
			{"description": bson.M{"$regex": keyword, "$options": "i"}},
			{"tags": bson.M{"$in": []string{keyword}}},
		},
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "event_date", Value: 1}})

	cursor, err := s.eventColl.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to search events: %w", err)
	}
	defer cursor.Close(ctx)

	var events []models.Event
	if err = cursor.All(ctx, &events); err != nil {
		return nil, fmt.Errorf("failed to decode search results: %w", err)
	}

	return events, nil
}
