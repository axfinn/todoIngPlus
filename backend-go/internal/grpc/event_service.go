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
)

// EventServiceServer gRPC 包装
type EventServiceServer struct {
	pb.UnimplementedEventServiceServer
	core *services.EventService
}

func NewEventServiceServer(db *mongo.Database) *EventServiceServer { // 保留签名兼容现有调用
	repo := repository.NewEventRepository(db)
	return &EventServiceServer{core: services.NewEventService(repo)}
}

// CreateEvent
func (s *EventServiceServer) CreateEvent(ctx context.Context, req *pb.CreateEventRequest) (*pb.CreateEventResponse, error) {
	if s.core == nil {
		return nil, status.Error(codes.FailedPrecondition, "service not init")
	}
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	userObj, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad user id")
	}
	if req == nil || req.Title == "" || req.EventDate == nil {
		return nil, status.Error(codes.InvalidArgument, "title/date required")
	}
	var recCfg map[string]interface{}
	if len(req.RecurrenceConfig) > 0 {
		recCfg = make(map[string]interface{}, len(req.RecurrenceConfig))
		for k, v := range req.RecurrenceConfig {
			recCfg[k] = v
		}
	}
	mReq := models.CreateEventRequest{Title: req.Title, Description: req.Description, EventType: convert.ProtoToEventType(req.EventType), EventDate: req.EventDate.AsTime(), RecurrenceType: convert.ProtoToRecurrenceType(req.RecurrenceType), RecurrenceConfig: recCfg, ImportanceLevel: int(req.ImportanceLevel), Tags: req.Tags, Location: req.Location, IsAllDay: req.IsAllDay}
	if mReq.EventType == "" {
		mReq.EventType = "custom"
	}
	if mReq.RecurrenceType == "" {
		mReq.RecurrenceType = "none"
	}
	if mReq.ImportanceLevel == 0 {
		mReq.ImportanceLevel = 3
	}
	ev, err := s.core.CreateEvent(ctx, userObj, mReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create event err: %v", err)
	}
	return &pb.CreateEventResponse{Response: &pb.Response{Code: 201, Message: "created"}, Event: convert.EventToProto(ev)}, nil
}

// GetEvent
func (s *EventServiceServer) GetEvent(ctx context.Context, req *pb.GetEventRequest) (*pb.GetEventResponse, error) {
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
	userObj, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad user id")
	}
	id, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad event id")
	}
	ev, err := s.core.GetEvent(ctx, userObj, id)
	if err != nil {
		if err.Error() == "event not found" {
			return nil, status.Error(codes.NotFound, "event not found")
		}
		return nil, status.Errorf(codes.Internal, "get event err: %v", err)
	}
	return &pb.GetEventResponse{Response: &pb.Response{Code: 200, Message: "ok"}, Event: convert.EventToProto(ev)}, nil
}

// UpdateEvent
func (s *EventServiceServer) UpdateEvent(ctx context.Context, req *pb.UpdateEventRequest) (*pb.UpdateEventResponse, error) {
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
	userObj, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad user id")
	}
	id, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad event id")
	}
	upd := models.UpdateEventRequest{}
	if req.Title != "" {
		upd.Title = &req.Title
	}
	if req.Description != "" {
		upd.Description = &req.Description
	}
	if req.EventType != pb.EventType_EVENT_TYPE_UNSPECIFIED {
		et := convert.ProtoToEventType(req.EventType)
		upd.EventType = &et
	}
	if req.EventDate != nil {
		d := req.EventDate.AsTime()
		upd.EventDate = &d
	}
	if req.RecurrenceType != pb.RecurrenceType_RECURRENCE_TYPE_UNSPECIFIED {
		rt := convert.ProtoToRecurrenceType(req.RecurrenceType)
		upd.RecurrenceType = &rt
	}
	if len(req.RecurrenceConfig) > 0 {
		m := make(map[string]interface{}, len(req.RecurrenceConfig))
		for k, v := range req.RecurrenceConfig {
			m[k] = v
		}
		upd.RecurrenceConfig = m
	}
	if req.ImportanceLevel != 0 {
		il := int(req.ImportanceLevel)
		upd.ImportanceLevel = &il
	}
	if len(req.Tags) > 0 {
		upd.Tags = req.Tags
	}
	if req.Location != "" {
		l := req.Location
		upd.Location = &l
	}
	if req.IsAllDay {
		b := req.IsAllDay
		upd.IsAllDay = &b
	}
	if req.IsActive {
		b := req.IsActive
		upd.IsActive = &b
	}
	ev, err := s.core.UpdateEvent(ctx, userObj, id, upd)
	if err != nil {
		if err.Error() == "event not found" {
			return nil, status.Error(codes.NotFound, "event not found")
		}
		return nil, status.Errorf(codes.Internal, "update event err: %v", err)
	}
	return &pb.UpdateEventResponse{Response: &pb.Response{Code: 200, Message: "ok"}, Event: convert.EventToProto(ev)}, nil
}

