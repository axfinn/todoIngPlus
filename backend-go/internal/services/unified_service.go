package services

import (
    "context"
    "errors"
    "sort"
    "sync"
    "time"

    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"

    "github.com/axfinn/todoIng/backend-go/internal/models"
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
type simpleTask struct { ID string; Title string; Deadline *time.Time; ScheduledDate *time.Time }

// buildUpcomingItems 抽取出的纯构建 & 过滤 & 排序逻辑，便于单元测试
func buildUpcomingItems(now, end time.Time, sources []string, limit int, events []models.Event, reminders []models.UpcomingReminder, priorityTasks []models.PriorityTask, normalTasks []simpleTask) []models.UnifiedItem {
    items := make([]models.UnifiedItem, 0, len(events)+len(reminders)+len(priorityTasks)+len(normalTasks))
    sourceFilter := map[string]bool{}
    if len(sources) > 0 { for _, s2 := range sources { sourceFilter[s2] = true } }
    useFilter := len(sourceFilter) > 0
    // 事件
    for _, ev := range events {
        if ev.EventDate.After(end) { continue }
        secs := int(ev.EventDate.Sub(now).Seconds())
        daysLeft := int(ev.EventDate.Sub(now).Hours() / 24)
        if useFilter && !sourceFilter["event"] { continue }
        items = append(items, models.UnifiedItem{ID: ev.ID.Hex(), Source: "event", Title: ev.Title, ScheduledAt: ev.EventDate, CountdownSeconds: secs, DaysLeft: daysLeft, Importance: ev.ImportanceLevel, DetailURL: "/events", SourceID: ev.ID.Hex()})
    }
    // 提醒
    for _, r := range reminders {
        secs := int(r.ReminderAt.Sub(now).Seconds())
        daysLeft := int(r.ReminderAt.Sub(now).Hours() / 24)
        if useFilter && !sourceFilter["reminder"] { continue }
        items = append(items, models.UnifiedItem{ID: r.ID.Hex(), Source: "reminder", Title: r.Message, ScheduledAt: r.ReminderAt, CountdownSeconds: secs, DaysLeft: daysLeft, Importance: r.Importance, DetailURL: "/reminders", SourceID: r.ID.Hex()})
    }
    // 优先任务 & 去重集
    prioritySet := make(map[string]struct{})
    for _, t := range priorityTasks {
        var sched *time.Time
        if t.Deadline != nil { sched = t.Deadline } else if t.ScheduledDate != nil { sched = t.ScheduledDate }
        if sched == nil || sched.After(end) || sched.Before(now) { continue }
        if useFilter && !sourceFilter["task"] { continue }
        secs := int(sched.Sub(now).Seconds())
        daysLeft := int(sched.Sub(now).Hours() / 24)
        items = append(items, models.UnifiedItem{ID: t.ID, Source: "task", Title: t.Title, ScheduledAt: *sched, CountdownSeconds: secs, DaysLeft: daysLeft, PriorityScore: t.PriorityScore, DetailURL: "/dashboard", SourceID: t.ID})
        prioritySet[t.ID] = struct{}{}
    }
    // 普通任务
    for _, t := range normalTasks {
        if _, exists := prioritySet[t.ID]; exists { continue }
        var sched *time.Time
        if t.Deadline != nil { sched = t.Deadline } else if t.ScheduledDate != nil { sched = t.ScheduledDate }
        if sched == nil || sched.After(end) || sched.Before(now) { continue }
        if useFilter && !sourceFilter["task"] { continue }
        secs := int(sched.Sub(now).Seconds())
        daysLeft := int(sched.Sub(now).Hours() / 24)
        items = append(items, models.UnifiedItem{ID: t.ID, Source: "task", Title: t.Title, ScheduledAt: *sched, CountdownSeconds: secs, DaysLeft: daysLeft, DetailURL: "/dashboard", SourceID: t.ID})
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
    if limit > 0 && len(items) > limit { items = items[:limit] }
    return items
}

func (s *UnifiedService) GetUpcoming(ctx context.Context, userID primitive.ObjectID, hours int, sources []string, limit int) ([]models.UnifiedItem, error) {
    if s == nil || s.db == nil { return nil, errors.New("unified service db not initialized") }
    if hours <= 0 { hours = 24 * 7 }
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
        defer wg.Done(); days := hours / 24; if days < 1 { days = 1 }; events, eventsErr = eventSvc.GetUpcomingEvents(ctx, userID, days) }()
    go func() { // upcoming reminders
        defer wg.Done(); reminders, remindersErr = reminderSvc.GetUpcomingReminders(ctx, userID, hours) }()
    go func() { // priority tasks (already filters by createdBy)
        defer wg.Done(); priorityTasks, priorityErr = taskSortSvc.GetPriorityTasks(ctx, userID) }()
    go func() { // normal tasks with deadline or scheduledDate in window
        defer wg.Done()
        tasksColl := s.db.Collection("tasks")
        // BUGFIX: 原先错误使用 user_id 字段，Task 文档所有者字段是 createdBy(字符串形式的 ObjectID)
        activeStatuses := []string{"pending", "in_progress", "todo", "doing"}
        filter := bson.M{"createdBy": userID.Hex(), "status": bson.M{"$in": activeStatuses}, "$or": []bson.M{{"deadline": bson.M{"$gte": now, "$lte": end}}, {"scheduledDate": bson.M{"$gte": now, "$lte": end}}}}
        opts := options.Find().SetProjection(bson.M{"title":1, "deadline":1, "scheduledDate":1}).SetSort(bson.D{{Key: "deadline", Value: 1}})
        cur, err := tasksColl.Find(ctx, filter, opts); if err != nil { normalTasksErr = err; return }
        defer cur.Close(ctx)
        for cur.Next(ctx) {
            var doc struct { ID string `bson:"_id"`; Title string `bson:"title"`; Deadline *time.Time `bson:"deadline"`; ScheduledDate *time.Time `bson:"scheduledDate"` }
            if err := cur.Decode(&doc); err != nil { continue }
            normalTasks = append(normalTasks, simpleTask{ID: doc.ID, Title: doc.Title, Deadline: doc.Deadline, ScheduledDate: doc.ScheduledDate})
        }
    }()
    wg.Wait()
    _ = eventsErr; _ = remindersErr; _ = priorityErr; _ = normalTasksErr // TODO: 统一日志
    return buildUpcomingItems(now, end, sources, limit, events, reminders, priorityTasks, normalTasks), nil
}

// GetCalendar 聚合指定年月的事件/提醒(按 next_send 落在该月?) + 有日期的任务
func (s *UnifiedService) GetCalendar(ctx context.Context, userID primitive.ObjectID, year, month int) (models.UnifiedCalendarResponse, error) {
    if s == nil || s.db == nil { return models.UnifiedCalendarResponse{}, errors.New("unified service db not initialized") }
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
        if r.ReminderAt.IsZero() { continue }
        if r.ReminderAt.Before(start) || r.ReminderAt.Equal(end) || r.ReminderAt.After(end) { continue }
        dayKey := r.ReminderAt.Format("2006-01-02")
        daysMap[dayKey] = append(daysMap[dayKey], models.UnifiedCalendarItem{ID: r.ID.Hex(), Source: "reminder", Title: r.Message, ScheduledAt: r.ReminderAt, Importance: r.Importance, DetailURL: "/reminders"})
    }

    // 任务 (兼容 createdBy 字符串 / 未来可能的 user_id ObjectID)
    tasksColl := s.db.Collection("tasks")
    tFilter := bson.M{"$and": []bson.M{
        {"$or": []bson.M{{"createdBy": userID.Hex()}, {"user_id": userID}}},
        {"$or": []bson.M{{"deadline": bson.M{"$gte": start, "$lt": end}}, {"scheduledDate": bson.M{"$gte": start, "$lt": end}}}},
    }}
    cur, err := tasksColl.Find(ctx, tFilter, options.Find().SetProjection(bson.M{"title":1, "deadline":1, "scheduledDate":1}))
    if err == nil {
        defer cur.Close(ctx)
        for cur.Next(ctx) {
            var doc struct { ID string `bson:"_id"`; Title string `bson:"title"`; Deadline *time.Time `bson:"deadline"`; ScheduledDate *time.Time `bson:"scheduledDate"` }
            if cur.Decode(&doc) == nil {
                var d *time.Time
                if doc.Deadline != nil { d = doc.Deadline } else { d = doc.ScheduledDate }
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
        sort.Slice(items, func(i,j int) bool { return items[i].ScheduledAt.Before(items[j].ScheduledAt) })
        daysMap[k] = items
        total += len(items)
    }
    return models.UnifiedCalendarResponse{Year: year, Month: month, Days: daysMap, Total: total}, nil
}
