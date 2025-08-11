package services

import (
	"context"
	"errors"
	"log"
	"sort"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/axfinn/todoIngPlus/backend-go/internal/models"
)

// UnifiedService 聚合服务
type UnifiedService struct {
	db *mongo.Database
}

func NewUnifiedService(db *mongo.Database) *UnifiedService { return &UnifiedService{db: db} }

// GetUpcoming 聚合获取未来 hours 内的任务/事件/提醒
// GetUpcoming 聚合获取未来 hours 内的任务/事件/提醒，可按 sources 过滤并限制数量
// sources 为空或包含未知值时默认全量(task,event,reminder)
// simpleTask 为普通任务轻量数据，用于聚合排序（放到文件级，便于测试）
type simpleTask struct {
	ID            string
	Title         string
	Deadline      *time.Time
	ScheduledDate *time.Time
	Unscheduled   bool // 标记是否是临时无日期任务（使用 now 占位）
}

// buildUpcomingItems 抽取出的纯构建 & 过滤 & 排序逻辑，便于单元测试
func buildUpcomingItems(now, end time.Time, sources []string, limit int, events []models.Event, reminders []models.UpcomingReminder, priorityTasks []models.PriorityTask, normalTasks []simpleTask) []models.UnifiedItem {
	items := make([]models.UnifiedItem, 0, len(events)+len(reminders)+len(priorityTasks)+len(normalTasks))
	sourceFilter := map[string]bool{}
	if len(sources) > 0 {
		for _, s2 := range sources {
			sourceFilter[s2] = true
		}
	}
	useFilter := len(sourceFilter) > 0
	// 事件
	for _, ev := range events {
		if ev.EventDate.After(end) {
			continue
		}
		secs := int(ev.EventDate.Sub(now).Seconds())
		daysLeft := int(ev.EventDate.Sub(now).Hours() / 24)
		if useFilter && !sourceFilter["event"] {
			continue
		}
		items = append(items, models.UnifiedItem{ID: ev.ID.Hex(), Source: "event", Title: ev.Title, ScheduledAt: ev.EventDate, CountdownSeconds: secs, DaysLeft: daysLeft, Importance: ev.ImportanceLevel, DetailURL: "/events", SourceID: ev.ID.Hex()})
	}
	// 提醒
	for _, r := range reminders {
		secs := int(r.ReminderAt.Sub(now).Seconds())
		daysLeft := int(r.ReminderAt.Sub(now).Hours() / 24)
		if useFilter && !sourceFilter["reminder"] {
			continue
		}
		items = append(items, models.UnifiedItem{ID: r.ID.Hex(), Source: "reminder", Title: r.Message, ScheduledAt: r.ReminderAt, CountdownSeconds: secs, DaysLeft: daysLeft, Importance: r.Importance, DetailURL: "/reminders", SourceID: r.ID.Hex()})
	}
	// 优先任务 & 去重集
	prioritySet := make(map[string]struct{})
	for _, t := range priorityTasks {
		// 计划时间优先采用 scheduledDate，其次 deadline（scheduledDate 表示用户安排的执行时间，deadline 是最后期限）
		var sched *time.Time
		if t.ScheduledDate != nil {
			sched = t.ScheduledDate
		} else if t.Deadline != nil {
			sched = t.Deadline
		}
		if sched == nil || sched.After(end) { // 允许已过当前时间(显示为逾期)
			continue
		}
		if useFilter && !sourceFilter["task"] {
			continue
		}
		secs := int(sched.Sub(now).Seconds())
		daysLeft := int(sched.Sub(now).Hours() / 24)
		items = append(items, models.UnifiedItem{ID: t.ID, Source: "task", Title: t.Title, ScheduledAt: *sched, CountdownSeconds: secs, DaysLeft: daysLeft, PriorityScore: t.PriorityScore, DetailURL: "/dashboard", SourceID: t.ID})
		prioritySet[t.ID] = struct{}{}
	}
	// 普通任务
	for _, t := range normalTasks {
		if _, exists := prioritySet[t.ID]; exists {
			continue
		}
		// 计划时间优先采用 scheduledDate，其次 deadline
		var sched *time.Time
		if t.ScheduledDate != nil {
			sched = t.ScheduledDate
		} else if t.Deadline != nil {
			sched = t.Deadline
		}
		if sched == nil { // FIX: 没有真实日期的任务不进入统一看板，避免显示伪造时间
			continue
		}
		if sched.After(end) { // 超出窗口
			continue
		}
		if useFilter && !sourceFilter["task"] {
			continue
		}
		secs := int(sched.Sub(now).Seconds())
		daysLeft := int(sched.Sub(now).Hours() / 24)
		items = append(items, models.UnifiedItem{ID: t.ID, Source: "task", Title: t.Title, ScheduledAt: *sched, CountdownSeconds: secs, DaysLeft: daysLeft, DetailURL: "/dashboard", SourceID: t.ID, IsUnscheduled: t.Unscheduled})
	}
	// 排序
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].ScheduledAt.Equal(items[j].ScheduledAt) {
			if items[i].Importance == items[j].Importance {
				return items[i].PriorityScore > items[j].PriorityScore
			}
			return items[i].Importance > items[j].Importance
		}
		return items[i].ScheduledAt.Before(items[j].ScheduledAt)
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

