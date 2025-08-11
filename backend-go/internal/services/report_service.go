package services

import (
	"context"
	"errors"
	"time"

	"github.com/axfinn/todoIngPlus/backend-go/internal/models"
	"github.com/axfinn/todoIngPlus/backend-go/internal/repository"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// ReportService 负责聚合统计（目前任务 + 事件 + 提醒）
type ReportService struct {
	db   *mongo.Database
	repo repository.ReportRepository
}

func NewReportService(db *mongo.Database) *ReportService { return &ReportService{db: db, repo: repository.NewReportRepository(db)} }

// Generate 聚合并持久化报表（不改变外部 proto 接口）
func (s *ReportService) Generate(ctx context.Context, userID string, in models.Report, start, end time.Time) (*models.Report, error) {
	if s == nil || s.db == nil || s.repo == nil {
		return nil, errors.New("report service not init")
	}
	if userID == "" {
		return nil, errors.New("user id missing")
	}
	if in.Title == "" {
		return nil, errors.New("title required")
	}
	// 选取时间范围
	if start.IsZero() || end.IsZero() || end.Before(start) {
		end = time.Now()
		switch in.Type {
		case "weekly":
			start = end.AddDate(0, 0, -7)
		case "monthly":
			start = end.AddDate(0, -1, 0)
		default:
			start = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, end.Location())
		}
	}
	// Tasks
	taskColl := s.db.Collection("tasks")
	taskFilter := bson.M{"createdBy": userID, "createdAt": bson.M{"$gte": start, "$lte": end}}
	taskCur, err := taskColl.Find(ctx, taskFilter)
	if err != nil {
		return nil, err
	}
	defer taskCur.Close(ctx)
	var tasks []models.Task
	for taskCur.Next(ctx) {
		var t models.Task
		if taskCur.Decode(&t) == nil {
			tasks = append(tasks, t)
		}
	}
	// Events
	eventColl := s.db.Collection("events")
	eventFilter := bson.M{"user_id": userID, "event_date": bson.M{"$gte": start, "$lte": end}}
	eventCur, _ := eventColl.Find(ctx, eventFilter)
	var totalEvents, upcomingEvents int
	var eventIDs []string
	if eventCur != nil {
		defer eventCur.Close(ctx)
		now := time.Now()
		for eventCur.Next(ctx) {
			var ev models.Event
			if eventCur.Decode(&ev) == nil {
				totalEvents++
				if ev.EventDate.After(now) {
					upcomingEvents++
				}
				eventIDs = append(eventIDs, ev.ID.Hex())
			}
		}
	}
	// Reminders
	reminderColl := s.db.Collection("reminders")
	remFilter := bson.M{"user_id": userID}
	remCur, _ := reminderColl.Find(ctx, remFilter)
	var totalReminders, activeReminders int
	if remCur != nil {
		defer remCur.Close(ctx)
		for remCur.Next(ctx) {
			var r models.Reminder
			if remCur.Decode(&r) == nil {
				totalReminders++
				if r.IsActive {
					activeReminders++
				}
			}
		}
	}

	now := time.Now()
	in.UserID = userID
	in.CreatedAt = now
	in.UpdatedAt = now
	in.Period = in.Type
	// 统计任务
	var total, completed, inProgress, overdue int
	var taskIDs []string
	for i := range tasks {
		t := tasks[i]
		taskIDs = append(taskIDs, t.ID)
		total++
		switch t.Status {
		case "Done", "done", "DONE":
			completed++
		case "InProgress", "in_progress", "IN_PROGRESS":
			inProgress++
		}
		if t.Deadline != nil && !t.Deadline.IsZero() && t.Deadline.Before(now) && !(t.Status == "Done" || t.Status == "done" || t.Status == "DONE") {
			overdue++
		}
	}
	var completionRate float64
	if total > 0 {
		completionRate = float64(completed) / float64(total) * 100.0
	}
	in.Tasks = taskIDs
	in.Statistics.TotalTasks = total
	in.Statistics.CompletedTasks = completed
	in.Statistics.InProgressTasks = inProgress
	in.Statistics.OverdueTasks = overdue
	in.Statistics.CompletionRate = int(completionRate)
	in.Statistics.TotalEvents = totalEvents
	in.Statistics.UpcomingEvents = upcomingEvents
	in.Statistics.TotalReminders = totalReminders
	in.Statistics.ActiveReminders = activeReminders
	// 持久化
	if err := s.repo.Insert(ctx, &in); err != nil {
		return nil, err
	}
	return &in, nil
}

func (s *ReportService) List(ctx context.Context, userID string) ([]models.Report, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("service not init")
	}
	return s.repo.ListByUser(ctx, userID)
}

func (s *ReportService) Get(ctx context.Context, userID, id string) (*models.Report, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("service not init")
	}
	return s.repo.FindByID(ctx, userID, id)
}

func (s *ReportService) Delete(ctx context.Context, userID, id string) (bool, error) {
	if s == nil || s.repo == nil {
		return false, errors.New("service not init")
	}
	return s.repo.Delete(ctx, userID, id)
}
