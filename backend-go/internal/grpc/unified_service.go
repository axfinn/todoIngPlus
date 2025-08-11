package grpcserver

import (
	"context"
	"fmt"
	"time"

	"github.com/axfinn/todoIngPlus/backend-go/internal/services"
	pb "github.com/axfinn/todoIngPlus/backend-go/pkg/api/v1"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type UnifiedServiceServer struct {
	pb.UnimplementedUnifiedServiceServer
	core *services.UnifiedService
}

func NewUnifiedServiceServer(db *mongo.Database) *UnifiedServiceServer {
	return &UnifiedServiceServer{core: services.NewUnifiedService(db)}
}

// GetUnifiedUpcoming 聚合未来窗口数据
func (s *UnifiedServiceServer) GetUnifiedUpcoming(ctx context.Context, req *pb.GetUnifiedUpcomingRequest) (*pb.GetUnifiedUpcomingResponse, error) {
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	userObj, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad user id")
	}
	if req == nil {
		req = &pb.GetUnifiedUpcomingRequest{}
	}
	hours := int(req.GetHours())
	if hours <= 0 {
		hours = 72
	}
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 50
	}
	if req.Debug {
		ctx = context.WithValue(ctx, "unifiedDebug", true)
	}
	items, dbg, err := s.core.GetUpcoming(ctx, userObj, hours, req.GetSources(), limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get upcoming err: %v", err)
	}
	respItems := make([]*pb.UnifiedItem, 0, len(items))
	stats := &pb.UnifiedUpcomingStats{}
	for i := range items {
		it := &items[i]
		respItems = append(respItems, &pb.UnifiedItem{Id: it.ID, Source: it.Source, Title: it.Title, ScheduledAt: timestamppb.New(it.ScheduledAt), CountdownSeconds: int32(it.CountdownSeconds), DaysLeft: int32(it.DaysLeft), Importance: int32(it.Importance), PriorityScore: it.PriorityScore, RelatedEventId: it.RelatedEventID, DetailUrl: it.DetailURL, SourceId: it.SourceID, IsUnscheduled: it.IsUnscheduled})
		switch it.Source {
		case "task":
			stats.Tasks++
		case "event":
			stats.Events++
		case "reminder":
			stats.Reminders++
		}
	}
	var pDbg *pb.UnifiedUpcomingDebug
	if dbg != nil {
		pDbg = &pb.UnifiedUpcomingDebug{GraceDays: int32(dbg.GraceDays), Hours: int32(dbg.Hours), Now: dbg.Now, End: dbg.End, AddedUnscheduled: int32(dbg.AddedUnscheduled)}
		pDbg.WindowCounts = &pb.UnifiedUpcomingWindowCounts{Deadline: int32(dbg.WindowCounts.Deadline), Scheduled: int32(dbg.WindowCounts.Scheduled), DueDateOnly: int32(dbg.WindowCounts.DueDateOnly), Total: int32(dbg.WindowCounts.Total)}
		pDbg.WindowFilter = make(map[string]string, len(dbg.WindowFilter))
		for k, v := range dbg.WindowFilter {
			pDbg.WindowFilter[k] = fmt.Sprint(v)
		}
		pDbg.UnscheduledFilter = make(map[string]string, len(dbg.UnscheduledFilter))
		for k, v := range dbg.UnscheduledFilter {
			pDbg.UnscheduledFilter[k] = fmt.Sprint(v)
		}
	}
	return &pb.GetUnifiedUpcomingResponse{Response: &pb.Response{Code: 200, Message: "ok"}, Items: respItems, Hours: int32(hours), Total: int32(len(respItems)), ServerTimestamp: time.Now().Unix(), Stats: stats, Debug: pDbg}, nil
}

// GetUnifiedCalendar 聚合指定年月的日历
func (s *UnifiedServiceServer) GetUnifiedCalendar(ctx context.Context, req *pb.GetUnifiedCalendarRequest) (*pb.GetUnifiedCalendarResponse, error) {
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	userObj, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad user id")
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	year := int(req.GetYear())
	month := int(req.GetMonth())
	if year <= 0 || month <= 0 || month > 12 {
		now := time.Now()
		year = now.Year()
		month = int(now.Month())
	}
	res, err := s.core.GetCalendar(ctx, userObj, year, month)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get calendar err: %v", err)
	}
	// 转换 map -> repeated days
	outDays := make([]*pb.UnifiedCalendarDay, 0, len(res.Days))
	for day, list := range res.Days {
		items := make([]*pb.UnifiedCalendarItem, 0, len(list))
		for i := range list {
			it := &list[i]
			items = append(items, &pb.UnifiedCalendarItem{Id: it.ID, Source: it.Source, Title: it.Title, ScheduledAt: timestamppb.New(it.ScheduledAt), Importance: int32(it.Importance), DetailUrl: it.DetailURL})
		}
		outDays = append(outDays, &pb.UnifiedCalendarDay{Date: day, Items: items})
	}
	return &pb.GetUnifiedCalendarResponse{Response: &pb.Response{Code: 200, Message: "ok"}, Year: int32(res.Year), Month: int32(res.Month), Days: outDays, Total: int32(res.Total), ServerTimestamp: time.Now().Unix()}, nil
}
