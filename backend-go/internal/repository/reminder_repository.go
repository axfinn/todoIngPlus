package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/axfinn/todoIngPlus/backend-go/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// ReminderRepository 提醒仓储，聚合逻辑集中此处
// 统一：所有方法不返回 mongo.ErrNoDocuments，转换为业务 not found / 空结果
// GetWithEvent / ListPagedWithEvent 使用 $lookup，避免上层直接聚合

type ReminderRepository interface {
	Insert(ctx context.Context, r *models.Reminder) error
	GetWithEvent(ctx context.Context, userID, reminderID primitive.ObjectID) (*models.ReminderWithEvent, error)
	UpdateFields(ctx context.Context, userID, reminderID primitive.ObjectID, set map[string]interface{}, recomputeNext bool) (*models.Reminder, error)
	Delete(ctx context.Context, userID, reminderID primitive.ObjectID) error
	ListPagedWithEvent(ctx context.Context, userID primitive.ObjectID, page, pageSize int, activeOnly bool) (*models.ReminderListResponse, error)
	ListSimple(ctx context.Context, userID primitive.ObjectID, activeOnly bool, limit int) ([]SimpleReminderDTO, error)
	Upcoming(ctx context.Context, userID primitive.ObjectID, hours int) ([]models.UpcomingReminder, error)
	Pending(ctx context.Context) ([]models.ReminderWithEvent, error)
	ToggleActive(ctx context.Context, userID, reminderID primitive.ObjectID) (bool, error)
	Snooze(ctx context.Context, userID, reminderID primitive.ObjectID, minutes int) error
	MarkSent(ctx context.Context, reminderID primitive.ObjectID) error
	CreateImmediateTest(ctx context.Context, userID, eventID primitive.ObjectID, message string, delaySeconds int) (*models.Reminder, *models.Event, error)
	// Preview 预览提醒下一次发送时间（供服务层直接使用）
	Preview(ctx context.Context, userID, eventID primitive.ObjectID, advanceDays int, times []string) (*PreviewReminderOutput, error)
}

type mongoReminderRepo struct{ db *mongo.Database }

func NewReminderRepository(db *mongo.Database) ReminderRepository { return &mongoReminderRepo{db: db} }

func (r *mongoReminderRepo) coll() *mongo.Collection   { return r.db.Collection("reminders") }
func (r *mongoReminderRepo) events() *mongo.Collection { return r.db.Collection("events") }

// SimpleReminderDTO 精简数据
type SimpleReminderDTO struct {
	ID            primitive.ObjectID `bson:"_id" json:"id"`
	EventID       primitive.ObjectID `bson:"event_id" json:"event_id"`
	EventTitle    string             `bson:"event_title" json:"event_title"`
	EventDate     time.Time          `bson:"event_date" json:"event_date"`
	AdvanceDays   int                `bson:"advance_days" json:"advance_days"`
	ReminderTimes []string           `bson:"reminder_times" json:"reminder_times"`
	ReminderType  string             `bson:"reminder_type" json:"reminder_type"`
	CustomMessage string             `bson:"custom_message" json:"custom_message,omitempty"`
	IsActive      bool               `bson:"is_active" json:"is_active"`
	NextSend      *time.Time         `bson:"next_send" json:"next_send,omitempty"`
	LastSent      *time.Time         `bson:"last_sent" json:"last_sent,omitempty"`
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at" json:"updated_at"`
}

func (r *mongoReminderRepo) Insert(ctx context.Context, m *models.Reminder) error {
	if m == nil {
		return errors.New("nil reminder")
	}
	if m.ID.IsZero() {
		m.ID = primitive.NewObjectID()
	}
	now := time.Now()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = now
	// 计算 next_send
	var ev models.Event
	if err := r.events().FindOne(ctx, bson.M{"_id": m.EventID, "user_id": m.UserID}).Decode(&ev); err == nil {
		m.NextSend = m.CalculateNextSendTime(ev)
	}
	_, err := r.coll().InsertOne(ctx, m)
	return err
}

func (r *mongoReminderRepo) GetWithEvent(ctx context.Context, userID, reminderID primitive.ObjectID) (*models.ReminderWithEvent, error) {
	pipeline := []bson.M{
		{"$match": bson.M{"_id": reminderID, "user_id": userID}},
		{"$lookup": bson.M{"from": "events", "localField": "event_id", "foreignField": "_id", "as": "event"}},
		{"$unwind": "$event"},
		{"$project": bson.M{"_id": 1, "event_id": 1, "user_id": 1, "advance_days": 1, "reminder_times": 1, "reminder_type": 1, "custom_message": 1, "is_active": 1, "last_sent": 1, "next_send": 1, "created_at": 1, "updated_at": 1, "event": "$event"}},
	}
	cur, err := r.coll().Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var res []models.ReminderWithEvent
	if err = cur.All(ctx, &res); err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, errors.New("reminder not found")
	}
	return &res[0], nil
}

