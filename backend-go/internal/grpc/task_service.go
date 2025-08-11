package grpcserver

import (
	"context"
	"errors"
	"time"

	"github.com/axfinn/todoIngPlus/backend-go/internal/convert"
	"github.com/axfinn/todoIngPlus/backend-go/internal/models"
	"github.com/axfinn/todoIngPlus/backend-go/internal/services"
	pb "github.com/axfinn/todoIngPlus/backend-go/pkg/api/v1"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TaskServiceServer 实现 pb.TaskService
type TaskServiceServer struct {
	pb.UnimplementedTaskServiceServer
	core *services.TaskService
	db   *mongo.Database
}

func NewTaskServiceServer(db *mongo.Database) *TaskServiceServer {
	return &TaskServiceServer{core: services.NewTaskService(db), db: db}
}

// helper 保留
func taskModelToProto(t *models.Task) *pb.Task {
	if t == nil {
		return nil
	}
	return convert.TaskToProto(t)
}

// CreateTask
func (s *TaskServiceServer) CreateTask(ctx context.Context, req *pb.CreateTaskRequest) (*pb.CreateTaskResponse, error) {
	if s.core == nil {
		return nil, status.Error(codes.FailedPrecondition, "service not init")
	}
	if req == nil || req.Title == "" {
		return nil, status.Error(codes.InvalidArgument, "title required")
	}
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	m := models.Task{Title: req.Title, Description: req.Description, Status: map[pb.TaskStatus]string{pb.TaskStatus_TASK_STATUS_TODO: "Todo", pb.TaskStatus_TASK_STATUS_IN_PROGRESS: "InProgress", pb.TaskStatus_TASK_STATUS_DONE: "Done"}[req.Status], Priority: map[pb.TaskPriority]string{pb.TaskPriority_TASK_PRIORITY_LOW: "Low", pb.TaskPriority_TASK_PRIORITY_MEDIUM: "Medium", pb.TaskPriority_TASK_PRIORITY_HIGH: "High"}[req.Priority]}
	if req.Deadline != nil {
		d := req.Deadline.AsTime()
		m.Deadline = &d
	}
	if req.ScheduledDate != nil {
		sd := req.ScheduledDate.AsTime()
		m.ScheduledDate = &sd
	}
	if req.Assignee != "" {
		a := req.Assignee
		m.Assignee = &a
	}
	res, err := s.core.Create(ctx, uid, m)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create err: %v", err)
	}
	return &pb.CreateTaskResponse{Response: &pb.Response{Code: 201, Message: "created"}, Task: taskModelToProto(res)}, nil
}

// GetTasks 列表
func (s *TaskServiceServer) GetTasks(ctx context.Context, req *pb.GetTasksRequest) (*pb.GetTasksResponse, error) {
	if s.core == nil {
		return nil, status.Error(codes.FailedPrecondition, "service not init")
	}
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	statusMap := map[pb.TaskStatus]string{pb.TaskStatus_TASK_STATUS_TODO: "Todo", pb.TaskStatus_TASK_STATUS_IN_PROGRESS: "InProgress", pb.TaskStatus_TASK_STATUS_DONE: "Done"}
	var st string
	if req.Status != pb.TaskStatus_TASK_STATUS_UNSPECIFIED {
		st = statusMap[req.Status]
	}
	var page, limit int64 = 1, 50
	if req.Pagination != nil {
		if req.Pagination.Page > 0 {
			page = int64(req.Pagination.Page)
		}
		if req.Pagination.Limit > 0 {
			limit = int64(req.Pagination.Limit)
		}
	}
	list, total, err := s.core.List(ctx, uid, st, page, limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list err: %v", err)
	}
	var tasks []*pb.Task
	for i := range list {
		tasks = append(tasks, taskModelToProto(&list[i]))
	}
	pg := &pb.PaginationResponse{Limit: int32(limit), Total: int32(total)}
	return &pb.GetTasksResponse{Response: &pb.Response{Code: 200, Message: "ok"}, Tasks: tasks, Pagination: pg}, nil
}

// GetTask 详情
func (s *TaskServiceServer) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.GetTaskResponse, error) {
	if s.core == nil {
		return nil, status.Error(codes.FailedPrecondition, "service not init")
	}
	if req == nil || req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	m, err := s.core.Get(ctx, uid, req.Id)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, status.Error(codes.NotFound, "task not found")
		}
		return nil, status.Errorf(codes.Internal, "get err: %v", err)
	}
	return &pb.GetTaskResponse{Response: &pb.Response{Code: 200, Message: "ok"}, Task: taskModelToProto(m)}, nil
}

