package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// EventComment 事件评论/时间线条目
// Type 可扩展: comment / status_change / attachment / system
// For timeline ordering we rely on CreatedAt ascending.
type EventComment struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	EventID   primitive.ObjectID `bson:"event_id" json:"event_id"`
	UserID    primitive.ObjectID `bson:"user_id" json:"user_id"`
	Type      string             `bson:"type" json:"type"` // comment/status_change/attachment/system
	Content   string             `bson:"content" json:"content"`
	Meta      map[string]string  `bson:"meta,omitempty" json:"meta,omitempty"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

// CreateEventCommentRequest 创建评论请求
type CreateEventCommentRequest struct {
	Content string            `json:"content" validate:"required,min=1,max=2000"`
	Type    string            `json:"type" validate:"omitempty,oneof=comment status_change attachment system"`
	Meta    map[string]string `json:"meta,omitempty"`
}

// UpdateEventCommentRequest 更新评论请求（仅作者允许 & 仅普通评论）
type UpdateEventCommentRequest struct {
	Content *string `json:"content,omitempty" validate:"omitempty,min=1,max=2000"`
}

// EventTimelineItem 聚合后前端展示结构（可含扩展字段）
type EventTimelineItem struct {
	ID        primitive.ObjectID `json:"id"`
	EventID   primitive.ObjectID `json:"event_id"`
	UserID    primitive.ObjectID `json:"user_id"`
	Type      string             `json:"type"`
	Content   string             `json:"content"`
	Meta      map[string]string  `json:"meta,omitempty"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
	// 展示增强
	UserName  string `json:"user_name,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}
