package grpcserver

import (
	"context"
	"time"

	"github.com/axfinn/todoIngPlus/backend-go/internal/convert"
	"github.com/axfinn/todoIngPlus/backend-go/internal/models"
	"github.com/axfinn/todoIngPlus/backend-go/internal/repository"
	"github.com/axfinn/todoIngPlus/backend-go/internal/services"
	pb "github.com/axfinn/todoIngPlus/backend-go/pkg/api/v1"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ReminderServiceServer struct {
	pb.UnimplementedReminderServiceServer
	core *services.ReminderService
}

func NewReminderServiceServer(db *mongo.Database) *ReminderServiceServer { // 保留签名
	repo := repository.NewReminderRepository(db)
	return &ReminderServiceServer{core: services.NewReminderService(repo)}
}

// CreateReminder
func (s *ReminderServiceServer) CreateReminder(ctx context.Context, req *pb.CreateReminderRequest) (*pb.CreateReminderResponse, error) {
	if req == nil || req.EventId == "" {
		return nil, status.Error(codes.InvalidArgument, "event_id required")
	}
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	userObj, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad user id")
	}
	eid, err := primitive.ObjectIDFromHex(req.EventId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad event id")
	}
	abs := make([]time.Time, 0, len(req.AbsoluteTimes))
	for _, t := range req.AbsoluteTimes {
		abs = append(abs, t.AsTime())
	}
	mReq := models.CreateReminderRequest{EventID: eid, AdvanceDays: int(req.AdvanceDays), ReminderTimes: req.ReminderTimes, AbsoluteTimes: abs, ReminderType: convert.ProtoToReminderType(req.ReminderType), CustomMessage: req.CustomMessage}
	r, err := s.core.CreateReminder(ctx, userObj, mReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create reminder err: %v", err)
	}
	return &pb.CreateReminderResponse{Response: &pb.Response{Code: 201, Message: "created"}, Reminder: convert.ReminderToProto(r)}, nil
}

// GetReminder
func (s *ReminderServiceServer) GetReminder(ctx context.Context, req *pb.GetReminderRequest) (*pb.GetReminderResponse, error) {
	if req == nil || req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	userObj, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad user id")
	}
	rid, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad reminder id")
	}
	res, err := s.core.GetReminder(ctx, userObj, rid)
	if err != nil {
		if err.Error() == "reminder not found" {
			return nil, status.Error(codes.NotFound, "not found")
		}
		return nil, status.Errorf(codes.Internal, "get reminder err: %v", err)
	}
	ret := convert.ReminderToProto(&res.Reminder)
	// 将事件也转换
	return &pb.GetReminderResponse{Response: &pb.Response{Code: 200, Message: "ok"}, Reminder: &pb.ReminderWithEvent{Reminder: ret, Event: convert.EventToProto(&res.Event)}}, nil
}

// UpdateReminder
func (s *ReminderServiceServer) UpdateReminder(ctx context.Context, req *pb.UpdateReminderRequest) (*pb.UpdateReminderResponse, error) {
	if req == nil || req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	userObj, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad user id")
	}
	rid, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad reminder id")
	}
	upd := models.UpdateReminderRequest{}
	if req.AdvanceDays != 0 {
		v := int(req.AdvanceDays)
		upd.AdvanceDays = &v
	}
	if len(req.ReminderTimes) > 0 {
		upd.ReminderTimes = req.ReminderTimes
	}
	if len(req.AbsoluteTimes) > 0 {
		at := make([]time.Time, 0, len(req.AbsoluteTimes))
		for _, t := range req.AbsoluteTimes {
			at = append(at, t.AsTime())
		}
		upd.AbsoluteTimes = &at
	}
	if req.ReminderType != pb.ReminderType_REMINDER_TYPE_UNSPECIFIED {
		rt := convert.ProtoToReminderType(req.ReminderType)
		upd.ReminderType = &rt
	}
	if req.CustomMessage != "" {
		cm := req.CustomMessage
		upd.CustomMessage = &cm
	}
	if req.IsActive {
		b := req.IsActive
		upd.IsActive = &b
	}
	r, err := s.core.UpdateReminder(ctx, userObj, rid, upd)
	if err != nil {
		if err.Error() == "reminder not found" {
			return nil, status.Error(codes.NotFound, "not found")
		}
		return nil, status.Errorf(codes.Internal, "update reminder err: %v", err)
	}
	return &pb.UpdateReminderResponse{Response: &pb.Response{Code: 200, Message: "ok"}, Reminder: convert.ReminderToProto(r)}, nil
}

