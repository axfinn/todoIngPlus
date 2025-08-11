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

// TaskSortService 任务排序服务
type TaskSortService struct {
	db             *mongo.Database
	taskColl       *mongo.Collection
	eventColl      *mongo.Collection
	reminderColl   *mongo.Collection
	sortConfigColl *mongo.Collection
}

// NewTaskSortService 创建任务排序服务
func NewTaskSortService(db *mongo.Database) *TaskSortService {
	return &TaskSortService{
		db:             db,
		taskColl:       db.Collection("tasks"),
		eventColl:      db.Collection("events"),
		reminderColl:   db.Collection("reminders"),
		sortConfigColl: db.Collection("task_sort_configs"),
	}
}

// GetTaskSortConfig 获取用户的任务排序配置
func (s *TaskSortService) GetTaskSortConfig(ctx context.Context, userID primitive.ObjectID) (*models.TaskSortConfig, error) {
	var config models.TaskSortConfig

	err := s.sortConfigColl.FindOne(ctx, bson.M{"user_id": userID}).Decode(&config)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// 创建默认配置
			config = models.TaskSortConfig{
				UserID:          userID,
				PriorityDays:    7,   // 默认7天内的任务优先显示
				MaxDisplayCount: 20,  // 默认显示20个任务
				WeightUrgent:    0.7, // 紧急权重70%
				WeightImportant: 0.3, // 重要权重30%
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}

			_, err = s.sortConfigColl.InsertOne(ctx, config)
			if err != nil {
				return nil, fmt.Errorf("failed to create default sort config: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get sort config: %w", err)
		}
	}

	return &config, nil
}

// UpdateTaskSortConfig 更新任务排序配置
func (s *TaskSortService) UpdateTaskSortConfig(ctx context.Context, userID primitive.ObjectID, req models.UpdateTaskSortConfigRequest) (*models.TaskSortConfig, error) {
	filter := bson.M{"user_id": userID}

	update := bson.M{
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}

	// 只更新提供的字段
	setFields := update["$set"].(bson.M)
	if req.PriorityDays != nil {
		setFields["priority_days"] = *req.PriorityDays
	}
	if req.MaxDisplayCount != nil {
		setFields["max_display_count"] = *req.MaxDisplayCount
	}
	if req.WeightUrgent != nil {
		setFields["weight_urgent"] = *req.WeightUrgent
	}
	if req.WeightImportant != nil {
		setFields["weight_important"] = *req.WeightImportant
	}

	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	var config models.TaskSortConfig
	err := s.sortConfigColl.FindOneAndUpdate(ctx, filter, update, opts).Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to update sort config: %w", err)
	}

	return &config, nil
}

// CalculateTaskPriority 计算任务优先级分数
func (s *TaskSortService) CalculateTaskPriority(task models.Task, config models.TaskSortConfig) float64 {
	now := time.Now()

	// 如果任务没有截止时间，给一个默认的低优先级
	if task.Deadline == nil {
		return 0.1
	}

	// 计算距离截止时间的天数
	daysUntilDue := task.Deadline.Sub(now).Hours() / 24

	// 紧急程度计算（越接近截止时间分数越高）
	var urgencyScore float64
	if daysUntilDue <= 0 {
		// 已过期的任务优先级最高
		urgencyScore = 1.0
	} else if daysUntilDue <= float64(config.PriorityDays) {
		// 在优先天数内的任务，按比例计算紧急程度
		urgencyScore = math.Max(0, (float64(config.PriorityDays)-daysUntilDue)/float64(config.PriorityDays))
	} else {
		// 超出优先天数的任务，给一个较低的紧急分数
		urgencyScore = math.Max(0, 0.2-(daysUntilDue-float64(config.PriorityDays))/100)
	}

	// 重要程度计算（基于任务状态和优先级）
	var importanceScore float64
	switch task.Status {
	case "pending", "todo":
		importanceScore = 0.8
	case "in_progress", "doing":
		importanceScore = 1.0
	case "completed", "done":
		importanceScore = 0.0
	case "cancelled":
		importanceScore = 0.0
	default:
		importanceScore = 0.5
	}

	// 加入优先级权重
	switch task.Priority {
	case "high":
		importanceScore *= 1.2
	case "medium":
		importanceScore *= 1.0
	case "low":
		importanceScore *= 0.8
	default:
		importanceScore *= 1.0
	}

	// 综合分数计算
	totalScore := urgencyScore*config.WeightUrgent + importanceScore*config.WeightImportant

	// 确保分数在 0-1 之间
	return math.Max(0, math.Min(1, totalScore))
}