func buildTaskWindowFilter(userID primitive.ObjectID, graceStart, end time.Time) bson.M {
	return bson.M{"$and": []bson.M{
		{"$or": []bson.M{{"createdBy": userID.Hex()}, {"createdBy": userID}, {"user_id": userID}}},
		{"status": bson.M{"$nin": []string{"Done", "done", "DONE", "已完成"}}},
		{"$or": []bson.M{{"deadline": bson.M{"$gte": graceStart, "$lte": end}}, {"scheduledDate": bson.M{"$gte": graceStart, "$lte": end}}, {"dueDate": bson.M{"$gte": graceStart, "$lte": end}}}},
	}}
}

func buildUnscheduledFilter(userID primitive.ObjectID, since time.Time) bson.M {
	return bson.M{
		"$or":    []bson.M{{"createdBy": userID.Hex()}, {"createdBy": userID}, {"user_id": userID}},
		"status": bson.M{"$nin": []string{"Done", "done", "DONE", "已完成"}},
		"$and": []bson.M{
			{"$or": []bson.M{{"deadline": bson.M{"$exists": false}}, {"deadline": nil}}},
			{"$or": []bson.M{{"scheduledDate": bson.M{"$exists": false}}, {"scheduledDate": nil}}},
		},
		"createdAt": bson.M{"$gte": since},
	}
}

