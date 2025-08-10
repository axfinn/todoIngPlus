package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Reminder 提醒模型
type Reminder struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	EventID       primitive.ObjectID `bson:"event_id" json:"event_id"`
	UserID        primitive.ObjectID `bson:"user_id" json:"user_id"`
	AdvanceDays   int                `bson:"advance_days" json:"advance_days" validate:"min=0,max=365"`
	ReminderTimes []string           `bson:"reminder_times" json:"reminder_times"` // ["09:00", "18:00"]
	// AbsoluteTimes 允许直接指定绝对提醒时间（UTC）列表，优先于基于事件/AdvanceDays + ReminderTimes 的计算
	AbsoluteTimes []time.Time `bson:"absolute_times,omitempty" json:"absolute_times,omitempty"`
	ReminderType  string      `bson:"reminder_type" json:"reminder_type" validate:"required,oneof=app email both"`
	CustomMessage string      `bson:"custom_message,omitempty" json:"custom_message,omitempty"`
	IsActive      bool        `bson:"is_active" json:"is_active"`
	LastSent      *time.Time  `bson:"last_sent,omitempty" json:"last_sent,omitempty"`
	NextSend      *time.Time  `bson:"next_send,omitempty" json:"next_send,omitempty"`
	CreatedAt     time.Time   `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time   `bson:"updated_at" json:"updated_at"`
}

// CreateReminderRequest 创建提醒请求
type CreateReminderRequest struct {
	EventID       primitive.ObjectID `json:"event_id" validate:"required"`
	AdvanceDays   int                `json:"advance_days" validate:"min=0,max=365"`
	ReminderTimes []string           `json:"reminder_times" validate:"required,min=1"`
	AbsoluteTimes []time.Time        `json:"absolute_times,omitempty"`
	ReminderType  string             `json:"reminder_type" validate:"required,oneof=app email both"`
	CustomMessage string             `json:"custom_message,omitempty"`
}

// UpdateReminderRequest 更新提醒请求
type UpdateReminderRequest struct {
	AdvanceDays   *int         `json:"advance_days,omitempty" validate:"omitempty,min=0,max=365"`
	ReminderTimes []string     `json:"reminder_times,omitempty" validate:"omitempty,min=1"`
	ReminderType  *string      `json:"reminder_type,omitempty" validate:"omitempty,oneof=app email both"`
	CustomMessage *string      `json:"custom_message,omitempty"`
	IsActive      *bool        `json:"is_active,omitempty"`
	AbsoluteTimes *[]time.Time `json:"absolute_times,omitempty"`
}

// ReminderListResponse 提醒列表响应
type ReminderListResponse struct {
	Reminders  []ReminderWithEvent `json:"reminders"`
	Total      int64               `json:"total"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"page_size"`
	TotalPages int                 `json:"total_pages"`
}

// ReminderWithEvent 包含事件信息的提醒
type ReminderWithEvent struct {
	Reminder
	Event Event `json:"event"`
}

// UpcomingReminder 即将到来的提醒
type UpcomingReminder struct {
	ID         primitive.ObjectID `json:"id"`
	EventTitle string             `json:"event_title"`
	EventDate  time.Time          `json:"event_date"`
	Message    string             `json:"message"`
	DaysLeft   int                `json:"days_left"`
	Importance int                `json:"importance"`
	ReminderAt time.Time          `json:"reminder_at"`
}

// TaskSortConfig 任务排序配置
type TaskSortConfig struct {
	UserID          primitive.ObjectID `bson:"user_id" json:"user_id"`
	PriorityDays    int                `bson:"priority_days" json:"priority_days" validate:"min=1,max=30"`          // 优先显示天数，默认7天
	MaxDisplayCount int                `bson:"max_display_count" json:"max_display_count" validate:"min=5,max=100"` // 最大显示数量，默认20
	WeightUrgent    float64            `bson:"weight_urgent" json:"weight_urgent" validate:"min=0,max=1"`           // 紧急权重，默认0.7
	WeightImportant float64            `bson:"weight_important" json:"weight_important" validate:"min=0,max=1"`     // 重要权重，默认0.3
	CreatedAt       time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time          `bson:"updated_at" json:"updated_at"`
}

// UpdateTaskSortConfigRequest 更新任务排序配置请求
type UpdateTaskSortConfigRequest struct {
	PriorityDays    *int     `json:"priority_days,omitempty" validate:"omitempty,min=1,max=30"`
	MaxDisplayCount *int     `json:"max_display_count,omitempty" validate:"omitempty,min=5,max=100"`
	WeightUrgent    *float64 `json:"weight_urgent,omitempty" validate:"omitempty,min=0,max=1"`
	WeightImportant *float64 `json:"weight_important,omitempty" validate:"omitempty,min=0,max=1"`
}

// PriorityTask 优先任务
type PriorityTask struct {
	Task
	PriorityScore float64 `json:"priority_score"`
	DaysLeft      int     `json:"days_left"`
	IsUrgent      bool    `json:"is_urgent"`
}

// DashboardData 看板数据
type DashboardData struct {
	UpcomingEvents   []Event            `json:"upcoming_events"`
	PriorityTasks    []PriorityTask     `json:"priority_tasks"`
	PendingReminders []UpcomingReminder `json:"pending_reminders"`
	EventSummary     EventSummary       `json:"event_summary"`
	TaskSummary      TaskSummary        `json:"task_summary"`
}

// EventSummary 事件统计
type EventSummary struct {
	TotalEvents   int64            `json:"total_events"`
	UpcomingCount int64            `json:"upcoming_count"`
	TodayCount    int64            `json:"today_count"`
	TypeBreakdown map[string]int64 `json:"type_breakdown"`
}

// TaskSummary 任务统计
type TaskSummary struct {
	TotalTasks    int64            `json:"total_tasks"`
	CompletedRate float64          `json:"completed_rate"`
	OverdueCount  int64            `json:"overdue_count"`
	StatusCounts  map[string]int64 `json:"status_counts"`
}

// CalculateNextSendTime 计算下次发送时间
func (r *Reminder) CalculateNextSendTime(event Event) *time.Time {
	if !r.IsActive {
		return nil
	}

	// 若配置了绝对时间列表，取第一个未来时间
	if len(r.AbsoluteTimes) > 0 {
		now := time.Now()
		var next *time.Time
		for _, t := range r.AbsoluteTimes {
			if t.After(now) {
				if next == nil || t.Before(*next) {
					tmp := t
					next = &tmp
				}
			}
		}
		return next
	}

	now := time.Now()

	// 获取事件的下次发生时间
	var eventTime *time.Time
	if event.RecurrenceType == "none" {
		if event.EventDate.After(now) {
			eventTime = &event.EventDate
		} else {
			return nil // 事件已过期
		}
	} else {
		eventTime = event.GetNextOccurrence(now)
		if eventTime == nil {
			return nil
		}
	}

	// 计算提醒日期
	reminderDate := eventTime.AddDate(0, 0, -r.AdvanceDays)

	// 如果提醒日期已过，返回 nil
	if reminderDate.Before(now.Truncate(24 * time.Hour)) {
		return nil
	}

	// 找到今天或之后的第一个提醒时间
	for _, timeStr := range r.ReminderTimes {
		if len(timeStr) == 5 { // HH:MM 格式
			hour := int((timeStr[0]-'0')*10 + (timeStr[1] - '0'))
			minute := int((timeStr[3]-'0')*10 + (timeStr[4] - '0'))

			reminderTime := time.Date(
				reminderDate.Year(), reminderDate.Month(), reminderDate.Day(),
				hour, minute, 0, 0, reminderDate.Location(),
			)

			// 如果这个时间在未来，返回它
			if reminderTime.After(now) {
				return &reminderTime
			}
		}
	}

	return nil
}

// ShouldSendReminder 检查是否应该发送提醒
func (r *Reminder) ShouldSendReminder() bool {
	if !r.IsActive || r.NextSend == nil {
		return false
	}

	now := time.Now()
	return r.NextSend.Before(now) || r.NextSend.Equal(now)
}