// GetPriorityTasks 获取按优先级排序的任务
func (s *TaskSortService) GetPriorityTasks(ctx context.Context, userID primitive.ObjectID) ([]models.PriorityTask, error) {
	// 获取排序配置
	config, err := s.GetTaskSortConfig(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sort config: %w", err)
	}

	// 查询用户的活跃任务
	filter := bson.M{
		"createdBy": userID.Hex(), // 使用字符串类型的用户ID
		"status":    bson.M{"$in": []string{"pending", "in_progress", "todo", "doing"}},
	}

	cursor, err := s.taskColl.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find tasks: %w", err)
	}
	defer cursor.Close(ctx)

	var tasks []models.Task
	if err = cursor.All(ctx, &tasks); err != nil {
		return nil, fmt.Errorf("failed to decode tasks: %w", err)
	}

	// 计算每个任务的优先级分数
	var priorityTasks []models.PriorityTask
	now := time.Now()

	for _, task := range tasks {
		priorityScore := s.CalculateTaskPriority(task, *config)

		var daysLeft int
		if task.Deadline != nil {
			daysLeft = int(task.Deadline.Sub(now).Hours() / 24)
		}

		isUrgent := daysLeft <= config.PriorityDays && daysLeft >= 0

		priorityTask := models.PriorityTask{
			Task:          task,
			PriorityScore: priorityScore,
			DaysLeft:      daysLeft,
			IsUrgent:      isUrgent,
		}

		priorityTasks = append(priorityTasks, priorityTask)
	}

	// 按优先级分数排序（降序）
	for i := 0; i < len(priorityTasks)-1; i++ {
		for j := i + 1; j < len(priorityTasks); j++ {
			if priorityTasks[i].PriorityScore < priorityTasks[j].PriorityScore {
				priorityTasks[i], priorityTasks[j] = priorityTasks[j], priorityTasks[i]
			}
		}
	}

	// 限制返回数量
	if len(priorityTasks) > config.MaxDisplayCount {
		priorityTasks = priorityTasks[:config.MaxDisplayCount]
	}

	return priorityTasks, nil
}

// GetSmartTaskList 获取智能排序的任务列表
func (s *TaskSortService) GetSmartTaskList(ctx context.Context, userID primitive.ObjectID, includeEvents bool) ([]models.PriorityTask, error) {
	priorityTasks, err := s.GetPriorityTasks(ctx, userID)
	if err != nil {
		return nil, err
	}

	if !includeEvents {
		return priorityTasks, nil
	}

	// 如果需要包含事件，可以将即将到来的事件也转换为"任务"
	// 这里可以根据需要实现将事件转换为任务的逻辑

	return priorityTasks, nil
}

// GetDashboardData 获取看板数据
func (s *TaskSortService) GetDashboardData(ctx context.Context, userID primitive.ObjectID) (*models.DashboardData, error) {
	// 并发获取各种数据
	type result struct {
		upcomingEvents   []models.Event
		priorityTasks    []models.PriorityTask
		pendingReminders []models.UpcomingReminder
		eventSummary     models.EventSummary
		taskSummary      models.TaskSummary
		err              error
	}

	resultChan := make(chan result, 1)

	go func() {
		var res result

		// 获取即将到来的事件（7天内）
		eventService := NewEventService(s.db)
		res.upcomingEvents, res.err = eventService.GetUpcomingEvents(ctx, userID, 7)
		if res.err != nil {
			resultChan <- res
			return
		}

		// 获取优先任务
		res.priorityTasks, res.err = s.GetPriorityTasks(ctx, userID)
		if res.err != nil {
			resultChan <- res
			return
		}

		// 获取待处理的提醒
		reminderService := NewReminderService(s.db)
		res.pendingReminders, res.err = reminderService.GetUpcomingReminders(ctx, userID, 24)
		if res.err != nil {
			resultChan <- res
			return
		}

		// 获取事件统计
		res.eventSummary, res.err = s.getEventSummary(ctx, userID)
		if res.err != nil {
			resultChan <- res
			return
		}

		// 获取任务统计
		res.taskSummary, res.err = s.getTaskSummary(ctx, userID)
		if res.err != nil {
			resultChan <- res
			return
		}

		resultChan <- res
	}()

	res := <-resultChan
	if res.err != nil {
		return nil, res.err
	}

	return &models.DashboardData{
		UpcomingEvents:   res.upcomingEvents,
		PriorityTasks:    res.priorityTasks,
		PendingReminders: res.pendingReminders,
		EventSummary:     res.eventSummary,
		TaskSummary:      res.taskSummary,
	}, nil
}

