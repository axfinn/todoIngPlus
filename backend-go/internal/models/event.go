package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Event 事件模型
type Event struct {
	ID               primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	UserID           primitive.ObjectID     `bson:"user_id" json:"user_id"`
	Title            string                 `bson:"title" json:"title" validate:"required,min=1,max=100"`
	Description      string                 `bson:"description" json:"description"`
	EventType        string                 `bson:"event_type" json:"event_type" validate:"required,oneof=birthday anniversary holiday custom meeting deadline"`
	EventDate        time.Time              `bson:"event_date" json:"event_date" validate:"required"`
	RecurrenceType   string                 `bson:"recurrence_type" json:"recurrence_type" validate:"oneof=none yearly monthly weekly daily"`
	RecurrenceConfig map[string]interface{} `bson:"recurrence_config,omitempty" json:"recurrence_config,omitempty"`
	ImportanceLevel  int                    `bson:"importance_level" json:"importance_level" validate:"min=1,max=5"`
	Tags             []string               `bson:"tags" json:"tags"`
	Location         string                 `bson:"location,omitempty" json:"location,omitempty"`
	IsAllDay         bool                   `bson:"is_all_day" json:"is_all_day"`
	CreatedAt        time.Time              `bson:"created_at" json:"created_at"`
	UpdatedAt        time.Time              `bson:"updated_at" json:"updated_at"`
	IsActive         bool                   `bson:"is_active" json:"is_active"`
	LastTriggeredAt  *time.Time             `bson:"last_triggered_at,omitempty" json:"last_triggered_at,omitempty"` // 系统自动时间线记录最近一次事件开始触发时间
}

// CreateEventRequest 创建事件请求
type CreateEventRequest struct {
	Title            string                 `json:"title" validate:"required,min=1,max=100"`
	Description      string                 `json:"description"`
	EventType        string                 `json:"event_type" validate:"required,oneof=birthday anniversary holiday custom meeting deadline"`
	EventDate        time.Time              `json:"event_date" validate:"required"`
	RecurrenceType   string                 `json:"recurrence_type" validate:"oneof=none yearly monthly weekly daily"`
	RecurrenceConfig map[string]interface{} `json:"recurrence_config,omitempty"`
	ImportanceLevel  int                    `json:"importance_level" validate:"min=1,max=5"`
	Tags             []string               `json:"tags"`
	Location         string                 `json:"location,omitempty"`
	IsAllDay         bool                   `json:"is_all_day"`
}

// UpdateEventRequest 更新事件请求
type UpdateEventRequest struct {
	Title            *string                `json:"title,omitempty" validate:"omitempty,min=1,max=100"`
	Description      *string                `json:"description,omitempty"`
	EventType        *string                `json:"event_type,omitempty" validate:"omitempty,oneof=birthday anniversary holiday custom meeting deadline"`
	EventDate        *time.Time             `json:"event_date,omitempty"`
	RecurrenceType   *string                `json:"recurrence_type,omitempty" validate:"omitempty,oneof=none yearly monthly weekly daily"`
	RecurrenceConfig map[string]interface{} `json:"recurrence_config,omitempty"`
	ImportanceLevel  *int                   `json:"importance_level,omitempty" validate:"omitempty,min=1,max=5"`
	Tags             []string               `json:"tags,omitempty"`
	Location         *string                `json:"location,omitempty"`
	IsAllDay         *bool                  `json:"is_all_day,omitempty"`
	IsActive         *bool                  `json:"is_active,omitempty"`
}

// EventListResponse 事件列表响应
type EventListResponse struct {
	Events     []Event `json:"events"`
	Total      int64   `json:"total"`
	Page       int     `json:"page"`
	PageSize   int     `json:"page_size"`
	TotalPages int     `json:"total_pages"`
}

// EventCalendarResponse 日历视图响应
type EventCalendarResponse struct {
	Date   string  `json:"date"`
	Events []Event `json:"events"`
}

// GetNextOccurrence 获取下次发生时间
func (e *Event) GetNextOccurrence(after time.Time) *time.Time {
	if e.RecurrenceType == "none" {
		if e.EventDate.After(after) {
			return &e.EventDate
		}
		return nil
	}

	baseDate := e.EventDate
	next := baseDate

	switch e.RecurrenceType {
	case "yearly":
		// 年度循环：每年同一天
		year := after.Year()
		if after.Month() > baseDate.Month() ||
			(after.Month() == baseDate.Month() && after.Day() >= baseDate.Day()) {
			year++
		}
		next = time.Date(year, baseDate.Month(), baseDate.Day(),
			baseDate.Hour(), baseDate.Minute(), baseDate.Second(), 0, baseDate.Location())

	case "monthly":
		// 月度循环：每月同一天
		year, month := after.Year(), after.Month()
		if after.Day() >= baseDate.Day() {
			month++
			if month > 12 {
				year++
				month = 1
			}
		}
		next = time.Date(year, month, baseDate.Day(),
			baseDate.Hour(), baseDate.Minute(), baseDate.Second(), 0, baseDate.Location())

	case "weekly":
		// 周循环：每周同一天
		days := int(after.Sub(baseDate).Hours() / 24)
		weeks := (days / 7) + 1
		next = baseDate.AddDate(0, 0, weeks*7)

	case "daily":
		// 日循环：每天
		days := int(after.Sub(baseDate).Hours()/24) + 1
		next = baseDate.AddDate(0, 0, days)
	}

	return &next
}

// IsUpcoming 检查是否为即将到来的事件（7天内）
func (e *Event) IsUpcoming() bool {
	now := time.Now()
	sevenDaysLater := now.AddDate(0, 0, 7)

	if e.RecurrenceType == "none" {
		return e.EventDate.After(now) && e.EventDate.Before(sevenDaysLater)
	}

	nextOccurrence := e.GetNextOccurrence(now)
	if nextOccurrence == nil {
		return false
	}

	return nextOccurrence.Before(sevenDaysLater)
}
