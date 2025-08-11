package services

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/axfinn/todoIngPlus/backend-go/internal/models"
	"github.com/axfinn/todoIngPlus/backend-go/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ReminderService 精简：只组合仓储层
type ReminderService struct {
	repo repository.ReminderRepository
}

// NewReminderService 创建提醒服务
func NewReminderService(repo repository.ReminderRepository) *ReminderService {
	return &ReminderService{repo: repo}
}

// CreateReminder 创建提醒
func (s *ReminderService) CreateReminder(ctx context.Context, userID primitive.ObjectID, req models.CreateReminderRequest) (*models.Reminder, error) {
	if s.repo == nil {
		return nil, errors.New("reminder repo nil")
	}
	r := &models.Reminder{
		ID:            primitive.NewObjectID(),
		EventID:       req.EventID,
		UserID:        userID,
		AdvanceDays:   req.AdvanceDays,
		ReminderTimes: req.ReminderTimes,
		AbsoluteTimes: req.AbsoluteTimes,
		ReminderType:  req.ReminderType,
		CustomMessage: req.CustomMessage,
		IsActive:      true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := s.repo.Insert(ctx, r); err != nil {
		return nil, err
	}
	return r, nil
}

// GetReminder 获取单个提醒
func (s *ReminderService) GetReminder(ctx context.Context, userID, reminderID primitive.ObjectID) (*models.ReminderWithEvent, error) {
	return s.repo.GetWithEvent(ctx, userID, reminderID)
}

// UpdateReminder 更新提醒
func (s *ReminderService) UpdateReminder(ctx context.Context, userID, reminderID primitive.ObjectID, req models.UpdateReminderRequest) (*models.Reminder, error) {
	set := map[string]interface{}{}
	if req.AdvanceDays != nil {
		set["advance_days"] = *req.AdvanceDays
	}
	if req.ReminderTimes != nil {
		set["reminder_times"] = req.ReminderTimes
	}
	if req.ReminderType != nil {
		set["reminder_type"] = *req.ReminderType
	}
	if req.CustomMessage != nil {
		set["custom_message"] = *req.CustomMessage
	}
	if req.IsActive != nil {
		set["is_active"] = *req.IsActive
	}
	if req.AbsoluteTimes != nil {
		set["absolute_times"] = *req.AbsoluteTimes
	}
	if len(set) == 0 { // nothing to do
		res, err := s.repo.GetWithEvent(ctx, userID, reminderID)
		if err != nil {
			return nil, err
		}
		return &res.Reminder, nil
	}
	return s.repo.UpdateFields(ctx, userID, reminderID, set, true)
}

// DeleteReminder 删除提醒
func (s *ReminderService) DeleteReminder(ctx context.Context, userID, reminderID primitive.ObjectID) error {
	return s.repo.Delete(ctx, userID, reminderID)
}

// ListReminders 获取提醒列表
func (s *ReminderService) ListReminders(ctx context.Context, userID primitive.ObjectID, page, pageSize int, activeOnly bool) (*models.ReminderListResponse, error) {
	return s.repo.ListPagedWithEvent(ctx, userID, page, pageSize, activeOnly)
}

// ListSimpleReminders 返回简化列表（无分页，最多100条，便于前端快速展示）
func (s *ReminderService) ListSimpleReminders(ctx context.Context, userID primitive.ObjectID, activeOnly bool, limit int) ([]repository.SimpleReminderDTO, error) {
	return s.repo.ListSimple(ctx, userID, activeOnly, limit)
}

// GetUpcomingReminders 获取即将到来的提醒
func (s *ReminderService) GetUpcomingReminders(ctx context.Context, userID primitive.ObjectID, hours int) ([]models.UpcomingReminder, error) {
	return s.repo.Upcoming(ctx, userID, hours)
}

// GetPendingReminders 获取待发送的提醒
func (s *ReminderService) GetPendingReminders(ctx context.Context) ([]models.ReminderWithEvent, error) {
	return s.repo.Pending(ctx)
}

// MarkReminderSent 标记提醒已发送
func (s *ReminderService) MarkReminderSent(ctx context.Context, reminderID primitive.ObjectID) error {
	return s.repo.MarkSent(ctx, reminderID)
}

// SnoozeReminder 暂停提醒（推迟指定时间）
func (s *ReminderService) SnoozeReminder(ctx context.Context, userID, reminderID primitive.ObjectID, snoozeMinutes int) error {
	return s.repo.Snooze(ctx, userID, reminderID, snoozeMinutes)
}

// ToggleReminderActive 反转提醒激活状态
func (s *ReminderService) ToggleReminderActive(ctx context.Context, userID, reminderID primitive.ObjectID) (bool, error) {
	return s.repo.ToggleActive(ctx, userID, reminderID)
}

// CreateImmediateTestReminder 创建一个立即发送/短延迟的测试提醒（不通过正常 next_send 计算）
func (s *ReminderService) CreateImmediateTestReminder(ctx context.Context, userID, eventID primitive.ObjectID, message string, delaySeconds int) (*models.Reminder, *models.Event, error) {
	return s.repo.CreateImmediateTest(ctx, userID, eventID, message, delaySeconds)
}

// PreviewReminder 复用仓储 Preview，额外执行基础校验与时间字符串规范化
func (s *ReminderService) PreviewReminder(ctx context.Context, userID, eventID primitive.ObjectID, advanceDays int, times []string) (*repository.PreviewReminderOutput, error) {
	if advanceDays < 0 || advanceDays > 365 { // 轻度校验
		advanceDays = 0
	}
	clean := make([]string, 0, len(times))
	for _, t := range times {
		tt := strings.TrimSpace(t)
		if tt == "" {
			continue
		}
		if len(tt) == 5 && tt[2] == ':' {
			clean = append(clean, tt)
		}
	}
	sort.Strings(clean)
	return s.repo.Preview(ctx, userID, eventID, advanceDays, clean)
}
