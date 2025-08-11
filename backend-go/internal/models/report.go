package models

import "time"

type Statistics struct {
	TotalTasks      int `bson:"totalTasks" json:"totalTasks"`
	CompletedTasks  int `bson:"completedTasks" json:"completedTasks"`
	InProgressTasks int `bson:"inProgressTasks" json:"inProgressTasks"`
	OverdueTasks    int `bson:"overdueTasks" json:"overdueTasks"`
	CompletionRate  int `bson:"completionRate" json:"completionRate"`
	// 扩展: 事件 & 提醒统计（不影响现有前端, 先存储内部备用）
	TotalEvents     int `bson:"totalEvents,omitempty" json:"totalEvents,omitempty"`
	UpcomingEvents  int `bson:"upcomingEvents,omitempty" json:"upcomingEvents,omitempty"`
	TotalReminders  int `bson:"totalReminders,omitempty" json:"totalReminders,omitempty"`
	ActiveReminders int `bson:"activeReminders,omitempty" json:"activeReminders,omitempty"`
}

type Report struct {
	ID              string     `bson:"_id,omitempty" json:"id"`
	UserID          string     `bson:"userId" json:"userId"`
	Type            string     `bson:"type" json:"type"`
	Period          string     `bson:"period" json:"period"`
	Title           string     `bson:"title" json:"title"`
	Content         string     `bson:"content" json:"content"`
	PolishedContent *string    `bson:"polishedContent" json:"polishedContent"`
	Tasks           []string   `bson:"tasks" json:"tasks"`
	Statistics      Statistics `bson:"statistics" json:"statistics"`
	CreatedAt       time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time  `bson:"updatedAt" json:"updatedAt"`
}
