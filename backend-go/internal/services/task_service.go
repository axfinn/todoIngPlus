package services

import (
	"context"
	"errors"
	"time"

	"github.com/axfinn/todoIngPlus/backend-go/internal/models"
	"github.com/axfinn/todoIngPlus/backend-go/internal/repository"
)

// TaskService 抽离出的任务领域服务
// 使用 repository 进行数据访问
type TaskService struct {
	repo repository.TaskRepository
}

func NewTaskService(db repository.TaskRepository) *TaskService { return &TaskService{repo: db} }

// Create 新建任务
func (s *TaskService) Create(ctx context.Context, userID string, in models.Task) (*models.Task, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("task service not init")
	}
	if userID == "" {
		return nil, errors.New("user id missing")
	}
	if in.Title == "" {
		return nil, errors.New("title required")
	}
	now := time.Now()
	in.CreatedBy = userID
	in.CreatedAt = now
	in.UpdatedAt = now
	if in.Status == "" {
		in.Status = "Todo"
	}
	if in.Priority == "" {
		in.Priority = "Medium"
	}
	if err := s.repo.Insert(ctx, &in); err != nil {
		return nil, err
	}
	return &in, nil
}

// List 任务分页
func (s *TaskService) List(ctx context.Context, userID string, status string, page, limit int64) ([]models.Task, int64, error) {
	if s == nil || s.repo == nil {
		return nil, 0, errors.New("task service not init")
	}
	if userID == "" {
		return nil, 0, errors.New("user id missing")
	}
	return s.repo.List(ctx, userID, status, page, limit)
}

// Get 单条任务
func (s *TaskService) Get(ctx context.Context, userID, id string) (*models.Task, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("task service not init")
	}
	if userID == "" || id == "" {
		return nil, errors.New("invalid params")
	}
	return s.repo.FindByID(ctx, userID, id)
}

// Update 更新任务（部分字段）
func (s *TaskService) Update(ctx context.Context, userID string, req models.TaskUpdateRequest) (*models.Task, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("task service not init")
	}
	if userID == "" || req.ID == "" {
		return nil, errors.New("invalid params")
	}
	set := map[string]interface{}{"updatedAt": time.Now()}
	if req.Title != nil {
		set["title"] = *req.Title
	}
	if req.Description != nil {
		set["description"] = *req.Description
	}
	if req.Assignee != nil {
		set["assignee"] = *req.Assignee
	}
	if req.Status != nil {
		set["status"] = *req.Status
	}
	if req.Priority != nil {
		set["priority"] = *req.Priority
	}
	if req.Deadline != nil {
		set["deadline"] = *req.Deadline
	}
	if req.ScheduledDate != nil {
		set["scheduledDate"] = *req.ScheduledDate
	}
	if len(req.Comments) > 0 {
		set["comments"] = req.Comments
	}
	// 类型转换
	bset := make(map[string]interface{}, len(set))
	for k, v := range set {
		bset[k] = v
	}
	// repository UpdatePartial 需要 bson.M; 这里直接断言即可
	return s.repo.UpdatePartial(ctx, userID, req.ID, bset)
}

// Delete 删除任务
func (s *TaskService) Delete(ctx context.Context, userID, id string) error {
	if s == nil || s.repo == nil {
		return errors.New("task service not init")
	}
	if userID == "" || id == "" {
		return errors.New("invalid params")
	}
	return s.repo.Delete(ctx, userID, id)
}