// UpdateTask
func (s *TaskServiceServer) UpdateTask(ctx context.Context, req *pb.UpdateTaskRequest) (*pb.UpdateTaskResponse, error) {
	if s.core == nil {
		return nil, status.Error(codes.FailedPrecondition, "service not init")
	}
	if req == nil || req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	upd := models.TaskUpdateRequest{ID: req.Id}
	if req.Title != "" {
		upd.Title = &req.Title
	}
	if req.Description != "" {
		upd.Description = &req.Description
	}
	if req.Assignee != "" {
		upd.Assignee = &req.Assignee
	}
	if req.Status != pb.TaskStatus_TASK_STATUS_UNSPECIFIED {
		m := map[pb.TaskStatus]string{pb.TaskStatus_TASK_STATUS_TODO: "Todo", pb.TaskStatus_TASK_STATUS_IN_PROGRESS: "InProgress", pb.TaskStatus_TASK_STATUS_DONE: "Done"}[req.Status]
		upd.Status = &m
	}
	if req.Priority != pb.TaskPriority_TASK_PRIORITY_UNSPECIFIED {
		p := map[pb.TaskPriority]string{pb.TaskPriority_TASK_PRIORITY_LOW: "Low", pb.TaskPriority_TASK_PRIORITY_MEDIUM: "Medium", pb.TaskPriority_TASK_PRIORITY_HIGH: "High"}[req.Priority]
		upd.Priority = &p
	}
	if req.Deadline != nil {
		d := req.Deadline.AsTime()
		upd.Deadline = &d
	}
	if req.ScheduledDate != nil {
		sd := req.ScheduledDate.AsTime()
		upd.ScheduledDate = &sd
	}
	if len(req.Comments) > 0 {
		var cs []models.Comment
		for _, pc := range req.Comments {
			created := time.Now()
			if pc.CreatedAt != nil {
				created = pc.CreatedAt.AsTime()
			}
			cs = append(cs, models.Comment{Text: pc.Text, CreatedBy: pc.CreatedBy, CreatedAt: created})
		}
		upd.Comments = cs
	}
	m, err := s.core.Update(ctx, uid, upd)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, status.Error(codes.NotFound, "task not found")
		}
		return nil, status.Errorf(codes.Internal, "update err: %v", err)
	}
	return &pb.UpdateTaskResponse{Response: &pb.Response{Code: 200, Message: "ok"}, Task: taskModelToProto(m)}, nil
}

// DeleteTask
func (s *TaskServiceServer) DeleteTask(ctx context.Context, req *pb.DeleteTaskRequest) (*pb.Response, error) {
	if s.core == nil {
		return nil, status.Error(codes.FailedPrecondition, "service not init")
	}
	if req == nil || req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	if err := s.core.Delete(ctx, uid, req.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "delete err: %v", err)
	}
	return &pb.Response{Code: 200, Message: "deleted"}, nil
}

// GetTaskSortConfig & UpdateTaskSortConfig: 暂时仍直接使用排序服务（独立复杂逻辑）
func (s *TaskServiceServer) GetTaskSortConfig(ctx context.Context, req *pb.GetTaskSortConfigRequest) (*pb.GetTaskSortConfigResponse, error) {
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	userObj, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad user id")
	}
	sortSvc := services.NewTaskSortService(s.db)
	cfg, err := sortSvc.GetTaskSortConfig(ctx, userObj)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cfg err: %v", err)
	}
	return &pb.GetTaskSortConfigResponse{Response: &pb.Response{Code: 200, Message: "ok"}, Config: &pb.TaskSortConfig{UserId: uid, PriorityDays: int32(cfg.PriorityDays), MaxDisplayCount: int32(cfg.MaxDisplayCount), WeightUrgent: cfg.WeightUrgent, WeightImportant: cfg.WeightImportant, CreatedAt: timestamppb.New(cfg.CreatedAt), UpdatedAt: timestamppb.New(cfg.UpdatedAt)}}, nil
}

func (s *TaskServiceServer) UpdateTaskSortConfig(ctx context.Context, req *pb.UpdateTaskSortConfigRequest) (*pb.UpdateTaskSortConfigResponse, error) {
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	userObj, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad user id")
	}
	sortSvc := services.NewTaskSortService(s.db)
	updReq := models.UpdateTaskSortConfigRequest{}
	if req.PriorityDays > 0 {
		v := int(req.PriorityDays)
		updReq.PriorityDays = &v
	}
	if req.MaxDisplayCount > 0 {
		v := int(req.MaxDisplayCount)
		updReq.MaxDisplayCount = &v
	}
	if req.WeightUrgent > 0 {
		v := req.WeightUrgent
		updReq.WeightUrgent = &v
	}
	if req.WeightImportant > 0 {
		v := req.WeightImportant
		updReq.WeightImportant = &v
	}
	cfg, err := sortSvc.UpdateTaskSortConfig(ctx, userObj, updReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update cfg err: %v", err)
	}
	return &pb.UpdateTaskSortConfigResponse{Response: &pb.Response{Code: 200, Message: "ok"}, Config: &pb.TaskSortConfig{UserId: uid, PriorityDays: int32(cfg.PriorityDays), MaxDisplayCount: int32(cfg.MaxDisplayCount), WeightUrgent: cfg.WeightUrgent, WeightImportant: cfg.WeightImportant, CreatedAt: timestamppb.New(cfg.CreatedAt), UpdatedAt: timestamppb.New(cfg.UpdatedAt)}}, nil
}