func (r *mongoReminderRepo) UpdateFields(ctx context.Context, userID, reminderID primitive.ObjectID, set map[string]interface{}, recomputeNext bool) (*models.Reminder, error) {
	if set == nil {
		set = map[string]interface{}{}
	}
	set["updated_at"] = time.Now()
	bset := bson.M{}
	for k, v := range set {
		bset[k] = v
	}
	res, err := r.coll().UpdateOne(ctx, bson.M{"_id": reminderID, "user_id": userID}, bson.M{"$set": bset})
	if err != nil {
		return nil, err
	}
	if res.MatchedCount == 0 {
		return nil, errors.New("reminder not found")
	}
	var rm models.Reminder
	if err = r.coll().FindOne(ctx, bson.M{"_id": reminderID}).Decode(&rm); err != nil {
		return nil, err
	}
	if recomputeNext {
		var ev models.Event
		if err := r.events().FindOne(ctx, bson.M{"_id": rm.EventID}).Decode(&ev); err == nil {
			if nx := rm.CalculateNextSendTime(ev); true {
				_, _ = r.coll().UpdateOne(ctx, bson.M{"_id": rm.ID}, bson.M{"$set": bson.M{"next_send": nx, "updated_at": time.Now()}})
				rm.NextSend = nx
			}
		}
	}
	return &rm, nil
}