// DeleteReminder
func (s *ReminderServiceServer) DeleteReminder(ctx context.Context, req *pb.DeleteReminderRequest) (*pb.Response, error) {
	if req == nil || req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	userObj, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad user id")
	}
	rid, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad reminder id")
	}
	if err := s.core.DeleteReminder(ctx, userObj, rid); err != nil {
		if err.Error() == "reminder not found" {
			return nil, status.Error(codes.NotFound, "not found")
		}
		return nil, status.Errorf(codes.Internal, "delete reminder err: %v", err)
	}
	return &pb.Response{Code: 200, Message: "deleted"}, nil
}

// ListReminders
func (s *ReminderServiceServer) ListReminders(ctx context.Context, req *pb.ListRemindersRequest) (*pb.ListRemindersResponse, error) {
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	userObj, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad user id")
	}
	page, limit := 1, 20
	if req != nil && req.Pagination != nil {
		if req.Pagination.GetPage() > 0 {
			page = int(req.Pagination.GetPage())
		}
		if req.Pagination.GetLimit() > 0 {
			limit = int(req.Pagination.GetLimit())
		}
	}
	resp, err := s.core.ListReminders(ctx, userObj, page, limit, req.GetActiveOnly())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list reminders err: %v", err)
	}
	out := make([]*pb.ReminderWithEvent, 0, len(resp.Reminders))
	for i := range resp.Reminders {
		r := convert.ReminderToProto(&resp.Reminders[i].Reminder)
		out = append(out, &pb.ReminderWithEvent{Reminder: r, Event: convert.EventToProto(&resp.Reminders[i].Event)})
	}
	pg := &pb.PaginationResponse{Page: int32(resp.Page), Limit: int32(resp.PageSize), Total: int32(resp.Total)}
	return &pb.ListRemindersResponse{Response: &pb.Response{Code: 200, Message: "ok"}, Reminders: out, Pagination: pg}, nil
}

// ListSimpleReminders
func (s *ReminderServiceServer) ListSimpleReminders(ctx context.Context, req *pb.ListSimpleRemindersRequest) (*pb.ListSimpleRemindersResponse, error) {
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	userObj, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad user id")
	}
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 50
	}
	list, err := s.core.ListSimpleReminders(ctx, userObj, req.GetActiveOnly(), limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list simple reminders err: %v", err)
	}
	items := make([]*pb.Reminder, 0, len(list))
	for i := range list { // 组装 Reminder proto (无事件)
		items = append(items, &pb.Reminder{Id: list[i].ID.Hex(), EventId: list[i].EventID.Hex(), UserId: userObj.Hex(), AdvanceDays: int32(list[i].AdvanceDays), ReminderTimes: list[i].ReminderTimes, ReminderType: convert.ReminderTypeToProto(list[i].ReminderType), CustomMessage: list[i].CustomMessage, IsActive: list[i].IsActive})
	}
	return &pb.ListSimpleRemindersResponse{Response: &pb.Response{Code: 200, Message: "ok"}, Reminders: items, Count: int32(len(items))}, nil
}

