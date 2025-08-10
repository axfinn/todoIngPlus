package models

import "time"

// UnifiedItem 统一聚合条目
type UnifiedItem struct {
	ID               string    `json:"id"`
	Source           string    `json:"source"` // task | event | reminder
	Title            string    `json:"title"`
	ScheduledAt      time.Time `json:"scheduled_at"`
	CountdownSeconds int       `json:"countdown_seconds"`
	DaysLeft         int       `json:"days_left"`
	Importance       int       `json:"importance,omitempty"`
	PriorityScore    float64   `json:"priority_score,omitempty"`
	RelatedEventID   string    `json:"related_event_id,omitempty"`
	DetailURL        string    `json:"detail_url,omitempty"`
	SourceID         string    `json:"source_id,omitempty"`      // 原始对象ID（与 ID 相同或不同）
	IsUnscheduled    bool      `json:"is_unscheduled,omitempty"` // 对于无日期临时展示的任务
}

// UnifiedUpcomingResponse 响应
type UnifiedUpcomingResponse struct {
	Items           []UnifiedItem         `json:"items"`
	Hours           int                   `json:"hours"`
	Total           int                   `json:"total"`
	ServerTimestamp int64                 `json:"server_timestamp"` // 服务器生成 Unix 秒，用于客户端计算延迟
	Stats           *UnifiedUpcomingStats `json:"stats,omitempty"`
	Debug           *UnifiedUpcomingDebug `json:"debug,omitempty"`
}

// UnifiedUpcomingStats 统计拆分
type UnifiedUpcomingStats struct {
	Tasks     int `json:"tasks"`
	Events    int `json:"events"`
	Reminders int `json:"reminders"`
}

// UnifiedUpcomingDebug 调试信息（仅 debug=1 时返回）
type UnifiedUpcomingDebug struct {
	GraceDays         int                    `json:"grace_days"`
	Hours             int                    `json:"hours"`
	Now               string                 `json:"now"`
	End               string                 `json:"end"`
	WindowFilter      map[string]interface{} `json:"window_filter"`
	UnscheduledFilter map[string]interface{} `json:"unscheduled_filter"`
	WindowCounts      struct {
		Deadline    int `json:"deadline"`
		Scheduled   int `json:"scheduled"`
		DueDateOnly int `json:"due_date_only"`
		Total       int `json:"total"`
	} `json:"window_counts"`
	AddedUnscheduled int `json:"added_unscheduled"`
}

// UnifiedCalendarItem 日历条目（不需要倒计时精度，可复用主要字段）
type UnifiedCalendarItem struct {
	ID          string    `json:"id"`
	Source      string    `json:"source"`
	Title       string    `json:"title"`
	ScheduledAt time.Time `json:"scheduled_at"`
	Importance  int       `json:"importance,omitempty"`
	DetailURL   string    `json:"detail_url,omitempty"`
}

type UnifiedCalendarResponse struct {
	Year            int                              `json:"year"`
	Month           int                              `json:"month"`
	Days            map[string][]UnifiedCalendarItem `json:"days"`
	Total           int                              `json:"total"`
	ServerTimestamp int64                            `json:"server_timestamp"`
}