func (r *mongoReminderRepo) Delete(ctx context.Context, userID, reminderID primitive.ObjectID) error {
	res, err := r.coll().DeleteOne(ctx, bson.M{"_id": reminderID, "user_id": userID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return errors.New("reminder not found")
	}
	return nil
}

func (r *mongoReminderRepo) ListPagedWithEvent(ctx context.Context, userID primitive.ObjectID, page, pageSize int, activeOnly bool) (*models.ReminderListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	match := bson.M{"user_id": userID}
	if activeOnly {
		match["is_active"] = true
	}
	pipeline := []bson.M{
		{"$match": match},
		{"$lookup": bson.M{"from": "events", "localField": "event_id", "foreignField": "_id", "as": "event"}},
		{"$unwind": "$event"},
		{"$project": bson.M{"_id": 1, "event_id": 1, "user_id": 1, "advance_days": 1, "reminder_times": 1, "reminder_type": 1, "custom_message": 1, "is_active": 1, "last_sent": 1, "next_send": 1, "created_at": 1, "updated_at": 1, "event": "$event"}},
	}
	// count
	countPipe := append(append([]bson.M{}, pipeline[:1]...), bson.M{"$count": "total"})
	curCount, err := r.coll().Aggregate(ctx, countPipe)
	if err != nil {
		return nil, err
	}
	var cnt []bson.M
	_ = curCount.All(ctx, &cnt)
	curCount.Close(ctx)
	var total int64
	if len(cnt) > 0 {
		switch v := cnt[0]["total"].(type) {
		case int32:
			total = int64(v)
		case int64:
			total = v
		}
	}
	skip := (page - 1) * pageSize
	pipeline = append(pipeline, bson.M{"$sort": bson.M{"created_at": -1}}, bson.M{"$skip": skip}, bson.M{"$limit": pageSize})
	cur, err := r.coll().Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var list []models.ReminderWithEvent
	if err = cur.All(ctx, &list); err != nil {
		return nil, err
	}
	pages := int(math.Ceil(float64(total) / float64(pageSize)))
	return &models.ReminderListResponse{Reminders: list, Total: total, Page: page, PageSize: pageSize, TotalPages: pages}, nil
}

func (r *mongoReminderRepo) ListSimple(ctx context.Context, userID primitive.ObjectID, activeOnly bool, limit int) ([]SimpleReminderDTO, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	match := bson.M{"user_id": userID}
	if activeOnly {
		match["is_active"] = true
	}
	pipeline := []bson.M{
		{"$match": match},
		{"$lookup": bson.M{"from": "events", "localField": "event_id", "foreignField": "_id", "as": "event"}},
		{"$unwind": "$event"},
		{"$project": bson.M{"_id": 1, "event_id": 1, "event_title": "$event.title", "event_date": "$event.event_date", "advance_days": 1, "reminder_times": 1, "reminder_type": 1, "custom_message": 1, "is_active": 1, "next_send": 1, "last_sent": 1, "created_at": 1, "updated_at": 1}},
		{"$sort": bson.M{"created_at": -1}},
		{"$limit": limit},
	}
	cur, err := r.coll().Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []SimpleReminderDTO
	if err = cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *mongoReminderRepo) Upcoming(ctx context.Context, userID primitive.ObjectID, hours int) ([]models.UpcomingReminder, error) {
	if hours <= 0 {
		hours = 24
	}
	now := time.Now()
	end := now.Add(time.Duration(hours) * time.Hour)
	pipeline := []bson.M{{"$match": bson.M{"user_id": userID, "is_active": true, "next_send": bson.M{"$gte": now, "$lte": end}}}, {"$lookup": bson.M{"from": "events", "localField": "event_id", "foreignField": "_id", "as": "event"}}, {"$unwind": "$event"}, {"$project": bson.M{"_id": 1, "custom_message": 1, "event_title": "$event.title", "event_date": "$event.event_date", "importance_level": "$event.importance_level", "next_send": 1}}, {"$sort": bson.M{"next_send": 1}}}
	cur, err := r.coll().Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	parse := func(v interface{}) *time.Time {
		switch tv := v.(type) {
		case primitive.DateTime:
			t := tv.Time()
			return &t
		case time.Time:
			return &tv
		case *time.Time:
			return tv
		case string:
			if tv == "" {
				return nil
			}
			if t1, e := time.Parse(time.RFC3339, tv); e == nil {
				return &t1
			}
			if t2, e := time.Parse("2006-01-02 15:04:05", tv); e == nil {
				return &t2
			}
			if t3, e := time.Parse("2006-01-02", tv); e == nil {
				return &t3
			}
		}
		return nil
	}
	var out []models.UpcomingReminder
	for cur.Next(ctx) {
		var raw bson.M
		if cur.Decode(&raw) != nil {
			continue
		}
		id, _ := raw["_id"].(primitive.ObjectID)
		title, _ := raw["event_title"].(string)
		evDate := parse(raw["event_date"])
		nx := parse(raw["next_send"])
		if nx == nil || nx.Year() < 2000 {
			continue
		}
		msg := ""
		if cm, ok := raw["custom_message"].(string); ok && cm != "" {
			msg = cm
		} else if title != "" {
			msg = "提醒: " + title
		} else {
			msg = "提醒"
		}
		imp := 0
		switch v := raw["importance_level"].(type) {
		case int32:
			imp = int(v)
		case int64:
			imp = int(v)
		case int:
			imp = v
		}
		base := nx
		if evDate != nil && !evDate.IsZero() {
			base = evDate
		}
		daysLeft := int(base.Sub(now).Hours() / 24)
		ed := time.Time{}
		if evDate != nil && evDate.Year() >= 2000 {
			ed = *evDate
		}
		out = append(out, models.UpcomingReminder{ID: id, EventTitle: title, EventDate: ed, Message: msg, DaysLeft: daysLeft, Importance: imp, ReminderAt: *nx})
	}
	return out, cur.Err()
}

func (r *mongoReminderRepo) Pending(ctx context.Context) ([]models.ReminderWithEvent, error) {
	now := time.Now()
	pipeline := []bson.M{{"$match": bson.M{"is_active": true, "next_send": bson.M{"$lte": now}}}, {"$lookup": bson.M{"from": "events", "localField": "event_id", "foreignField": "_id", "as": "event"}}, {"$unwind": "$event"}, {"$sort": bson.M{"next_send": 1}}}
	cur, err := r.coll().Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var list []models.ReminderWithEvent
	if err = cur.All(ctx, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *mongoReminderRepo) ToggleActive(ctx context.Context, userID, reminderID primitive.ObjectID) (bool, error) {
	var rm models.Reminder
	if err := r.coll().FindOne(ctx, bson.M{"_id": reminderID, "user_id": userID}).Decode(&rm); err != nil {
		if err == mongo.ErrNoDocuments {
			return false, errors.New("reminder not found")
		}
		return false, err
	}
	newVal := !rm.IsActive
	_, err := r.coll().UpdateOne(ctx, bson.M{"_id": rm.ID}, bson.M{"$set": bson.M{"is_active": newVal, "updated_at": time.Now()}})
	return newVal, err
}

func (r *mongoReminderRepo) Snooze(ctx context.Context, userID, reminderID primitive.ObjectID, minutes int) error {
	if minutes <= 0 {
		minutes = 60
	}
	res, err := r.coll().UpdateOne(ctx, bson.M{"_id": reminderID, "user_id": userID}, bson.M{"$set": bson.M{"next_send": time.Now().Add(time.Duration(minutes) * time.Minute), "updated_at": time.Now()}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return errors.New("reminder not found")
	}
	return nil
}

func (r *mongoReminderRepo) MarkSent(ctx context.Context, reminderID primitive.ObjectID) error {
	var rm models.Reminder
	if err := r.coll().FindOne(ctx, bson.M{"_id": reminderID}).Decode(&rm); err != nil {
		return err
	}
	var ev models.Event
	if err := r.events().FindOne(ctx, bson.M{"_id": rm.EventID}).Decode(&ev); err != nil {
		return err
	}
	now := time.Now()
	next := rm.CalculateNextSendTime(ev)
	_, err := r.coll().UpdateOne(ctx, bson.M{"_id": reminderID}, bson.M{"$set": bson.M{"last_sent": now, "next_send": next, "updated_at": now}})
	return err
}

func (r *mongoReminderRepo) CreateImmediateTest(ctx context.Context, userID, eventID primitive.ObjectID, message string, delaySeconds int) (*models.Reminder, *models.Event, error) {
	if delaySeconds < 0 {
		delaySeconds = 0
	}
	if delaySeconds > 3600 {
		delaySeconds = 3600
	}
	var ev models.Event
	if err := r.events().FindOne(ctx, bson.M{"_id": eventID, "user_id": userID}).Decode(&ev); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil, errors.New("event not found")
		}
		return nil, nil, err
	}
	now := time.Now()
	rm := &models.Reminder{ID: primitive.NewObjectID(), EventID: eventID, UserID: userID, AdvanceDays: 0, ReminderTimes: []string{now.Add(time.Duration(delaySeconds) * time.Second).Format("15:04")}, ReminderType: "app", CustomMessage: message, IsActive: true, CreatedAt: now, UpdatedAt: now}
	ns := now.Add(time.Duration(delaySeconds) * time.Second)
	rm.NextSend = &ns
	if _, err := r.coll().InsertOne(ctx, rm); err != nil {
		return nil, nil, err
	}
	return rm, &ev, nil
}

// Preview 逻辑可放在 service 组合中：这里只提供 event + 计算函数
// 辅助：构造预览文本
func BuildPreview(event *models.Event, eventFound bool, advanceDays int, times []string, next *time.Time) (string, []string) {
	var errs []string
	if !eventFound {
		errs = append(errs, "event not found")
	}
	var b strings.Builder
	if event != nil {
		b.WriteString(event.Title)
	} else {
		b.WriteString("(event)")
	}
	if event != nil {
		b.WriteString(" @ ")
		b.WriteString(event.EventDate.Format("2006-01-02 15:04"))
	}
	b.WriteString(fmt.Sprintf(" | advance %d day(s) | times %v", advanceDays, times))
	if next != nil {
		b.WriteString(" => next ")
		b.WriteString(next.Format("2006-01-02 15:04"))
	}
	return b.String(), errs
}

// 预览计算（供 service 使用）
func (r *mongoReminderRepo) computePreview(ctx context.Context, userID, eventID primitive.ObjectID, advanceDays int, times []string) (*models.Event, *time.Time, error) {
	var ev models.Event
	err := r.events().FindOne(ctx, bson.M{"_id": eventID, "user_id": userID}).Decode(&ev)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	temp := models.Reminder{EventID: eventID, AdvanceDays: advanceDays, ReminderTimes: times, ReminderType: "app", IsActive: true}
	return &ev, temp.CalculateNextSendTime(ev), nil
}

// PreviewReminderOutput 提供给上层的复合结果
type PreviewReminderOutput struct {
	Event          *models.Event
	Next           *time.Time
	Text           string
	ValidationErrs []string
}

func (r *mongoReminderRepo) Preview(ctx context.Context, userID, eventID primitive.ObjectID, advanceDays int, times []string) (*PreviewReminderOutput, error) {
	ev, next, err := r.computePreview(ctx, userID, eventID, advanceDays, times)
	if err != nil {
		return nil, err
	}
	text, errs := BuildPreview(ev, ev != nil, advanceDays, times, next)
	return &PreviewReminderOutput{Event: ev, Next: next, Text: text, ValidationErrs: errs}, nil
}
