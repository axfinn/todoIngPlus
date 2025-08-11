package services

import (
	"context"
	"errors"
	"time"

	"github.com/axfinn/todoIngPlus/backend-go/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TaskService 抽离出的任务领域服务，供 gRPC / HTTP 共用
// 只处理业务与数据访问，不关心传输层（status code / proto）
type TaskService struct {
	db *mongo.Database
}

func NewTaskService(db *mongo.Database) *TaskService { return &TaskService{db: db} }

// Create 新建任务
func (s *TaskService) Create(ctx context.Context, userID string, in models.Task) (*models.Task, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("task service db not init")
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
	res, err := s.db.Collection("tasks").InsertOne(ctx, &in)
	if err != nil {
		return nil, err
	}
	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		in.ID = oid.Hex()
	}
	return &in, nil
}

// List 任务分页
func (s *TaskService) List(ctx context.Context, userID string, status string, page, limit int64) ([]models.Task, int64, error) {
	if s == nil || s.db == nil {
		return nil, 0, errors.New("task service db not init")
	}
	if userID == "" {
		return nil, 0, errors.New("user id missing")
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if page <= 0 {
		page = 1
	}
	coll := s.db.Collection("tasks")
	filter := bson.M{"createdBy": userID}
	if status != "" {
		filter["status"] = status
	}
	findOpts := options.Find().SetLimit(limit).SetSkip((page - 1) * limit)
	cur, err := coll.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	var list []models.Task
	for cur.Next(ctx) {
		var m models.Task
		if cur.Decode(&m) == nil {
			list = append(list, m)
		}
	}
	// total 简化：若需要精确可另行 CountDocuments
	return list, int64(len(list)), cur.Err()
}

// Get 单条任务
func (s *TaskService) Get(ctx context.Context, userID, id string) (*models.Task, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("task service db not init")
	}
	if userID == "" || id == "" {
		return nil, errors.New("invalid params")
	}
	coll := s.db.Collection("tasks")
	filter := bson.M{"createdBy": userID, "$or": []bson.M{{"_id": id}}}
	if oid, err := primitive.ObjectIDFromHex(id); err == nil {
		filter["$or"] = append(filter["$or"].([]bson.M), bson.M{"_id": oid})
	}
	var m models.Task
	if err := coll.FindOne(ctx, filter).Decode(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

// Update 更新任务（部分字段）
func (s *TaskService) Update(ctx context.Context, userID string, req models.TaskUpdateRequest) (*models.Task, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("task service db not init")
	}
	if userID == "" || req.ID == "" {
		return nil, errors.New("invalid params")
	}
	coll := s.db.Collection("tasks")
	set := bson.M{"updatedAt": time.Now()}
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
	filter := bson.M{"createdBy": userID, "$or": []bson.M{{"_id": req.ID}}}
	if oid, err := primitive.ObjectIDFromHex(req.ID); err == nil {
		filter["$or"] = append(filter["$or"].([]bson.M), bson.M{"_id": oid})
	}
	if _, err := coll.UpdateOne(ctx, filter, bson.M{"$set": set}); err != nil {
		return nil, err
	}
	return s.Get(ctx, userID, req.ID)
}

// Delete 删除任务
func (s *TaskService) Delete(ctx context.Context, userID, id string) error {
	if s == nil || s.db == nil {
		return errors.New("task service db not init")
	}
	if userID == "" || id == "" {
		return errors.New("invalid params")
	}
	coll := s.db.Collection("tasks")
	filter := bson.M{"createdBy": userID, "$or": []bson.M{{"_id": id}}}
	if oid, err := primitive.ObjectIDFromHex(id); err == nil {
		filter["$or"] = append(filter["$or"].([]bson.M), bson.M{"_id": oid})
	}
	_, err := coll.DeleteOne(ctx, filter)
	return err
}
