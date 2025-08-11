package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/axfinn/todoIngPlus/backend-go/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// EventRepository 事件仓储，封装所有 Mongo 访问与聚合逻辑
// 注意：这里允许跨集合操作（写时间线/更新提醒），上层 Service 不再引用 mongo/bson
//
// Advance 会：
// 1. 计算下次日期或结束事件
// 2. 更新事件
// 3. 异步重算相关提醒 next_send
// 4. 写 event_comments 系统时间线
//
type EventRepository interface {
	Insert(ctx context.Context, e *models.Event) error
	FindByID(ctx context.Context, userID, id primitive.ObjectID) (*models.Event, error)
	UpdateFields(ctx context.Context, userID, id primitive.ObjectID, set map[string]interface{}) (*models.Event, error)
	Delete(ctx context.Context, userID, id primitive.ObjectID) error
	Count(ctx context.Context, userID primitive.ObjectID, eventType string) (int64, error)
	ListPaged(ctx context.Context, userID primitive.ObjectID, page, pageSize int, eventType string, startDate, endDate *time.Time) (*models.EventListResponse, error)
	ListUpcoming(ctx context.Context, userID primitive.ObjectID, days int) ([]models.Event, error)
	CalendarRange(ctx context.Context, userID primitive.ObjectID, year, month int) ([]models.Event, error)
	Search(ctx context.Context, userID primitive.ObjectID, keyword string, limit int) ([]models.Event, error)
	Advance(ctx context.Context, userID, eventID primitive.ObjectID, reason string) (*models.Event, error)
	ListStartingWindow(ctx context.Context, from, to time.Time) ([]models.Event, error)
	MarkTriggered(ctx context.Context, id primitive.ObjectID, ts time.Time) error
}

type mongoEventRepo struct { db *mongo.Database }

func NewEventRepository(db *mongo.Database) EventRepository { return &mongoEventRepo{db: db} }

func (r *mongoEventRepo) coll() *mongo.Collection { return r.db.Collection("events") }

func (r *mongoEventRepo) reminders() *mongo.Collection { return r.db.Collection("reminders") }

func (r *mongoEventRepo) comments() *mongo.Collection { return r.db.Collection("event_comments") }

func (r *mongoEventRepo) Insert(ctx context.Context, e *models.Event) error {
	if e == nil { return errors.New("nil event") }
	if e.ID.IsZero() { e.ID = primitive.NewObjectID() }
	now := time.Now()
	if e.CreatedAt.IsZero() { e.CreatedAt = now }
	e.UpdatedAt = e.CreatedAt
	if e.RecurrenceType == "" { e.RecurrenceType = "none" }
	if e.ImportanceLevel == 0 { e.ImportanceLevel = 3 }
	if e.Tags == nil { e.Tags = []string{} }
	_, err := r.coll().InsertOne(ctx, e)
	if err != nil { return err }
	// 系统时间线
	go func(ev *models.Event){
		defer func(){ recover() }()
		_, _ = r.comments().InsertOne(context.Background(), bson.M{
			"_id": primitive.NewObjectID(),
			"event_id": ev.ID,
			"user_id": ev.UserID,
			"type": "system",
			"content": "created",
			"meta": bson.M{"action": "create", "date": ev.EventDate.Format(time.RFC3339)},
			"created_at": time.Now(),
			"updated_at": time.Now(),
		})
	}(e)
	return nil
}

func (r *mongoEventRepo) FindByID(ctx context.Context, userID, id primitive.ObjectID) (*models.Event, error) {
	var ev models.Event
	err := r.coll().FindOne(ctx, bson.M{"_id": id, "user_id": userID}).Decode(&ev)
	if err != nil { return nil, err }
	return &ev, nil
}

func (r *mongoEventRepo) UpdateFields(ctx context.Context, userID, id primitive.ObjectID, set map[string]interface{}) (*models.Event, error) {
	if set == nil { set = map[string]interface{}{} }
	set["updated_at"] = time.Now()
	bset := bson.M{}
	for k, v := range set { bset[k] = v }
	res, err := r.coll().UpdateOne(ctx, bson.M{"_id": id, "user_id": userID}, bson.M{"$set": bset})
	if err != nil { return nil, err }
	if res.MatchedCount == 0 { return nil, errors.New("not found") }
	return r.FindByID(ctx, userID, id)
}