// getEventSummary 获取事件统计
func (s *TaskSortService) getEventSummary(ctx context.Context, userID primitive.ObjectID) (models.EventSummary, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrow := today.AddDate(0, 0, 1)
	weekLater := today.AddDate(0, 0, 7)

	filter := bson.M{
		"user_id":   userID,
		"is_active": true,
	}

	// 总事件数
	totalEvents, err := s.eventColl.CountDocuments(ctx, filter)
	if err != nil {
		return models.EventSummary{}, fmt.Errorf("failed to count total events: %w", err)
	}

	// 即将到来的事件数（7天内）
	upcomingFilter := filter
	upcomingFilter["event_date"] = bson.M{
		"$gte": now,
		"$lte": weekLater,
	}
	upcomingCount, err := s.eventColl.CountDocuments(ctx, upcomingFilter)
	if err != nil {
		return models.EventSummary{}, fmt.Errorf("failed to count upcoming events: %w", err)
	}

	// 今天的事件数
	todayFilter := filter
	todayFilter["event_date"] = bson.M{
		"$gte": today,
		"$lt":  tomorrow,
	}
	todayCount, err := s.eventColl.CountDocuments(ctx, todayFilter)
	if err != nil {
		return models.EventSummary{}, fmt.Errorf("failed to count today events: %w", err)
	}

	// 按类型统计
	pipeline := []bson.M{
		{"$match": filter},
		{
			"$group": bson.M{
				"_id":   "$event_type",
				"count": bson.M{"$sum": 1},
			},
		},
	}

	cursor, err := s.eventColl.Aggregate(ctx, pipeline)
	if err != nil {
		return models.EventSummary{}, fmt.Errorf("failed to aggregate events by type: %w", err)
	}
	defer cursor.Close(ctx)

	typeBreakdown := make(map[string]int64)
	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		typeBreakdown[result.ID] = result.Count
	}

	return models.EventSummary{
		TotalEvents:   totalEvents,
		UpcomingCount: upcomingCount,
		TodayCount:    todayCount,
		TypeBreakdown: typeBreakdown,
	}, nil
}

// getTaskSummary 获取任务统计
func (s *TaskSortService) getTaskSummary(ctx context.Context, userID primitive.ObjectID) (models.TaskSummary, error) {
	filter := bson.M{
		"createdBy": userID.Hex(),
	}

	// 总任务数
	totalTasks, err := s.taskColl.CountDocuments(ctx, filter)
	if err != nil {
		return models.TaskSummary{}, fmt.Errorf("failed to count total tasks: %w", err)
	}

	// 已完成任务数
	completedFilter := bson.M{
		"createdBy": userID.Hex(),
		"status":    bson.M{"$in": []string{"completed", "done"}},
	}
	completedTasks, err := s.taskColl.CountDocuments(ctx, completedFilter)
	if err != nil {
		return models.TaskSummary{}, fmt.Errorf("failed to count completed tasks: %w", err)
	}

	// 计算完成率
	var completedRate float64
	if totalTasks > 0 {
		completedRate = float64(completedTasks) / float64(totalTasks) * 100
	}

	// 过期任务数
	now := time.Now()
	overdueFilter := bson.M{
		"createdBy": userID.Hex(),
		"status":    bson.M{"$in": []string{"pending", "in_progress", "todo", "doing"}},
		"deadline":  bson.M{"$lt": now},
	}
	overdueCount, err := s.taskColl.CountDocuments(ctx, overdueFilter)
	if err != nil {
		return models.TaskSummary{}, fmt.Errorf("failed to count overdue tasks: %w", err)
	}

	// 按状态统计
	pipeline := []bson.M{
		{"$match": filter},
		{
			"$group": bson.M{
				"_id":   "$status",
				"count": bson.M{"$sum": 1},
			},
		},
	}

	cursor, err := s.taskColl.Aggregate(ctx, pipeline)
	if err != nil {
		return models.TaskSummary{}, fmt.Errorf("failed to aggregate tasks by status: %w", err)
	}
	defer cursor.Close(ctx)

	statusCounts := make(map[string]int64)
	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		statusCounts[result.ID] = result.Count
	}

	return models.TaskSummary{
		TotalTasks:    totalTasks,
		CompletedRate: completedRate,
		OverdueCount:  overdueCount,
		StatusCounts:  statusCounts,
	}, nil
}
