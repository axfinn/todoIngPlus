package models

import (
    "time"
    "go.mongodb.org/mongo-driver/bson/primitive"
)

// Notification 用户应用内通知
type Notification struct {
    ID        primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
    UserID    primitive.ObjectID     `bson:"user_id" json:"user_id"`
    Type      string                 `bson:"type" json:"type"`
    Message   string                 `bson:"message" json:"message"`
    EventID   *primitive.ObjectID    `bson:"event_id,omitempty" json:"event_id,omitempty"`
    ReadAt    *time.Time             `bson:"read_at,omitempty" json:"read_at,omitempty"`
    CreatedAt time.Time              `bson:"created_at" json:"created_at"`
    Metadata  map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
}

// NotificationCreate 用于创建
type NotificationCreate struct {
    UserID  primitive.ObjectID
    Type    string
    Message string
    EventID *primitive.ObjectID
    Metadata map[string]interface{}
}