func (r *mongoEventRepo) Delete(ctx context.Context, userID, id primitive.ObjectID) error {
	res, err := r.coll().DeleteOne(ctx, bson.M{"_id": id, "user_id": userID})
	if err != nil { return err }
	if res.DeletedCount == 0 { return errors.New("not found") }
	// 级联删除提醒（忽略错误）
	go func(){ _, _ = r.reminders().DeleteMany(context.Background(), bson.M{"event_id": id}) }()
	return nil
}

func (r *mongoEventRepo) Count(ctx context.Context, userID primitive.ObjectID, eventType string) (int64, error) {
	filter := bson.M{"user_id": userID, "is_active": true}
	if eventType != "" { filter["event_type"] = eventType }
	return r.coll().CountDocuments(ctx, filter)
}

func (r *mongoEventRepo) ListPaged(ctx context.Context, userID primitive.ObjectID, page, pageSize int, eventType string, startDate, endDate *time.Time) (*models.EventListResponse, error) {
	if page < 1 { page = 1 }
	if pageSize < 1 || pageSize > 100 { pageSize = 20 }
	filter := bson.M{"user_id": userID, "is_active": true}
	if eventType != "" { filter["event_type"] = eventType }
	if startDate != nil || endDate != nil {
		dateFilter := bson.M{}
		if startDate != nil { dateFilter["$gte"] = *startDate }
		if endDate != nil { dateFilter["$lte"] = *endDate }
		filter["event_date"] = dateFilter
	}
	total, err := r.coll().CountDocuments(ctx, filter)
	if err != nil { return nil, err }
	skip := (page - 1) * pageSize
	opts := options.Find().SetSkip(int64(skip)).SetLimit(int64(pageSize)).SetSort(bson.D{{Key: "event_date", Value: 1}})
	cur, err := r.coll().Find(ctx, filter, opts)
	if err != nil { return nil, err }
	defer cur.Close(ctx)
	var events []models.Event
	if err = cur.All(ctx, &events); err != nil { return nil, err }
	pages := int(math.Ceil(float64(total) / float64(pageSize)))
	return &models.EventListResponse{Events: events, Total: total, Page: page, PageSize: pageSize, TotalPages: pages}, nil
}

func (r *mongoEventRepo) ListUpcoming(ctx context.Context, userID primitive.ObjectID, days int) ([]models.Event, error) {
	if days <= 0 { days = 7 }
	now := time.Now()
	endDate := now.AddDate(0,0,days)
	filter := bson.M{"user_id": userID, "is_active": true, "event_date": bson.M{"$gte": now, "$lte": endDate}}
	cur, err := r.coll().Find(ctx, filter, options.Find().SetSort(bson.D{{Key:"event_date", Value:1}}).SetLimit(50))
	if err != nil { return nil, err }
	defer cur.Close(ctx)
	var list []models.Event
	if err = cur.All(ctx, &list); err != nil { return nil, err }
	// 处理循环，将下一次实例化（保持与原逻辑一致）
	var out []models.Event
	for _, ev := range list {
		if ev.RecurrenceType != "none" {
			if next := ev.GetNextOccurrence(now); next != nil && next.Before(endDate) {
				cpy := ev; cpy.EventDate = *next; out = append(out, cpy)
			} else { out = append(out, ev) }
		} else { out = append(out, ev) }
	}
	return out, nil
}

func (r *mongoEventRepo) CalendarRange(ctx context.Context, userID primitive.ObjectID, year, month int) ([]models.Event, error) {
	start := time.Date(year, time.Month(month), 1, 0,0,0,0, time.UTC)
	end := start.AddDate(0,1,-1).Add(24*time.Hour - time.Nanosecond)
	filter := bson.M{"user_id": userID, "is_active": true, "event_date": bson.M{"$gte": start, "$lte": end}}
	cur, err := r.coll().Find(ctx, filter)
	if err != nil { return nil, err }
	defer cur.Close(ctx)
	var list []models.Event
	if err = cur.All(ctx, &list); err != nil { return nil, err }
	return list, nil
}

func (r *mongoEventRepo) Search(ctx context.Context, userID primitive.ObjectID, keyword string, limit int) ([]models.Event, error) {
	if keyword == "" { return []models.Event{}, nil }
	if limit <= 0 { limit = 20 }
	filter := bson.M{"user_id": userID, "is_active": true, "$or": []bson.M{{"title": bson.M{"$regex": keyword, "$options": "i"}}, {"description": bson.M{"$regex": keyword, "$options": "i"}}, {"tags": bson.M{"$in": []string{keyword}}}}}
	cur, err := r.coll().Find(ctx, filter, options.Find().SetLimit(int64(limit)).SetSort(bson.D{{Key:"event_date", Value:1}}))
	if err != nil { return nil, err }
	defer cur.Close(ctx)
	var list []models.Event
	if err = cur.All(ctx, &list); err != nil { return nil, err }
	return list, nil
}

