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
    SourceID         string    `json:"source_id,omitempty"` // 原始对象ID（与 ID 相同或不同）
}

// UnifiedUpcomingResponse 响应
type UnifiedUpcomingResponse struct {
    Items []UnifiedItem `json:"items"`
    Hours int           `json:"hours"`
    Total int           `json:"total"`
    ServerTimestamp int64 `json:"server_timestamp"` // 服务器生成 Unix 秒，用于客户端计算延迟
    Stats *UnifiedUpcomingStats `json:"stats,omitempty"`
}

// UnifiedUpcomingStats 统计拆分
type UnifiedUpcomingStats struct {
    Tasks int `json:"tasks"`
    Events int `json:"events"`
    Reminders int `json:"reminders"`
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
    Year  int                              `json:"year"`
    Month int                              `json:"month"`
    Days  map[string][]UnifiedCalendarItem `json:"days"`
    Total int                              `json:"total"`
    ServerTimestamp int64 `json:"server_timestamp"`
}