// DeleteEvent
func (s *EventServiceServer) DeleteEvent(ctx context.Context, req *pb.DeleteEventRequest) (*pb.Response, error) {
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
	userObj, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad user id")
	}
	id, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad event id")
	}
	if err := s.core.DeleteEvent(ctx, userObj, id); err != nil {
		if err.Error() == "event not found" {
			return nil, status.Error(codes.NotFound, "event not found")
		}
		return nil, status.Errorf(codes.Internal, "delete event err: %v", err)
	}
	return &pb.Response{Code: 200, Message: "deleted"}, nil
}

// ListEvents
func (s *EventServiceServer) ListEvents(ctx context.Context, req *pb.ListEventsRequest) (*pb.ListEventsResponse, error) {
	if s.core == nil {
		return nil, status.Error(codes.FailedPrecondition, "service not init")
	}
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	userObj, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad user id")
	}
	page, limit := 1, 20
	if req.Pagination != nil {
		if req.Pagination.GetPage() > 0 {
			page = int(req.Pagination.GetPage())
		}
		if req.Pagination.GetLimit() > 0 {
			limit = int(req.Pagination.GetLimit())
		}
	}
	var et string
	if req.EventType != pb.EventType_EVENT_TYPE_UNSPECIFIED {
		et = convert.ProtoToEventType(req.EventType)
	}
	var start, end *time.Time
	if req.UpcomingOnly {
		now := time.Now()
		s := now
		e := now.AddDate(0, 0, 30)
		start, end = &s, &e
	}
	resp, err := s.core.ListEvents(ctx, userObj, page, limit, et, start, end)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list events err: %v", err)
	}
	var items []*pb.Event
	for i := range resp.Events {
		items = append(items, convert.EventToProto(&resp.Events[i]))
	}
	pg := &pb.PaginationResponse{Limit: int32(limit), Total: int32(resp.Total), Page: int32(resp.Page)}
	return &pb.ListEventsResponse{Response: &pb.Response{Code: 200, Message: "ok"}, Events: items, Pagination: pg}, nil
}

// GetUpcomingEvents
func (s *EventServiceServer) GetUpcomingEvents(ctx context.Context, req *pb.GetUpcomingEventsRequest) (*pb.GetUpcomingEventsResponse, error) {
	if s.core == nil {
		return nil, status.Error(codes.FailedPrecondition, "service not init")
	}
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	userObj, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad user id")
	}
	days := int(req.GetDays())
	if days <= 0 {
		days = 7
	}
	list, err := s.core.GetUpcomingEvents(ctx, userObj, days)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "upcoming events err: %v", err)
	}
	var out []*pb.Event
	for i := range list {
		out = append(out, convert.EventToProto(&list[i]))
	}
	return &pb.GetUpcomingEventsResponse{Response: &pb.Response{Code: 200, Message: "ok"}, Events: out, Days: int32(days)}, nil
}

// GetCalendarEvents
func (s *EventServiceServer) GetCalendarEvents(ctx context.Context, req *pb.GetCalendarEventsRequest) (*pb.GetCalendarEventsResponse, error) {
	if s.core == nil {
		return nil, status.Error(codes.FailedPrecondition, "service not init")
	}
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	userObj, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad user id")
	}
	year, month := int(req.GetYear()), int(req.GetMonth())
	if year == 0 || month == 0 {
		now := time.Now()
		year = now.Year()
		month = int(now.Month())
	}
	cal, err := s.core.GetCalendarEvents(ctx, userObj, year, month)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "calendar events err: %v", err)
	}
	var days []*pb.CalendarDayEvents
	for date, evs := range cal {
		var arr []*pb.Event
		for i := range evs {
			arr = append(arr, convert.EventToProto(&evs[i]))
		}
		days = append(days, &pb.CalendarDayEvents{Date: date, Events: arr})
	}
	return &pb.GetCalendarEventsResponse{Response: &pb.Response{Code: 200, Message: "ok"}, Year: int32(year), Month: int32(month), Days: days}, nil
}

// 评论/时间线暂未迁移
func (s *EventServiceServer) AddEventComment(ctx context.Context, req *pb.AddEventCommentRequest) (*pb.AddEventCommentResponse, error) {
	return nil, status.Error(codes.Unimplemented, "comment feature not migrated")
}
func (s *EventServiceServer) UpdateEventComment(ctx context.Context, req *pb.UpdateEventCommentRequest) (*pb.UpdateEventCommentResponse, error) {
	return nil, status.Error(codes.Unimplemented, "comment feature not migrated")
}
func (s *EventServiceServer) DeleteEventComment(ctx context.Context, req *pb.DeleteEventCommentRequest) (*pb.Response, error) {
	return nil, status.Error(codes.Unimplemented, "comment feature not migrated")
}
func (s *EventServiceServer) ListEventTimeline(ctx context.Context, req *pb.ListEventTimelineRequest) (*pb.ListEventTimelineResponse, error) {
	return nil, status.Error(codes.Unimplemented, "comment feature not migrated")
}