func (r *mongoEventRepo) Advance(ctx context.Context, userID, eventID primitive.ObjectID, reason string) (*models.Event, error) {
	// 读取当前
	var ev models.Event
	if err := r.coll().FindOne(ctx, bson.M{"_id": eventID, "user_id": userID}).Decode(&ev); err != nil {
		if err == mongo.ErrNoDocuments { return nil, errors.New("event not found") }
		return nil, err
	}
	now := time.Now(); oldDate := ev.EventDate
	set := bson.M{"updated_at": now}
	if ev.RecurrenceType == "none" { // 完成
		set["is_active"] = false
		set["last_triggered_at"] = now
	} else {
		var next time.Time
		switch ev.RecurrenceType {
		case "daily": next = ev.EventDate.AddDate(0,0,1)
		case "weekly": next = ev.EventDate.AddDate(0,0,7)
		case "monthly": next = ev.EventDate.AddDate(0,1,0)
		case "yearly": next = ev.EventDate.AddDate(1,0,0)
		default:
			set["is_active"] = false
		}
		for !next.IsZero() && next.Before(now) { // 防御推进
			switch ev.RecurrenceType {
			case "daily": next = next.AddDate(0,0,1)
			case "weekly": next = next.AddDate(0,0,7)
			case "monthly": next = next.AddDate(0,1,0)
			case "yearly": next = next.AddDate(1,0,0)
			default: break
			}
		}
		if !next.IsZero() {
			set["event_date"] = next
			set["last_triggered_at"] = now
			ev.EventDate = next
		}
	}
	_, err := r.coll().UpdateOne(ctx, bson.M{"_id": ev.ID, "user_id": userID}, bson.M{"$set": set})
	if err != nil { return nil, fmt.Errorf("advance update err: %w", err) }
	// 异步重算提醒
	go func(e models.Event){
		defer func(){ recover() }()
		cur, err2 := r.reminders().Find(context.Background(), bson.M{"event_id": e.ID, "is_active": true})
		if err2 != nil { return }
		defer cur.Close(context.Background())
		for cur.Next(context.Background()) {
			var rm models.Reminder
			if cur.Decode(&rm) == nil {
				if len(rm.AbsoluteTimes) == 0 { // 相对事件时间
					upd := bson.M{"updated_at": time.Now()}
					if nx := rm.CalculateNextSendTime(e); nx != nil { upd["next_send"] = *nx } else { upd["next_send"] = nil }
					_, _ = r.reminders().UpdateByID(context.Background(), rm.ID, bson.M{"$set": upd})
				}
			}
		}
	}(ev)
	// 写时间线
	go func(old time.Time, newEv models.Event){
		defer func(){ recover() }()
		meta := bson.M{"action": "advance", "old": old.Format(time.RFC3339), "recurrence": newEv.RecurrenceType, "active": fmt.Sprintf("%v", newEv.IsActive)}
		if reason != "" { meta["reason"] = reason }
		if newEv.EventDate.After(old) { meta["new"] = newEv.EventDate.Format(time.RFC3339) }
		content := "advanced"
		if reason != "" { content += ": " + reason }
		_, _ = r.comments().InsertOne(context.Background(), bson.M{"_id": primitive.NewObjectID(), "event_id": newEv.ID, "user_id": newEv.UserID, "type": "system", "content": content, "meta": meta, "created_at": time.Now(), "updated_at": time.Now()})
	}(oldDate, ev)
	return &ev, nil
}

func (r *mongoEventRepo) ListStartingWindow(ctx context.Context, from, to time.Time) ([]models.Event, error) {
	filter := bson.M{"is_active": true, "event_date": bson.M{"$gte": from, "$lte": to}}
	cur, err := r.coll().Find(ctx, filter)
	if err != nil { return nil, err }
	defer cur.Close(ctx)
	var list []models.Event
	if err = cur.All(ctx, &list); err != nil { return nil, err }
	return list, nil
}

func (r *mongoEventRepo) MarkTriggered(ctx context.Context, id primitive.ObjectID, ts time.Time) error {
	_, err := r.coll().UpdateByID(ctx, id, bson.M{"$set": bson.M{"last_triggered_at": ts}})
	return err
}