func (s *UnifiedService) GetUpcoming(ctx context.Context, userID primitive.ObjectID, hours int, sources []string, limit int) ([]models.UnifiedItem, *models.UnifiedUpcomingDebug, error) {
	if s == nil || s.db == nil {
		return nil, nil, errors.New("unified service db not initialized")
	}
	if hours <= 0 {
		hours = 24 * 7
	}
	now := time.Now()
	end := now.Add(time.Duration(hours) * time.Hour)

	eventSvc := NewEventService(s.db)
	reminderSvc := NewReminderService(s.db)
	taskSortSvc := NewTaskSortService(s.db)

	var eventsErr, remindersErr, priorityErr, normalTasksErr error
	var events []models.Event
	var reminders []models.UpcomingReminder
	var priorityTasks []models.PriorityTask
	var normalTasks []simpleTask

	var wg sync.WaitGroup
	wg.Add(4)
	go func() { // upcoming events
		defer wg.Done()
		days := hours / 24
		if days < 1 {
			days = 1
		}
		events, eventsErr = eventSvc.GetUpcomingEvents(ctx, userID, days)
	}()
	go func() { // upcoming reminders
		defer wg.Done()
		reminders, remindersErr = reminderSvc.GetUpcomingReminders(ctx, userID, hours)
	}()
	go func() { // priority tasks (already filters by createdBy)
		defer wg.Done()
		priorityTasks, priorityErr = taskSortSvc.GetPriorityTasks(ctx, userID)
	}()
	go func() { // normal tasks with deadline/scheduled/dueDate window + recent unscheduled tasks
		defer wg.Done()
		tasksColl := s.db.Collection("tasks")
		// 新策略: 直接获取所有未完成任务(不加日期范围)以避免因日期字段为字符串/格式异常导致 Mongo 端过滤失败
		// 之后在内存中解析 deadline / scheduledDate / dueDate，统一计算展示时间
		baseFilter := bson.M{"$and": []bson.M{{"$or": []bson.M{{"createdBy": userID.Hex()}, {"createdBy": userID}, {"user_id": userID}}}, {"status": bson.M{"$nin": []string{"Done", "done", "DONE", "已完成"}}}}}
		// 设一个最大条数上限，防止用户有海量历史任务导致一次性拉取过大
		maxTasks := int64(1000)
		opts := options.Find().SetProjection(bson.M{"title": 1, "deadline": 1, "scheduledDate": 1, "dueDate": 1, "createdAt": 1}).SetSort(bson.D{{Key: "deadline", Value: 1}, {Key: "scheduledDate", Value: 1}, {Key: "createdAt", Value: -1}}).SetLimit(maxTasks)
		cur, err := tasksColl.Find(ctx, baseFilter, opts)
		if err != nil {
			normalTasksErr = err
			return
		}
		defer cur.Close(ctx)
		deadlineCount, scheduledCount, dueDateOnlyCount, unscheduledCount := 0, 0, 0, 0
		for cur.Next(ctx) {
			var raw bson.M
			if err := cur.Decode(&raw); err != nil {
				continue
			}
			id := ""
			if oid, ok := raw["_id"].(primitive.ObjectID); ok {
				id = oid.Hex()
			} else if s, ok := raw["_id"].(string); ok {
				id = s
			}
			if id == "" {
				continue
			}
			title, _ := raw["title"].(string)
			parseAnyTime := func(v interface{}) *time.Time {
				switch tv := v.(type) {
				case primitive.DateTime:
					tt := tv.Time()
					return &tt
				case time.Time:
					return &tv
				case *time.Time:
					return tv
				case string:
					if tv == "" {
						return nil
					}
					if t1, err := time.Parse(time.RFC3339, tv); err == nil {
						return &t1
					}
					if t2, err := time.Parse("2006-01-02", tv); err == nil {
						return &t2
					}
				}
				return nil
			}
			deadline := parseAnyTime(raw["deadline"])
			scheduled := parseAnyTime(raw["scheduledDate"])
			var dueDate *time.Time
			if deadline == nil { // 才考虑 dueDate 填补
				dueDate = parseAnyTime(raw["dueDate"])
				if dueDate != nil {
					deadline = dueDate
					dueDateOnlyCount++
				}
			}
			if deadline != nil {
				deadlineCount++
			} else if scheduled != nil {
				scheduledCount++
			}
			// FIX: 不再用 createdAt/now 伪造无日期任务的时间，防止统一看板显示错误的“任务时间”
			// 统一优先顺序: scheduledDate > deadline/dueDate；若都没有则标记为 unscheduled 并跳过聚合排序
			var sched *time.Time
			if scheduled != nil {
				sched = scheduled
			} else if deadline != nil {
				sched = deadline
			}
			unscheduled := false
			if sched == nil { // 完全无日期 => 仅标记，不赋值伪时间
				unscheduled = true
				unscheduledCount++
			}
			normalTasks = append(normalTasks, simpleTask{ID: id, Title: title, Deadline: deadline, ScheduledDate: sched, Unscheduled: unscheduled})
		}
		log.Printf("unified: tasks(all-active) deadline=%d scheduled=%d dueDateOnly=%d unscheduled=%d total=%d", deadlineCount, scheduledCount, dueDateOnlyCount, unscheduledCount, len(normalTasks))
	}()
	wg.Wait()
	if normalTasksErr != nil {
		log.Printf("unified: normal tasks query error: %v", normalTasksErr)
	}
	if eventsErr != nil {
		log.Printf("unified: events query error: %v", eventsErr)
	}
	if remindersErr != nil {
		log.Printf("unified: reminders query error: %v", remindersErr)
	}
	if priorityErr != nil {
		log.Printf("unified: priority tasks query error: %v", priorityErr)
	}
	log.Printf("unified: fetched events=%d reminders=%d priorityTasks=%d normalTasks=%d window=%dh", len(events), len(reminders), len(priorityTasks), len(normalTasks), hours)

	// Assemble debug if context requires it
	var dbg *models.UnifiedUpcomingDebug
	if v := ctx.Value("unifiedDebug"); v != nil {
		// Rebuild filters (cheap) for serialization
		graceDays := 2
		graceStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -graceDays)
		wf := buildTaskWindowFilter(userID, graceStart, end)
		uf := buildUnscheduledFilter(userID, now.Add(-24*time.Hour))
		dbg = &models.UnifiedUpcomingDebug{GraceDays: graceDays, Hours: hours, Now: now.Format(time.RFC3339), End: end.Format(time.RFC3339)}
		// window counts recovered from log not stored; do a lightweight recount over normalTasks slice
		for _, t := range normalTasks {
			if t.Deadline != nil {
				dbg.WindowCounts.Deadline++
			} else if t.ScheduledDate != nil && !t.Unscheduled {
				dbg.WindowCounts.Scheduled++
			}
		}
		// dueDateOnly 无法区分已合并的 dueDate -> Deadline，忽略
		dbg.WindowCounts.Total = len(normalTasks)
		// 记录未进入列表（被跳过的无日期任务）数量: 统计标记为 Unscheduled 的数量
		dbg.AddedUnscheduled = 0
		for _, t := range normalTasks {
			if t.Unscheduled {
				dbg.AddedUnscheduled++
			}
		}
		// convert bson.M to map[string]interface{}
		dbg.WindowFilter = map[string]interface{}(wf)
		dbg.UnscheduledFilter = map[string]interface{}(uf)
	}

	return buildUpcomingItems(now, end, sources, limit, events, reminders, priorityTasks, normalTasks), dbg, nil
}