// GetUpcomingReminders
func (s *ReminderServiceServer) GetUpcomingReminders(ctx context.Context, req *pb.GetUpcomingRemindersRequest) (*pb.GetUpcomingRemindersResponse, error) {
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	userObj, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad user id")
	}
	hours := int(req.GetHours())
	if hours <= 0 {
		hours = 24
	}
	list, err := s.core.GetUpcomingReminders(ctx, userObj, hours)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "upcoming reminders err: %v", err)
	}
	items := make([]*pb.UpcomingReminder, 0, len(list))
	for i := range list {
		items = append(items, &pb.UpcomingReminder{Id: list[i].ID.Hex(), EventTitle: list[i].EventTitle, EventDate: timestamppb.New(list[i].EventDate), Message: list[i].Message, DaysLeft: int32(list[i].DaysLeft), Importance: int32(list[i].Importance), ReminderAt: timestamppb.New(list[i].ReminderAt)})
	}
	return &pb.GetUpcomingRemindersResponse{Response: &pb.Response{Code: 200, Message: "ok"}, Reminders: items, Hours: int32(hours)}, nil
}

// PreviewReminder
func (s *ReminderServiceServer) PreviewReminder(ctx context.Context, req *pb.PreviewReminderRequest) (*pb.PreviewReminderResponse, error) {
	if req == nil || req.EventId == "" {
		return nil, status.Error(codes.InvalidArgument, "event_id required")
	}
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	userObj, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad user id")
	}
	eid, err := primitive.ObjectIDFromHex(req.EventId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad event id")
	}
	res, err := s.core.PreviewReminder(ctx, userObj, eid, int(req.AdvanceDays), req.ReminderTimes)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "preview reminder err: %v", err)
	}
	items := []*pb.PreviewReminderItem{}
	if res.Next != nil {
		items = append(items, &pb.PreviewReminderItem{ReminderAt: timestamppb.New(*res.Next), Message: res.Text})
	}
	return &pb.PreviewReminderResponse{Response: &pb.Response{Code: 200, Message: "ok"}, Schedule: items}, nil
}

// SnoozeReminder
func (s *ReminderServiceServer) SnoozeReminder(ctx context.Context, req *pb.SnoozeReminderRequest) (*pb.SnoozeReminderResponse, error) {
	if req == nil || req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	userObj, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad user id")
	}
	rid, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad reminder id")
	}
	if err := s.core.SnoozeReminder(ctx, userObj, rid, int(req.SnoozeMinutes)); err != nil {
		if err.Error() == "reminder not found" {
			return nil, status.Error(codes.NotFound, "not found")
		}
		return nil, status.Errorf(codes.Internal, "snooze reminder err: %v", err)
	}
	return &pb.SnoozeReminderResponse{Response: &pb.Response{Code: 200, Message: "ok"}, SnoozeMinutes: req.SnoozeMinutes}, nil
}

// ToggleReminderActive
func (s *ReminderServiceServer) ToggleReminderActive(ctx context.Context, req *pb.ToggleReminderActiveRequest) (*pb.ToggleReminderActiveResponse, error) {
	if req == nil || req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	userObj, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad user id")
	}
	rid, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad reminder id")
	}
	val, err := s.core.ToggleReminderActive(ctx, userObj, rid)
	if err != nil {
		if err.Error() == "reminder not found" {
			return nil, status.Error(codes.NotFound, "not found")
		}
		return nil, status.Errorf(codes.Internal, "toggle reminder err: %v", err)
	}
	return &pb.ToggleReminderActiveResponse{Response: &pb.Response{Code: 200, Message: "ok"}, IsActive: val}, nil
}

// CreateTestReminder
func (s *ReminderServiceServer) CreateTestReminder(ctx context.Context, req *pb.CreateTestReminderRequest) (*pb.CreateTestReminderResponse, error) {
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	userObj, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad user id")
	}
	var eid primitive.ObjectID
	if req.EventId != "" {
		eid, err = primitive.ObjectIDFromHex(req.EventId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "bad event id")
		}
	}
	if eid.IsZero() {
		return nil, status.Error(codes.InvalidArgument, "event_id required")
	}
	r, ev, err := s.core.CreateImmediateTestReminder(ctx, userObj, eid, req.Message, int(req.DelaySeconds))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create test reminder err: %v", err)
	}
	return &pb.CreateTestReminderResponse{Response: &pb.Response{Code: 201, Message: "created"}, Reminder: convert.ReminderToProto(r), Event: convert.EventToProto(ev), EmailSent: req.SendEmail}, nil
}
