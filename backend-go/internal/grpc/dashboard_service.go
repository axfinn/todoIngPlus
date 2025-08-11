package grpcserver

import (
	"context"

	"github.com/axfinn/todoIngPlus/backend-go/internal/convert"
	"github.com/axfinn/todoIngPlus/backend-go/internal/services"
	pb "github.com/axfinn/todoIngPlus/backend-go/pkg/api/v1"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DashboardServiceServer struct {
	pb.UnimplementedDashboardServiceServer
	core *services.TaskSortService
}

func NewDashboardServiceServer(db *mongo.Database) *DashboardServiceServer {
	return &DashboardServiceServer{core: services.NewTaskSortService(db)}
}

func (s *DashboardServiceServer) GetDashboardData(ctx context.Context, req *pb.GetDashboardDataRequest) (*pb.GetDashboardDataResponse, error) {
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	userObj, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad user id")
	}
	data, err := s.core.GetDashboardData(ctx, userObj)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "dashboard err: %v", err)
	}
	// assemble proto
	upcomingEvents := make([]*pb.Event, 0, len(data.UpcomingEvents))
	for i := range data.UpcomingEvents {
		upcomingEvents = append(upcomingEvents, convert.EventToProto(&data.UpcomingEvents[i]))
	}
	priorityTasks := make([]*pb.PriorityTask, 0, len(data.PriorityTasks))
	for i := range data.PriorityTasks {
		pt := data.PriorityTasks[i]
		priorityTasks = append(priorityTasks, &pb.PriorityTask{Task: convert.TaskToProto(&pt.Task), PriorityScore: pt.PriorityScore, DaysLeft: int32(pt.DaysLeft), IsUrgent: pt.IsUrgent})
	}
	pendingReminders := make([]*pb.UpcomingReminder, 0, len(data.PendingReminders))
	for i := range data.PendingReminders {
		r := data.PendingReminders[i]
		pendingReminders = append(pendingReminders, &pb.UpcomingReminder{Id: r.ID.Hex(), EventTitle: r.EventTitle, EventDate: timestamppb.New(r.EventDate), Message: r.Message, DaysLeft: int32(r.DaysLeft), Importance: int32(r.Importance), ReminderAt: timestamppb.New(r.ReminderAt)})
	}
	eSummary := &pb.EventSummary{TotalEvents: data.EventSummary.TotalEvents, UpcomingCount: data.EventSummary.UpcomingCount, TodayCount: data.EventSummary.TodayCount, TypeBreakdown: data.EventSummary.TypeBreakdown}
	tSummary := &pb.TaskSummary{TotalTasks: data.TaskSummary.TotalTasks, CompletedRate: data.TaskSummary.CompletedRate, OverdueCount: data.TaskSummary.OverdueCount, StatusCounts: data.TaskSummary.StatusCounts}
	return &pb.GetDashboardDataResponse{Response: &pb.Response{Code: 200, Message: "ok"}, Data: &pb.DashboardData{UpcomingEvents: upcomingEvents, PriorityTasks: priorityTasks, PendingReminders: pendingReminders, EventSummary: eSummary, TaskSummary: tSummary}}, nil
}

func (s *DashboardServiceServer) GetPriorityTasks(ctx context.Context, req *pb.GetPriorityTasksRequest) (*pb.GetPriorityTasksResponse, error) {
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	userObj, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad user id")
	}
	list, err := s.core.GetPriorityTasks(ctx, userObj)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "priority tasks err: %v", err)
	}
	items := make([]*pb.PriorityTask, 0, len(list))
	for i := range list {
		pt := list[i]
		items = append(items, &pb.PriorityTask{Task: convert.TaskToProto(&pt.Task), PriorityScore: pt.PriorityScore, DaysLeft: int32(pt.DaysLeft), IsUrgent: pt.IsUrgent})
	}
	return &pb.GetPriorityTasksResponse{Response: &pb.Response{Code: 200, Message: "ok"}, Tasks: items, Total: int32(len(items))}, nil
}