// GetCalendar 聚合指定年月的事件/提醒(按 next_send 落在该月?) + 有日期的任务
func (s *UnifiedService) GetCalendar(ctx context.Context, userID primitive.ObjectID, year, month int) (models.UnifiedCalendarResponse, error) {
	if s == nil || s.db == nil {
		return models.UnifiedCalendarResponse{}, errors.New("unified service db not initialized")
	}
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	daysMap := make(map[string][]models.UnifiedCalendarItem)

	eventSvc := NewEventService(s.db)
	events, _ := eventSvc.GetCalendarEvents(ctx, userID, year, month)
	for day, evs := range events {
		for _, ev := range evs {
			daysMap[day] = append(daysMap[day], models.UnifiedCalendarItem{ID: ev.ID.Hex(), Source: "event", Title: ev.Title, ScheduledAt: ev.EventDate, Importance: ev.ImportanceLevel, DetailURL: "/events"})
		}
	}

	// 提醒: 取 next_send 落在该月范围内
	reminderSvc := NewReminderService(s.db)
	reminders, _ := reminderSvc.GetUpcomingReminders(ctx, userID, 24*31) // 先取一月内的所有即将提醒
	for _, r := range reminders {
		if r.ReminderAt.IsZero() {
			continue
		}
		if r.ReminderAt.Before(start) || r.ReminderAt.Equal(end) || r.ReminderAt.After(end) {
			continue
		}
		dayKey := r.ReminderAt.Format("2006-01-02")
		daysMap[dayKey] = append(daysMap[dayKey], models.UnifiedCalendarItem{ID: r.ID.Hex(), Source: "reminder", Title: r.Message, ScheduledAt: r.ReminderAt, Importance: r.Importance, DetailURL: "/reminders"})
	}

	// 任务 (兼容 createdBy 字符串 / 未来可能的 user_id ObjectID)
	tasksColl := s.db.Collection("tasks")
	tFilter := bson.M{"$and": []bson.M{
		{"$or": []bson.M{{"createdBy": userID.Hex()}, {"user_id": userID}}},
		{"$or": []bson.M{{"deadline": bson.M{"$gte": start, "$lt": end}}, {"scheduledDate": bson.M{"$gte": start, "$lt": end}}}},
	}}
	cur, err := tasksColl.Find(ctx, tFilter, options.Find().SetProjection(bson.M{"title": 1, "deadline": 1, "scheduledDate": 1}))
	if err == nil {
		defer cur.Close(ctx)
		for cur.Next(ctx) {
			var doc struct {
				ID            string     `bson:"_id"`
				Title         string     `bson:"title"`
				Deadline      *time.Time `bson:"deadline"`
				ScheduledDate *time.Time `bson:"scheduledDate"`
			}
			if cur.Decode(&doc) == nil {
				var d *time.Time
				if doc.Deadline != nil {
					d = doc.Deadline
				} else {
					d = doc.ScheduledDate
				}
				if d != nil {
					key := d.Format("2006-01-02")
					daysMap[key] = append(daysMap[key], models.UnifiedCalendarItem{ID: doc.ID, Source: "task", Title: doc.Title, ScheduledAt: *d, DetailURL: "/dashboard"})
				}
			}
		}
	}

	total := 0
	for k := range daysMap { // 排序每日日项
		items := daysMap[k]
		sort.Slice(items, func(i, j int) bool { return items[i].ScheduledAt.Before(items[j].ScheduledAt) })
		daysMap[k] = items
		total += len(items)
	}
	return models.UnifiedCalendarResponse{Year: year, Month: month, Days: daysMap, Total: total}, nil
}
