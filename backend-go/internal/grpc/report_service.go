package grpcserver

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/axfinn/todoIngPlus/backend-go/internal/convert"
	"github.com/axfinn/todoIngPlus/backend-go/internal/models"
	"github.com/axfinn/todoIngPlus/backend-go/internal/services"
	pb "github.com/axfinn/todoIngPlus/backend-go/pkg/api/v1"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ReportServiceServer 轻量实现（基础 CRUD & 导出）
type ReportServiceServer struct {
	pb.UnimplementedReportServiceServer
	db   *mongo.Database
	core *services.ReportService
}

func NewReportServiceServer(db *mongo.Database) *ReportServiceServer {
	return &ReportServiceServer{db: db, core: services.NewReportService(db)}
}

func (s *ReportServiceServer) coll() *mongo.Collection { return s.db.Collection("reports") }

// GenerateReport 生成报表（聚合任务统计）
func (s *ReportServiceServer) GenerateReport(ctx context.Context, req *pb.GenerateReportRequest) (*pb.GenerateReportResponse, error) {
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	if req == nil || req.Title == "" {
		return nil, status.Error(codes.InvalidArgument, "title required")
	}
	if s.core == nil {
		return nil, status.Error(codes.FailedPrecondition, "report core not init")
	}
	var start, end time.Time
	if req.StartDate != nil {
		start = req.StartDate.AsTime()
	}
	if req.EndDate != nil {
		end = req.EndDate.AsTime()
	}
	repModel := models.Report{ID: primitive.NewObjectID().Hex(), Type: convert.ProtoToReportType(req.Type), Title: req.Title, Content: ""}
	res, err := s.core.Generate(ctx, uid, repModel, start, end)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate report err: %v", err)
	}
	return &pb.GenerateReportResponse{Response: &pb.Response{Code: 201, Message: "created"}, Report: convert.ReportToProto(res)}, nil
}

// GetReports 列表（简单分页）
func (s *ReportServiceServer) GetReports(ctx context.Context, req *pb.GetReportsRequest) (*pb.GetReportsResponse, error) {
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
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
	// 使用 repository
	list, err := s.core.List(ctx, uid)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list reports err: %v", err)
	}
	startIdx := (page - 1) * limit
	if startIdx > len(list) {
		startIdx = len(list)
	}
	endIdx := startIdx + limit
	if endIdx > len(list) {
		endIdx = len(list)
	}
	slice := list[startIdx:endIdx]
	out := make([]*pb.Report, 0, len(slice))
	for i := range slice {
		out = append(out, convert.ReportToProto(&slice[i]))
	}
	pg := pb.PaginationResponse{Page: int32(page), Limit: int32(limit), Total: int32(len(list)), TotalPages: int32((len(list) + limit - 1) / limit)}
	return &pb.GetReportsResponse{Response: &pb.Response{Code: 200, Message: "ok"}, Reports: out, Pagination: &pg}, nil
}

// GetReport 单个
func (s *ReportServiceServer) GetReport(ctx context.Context, req *pb.GetReportRequest) (*pb.GetReportResponse, error) {
	if req == nil || req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	r, err := s.core.Get(ctx, uid, req.Id)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, status.Error(codes.NotFound, "not found")
		}
		return nil, status.Errorf(codes.Internal, "get report err: %v", err)
	}
	return &pb.GetReportResponse{Response: &pb.Response{Code: 200, Message: "ok"}, Report: convert.ReportToProto(r)}, nil
}

// DeleteReport
func (s *ReportServiceServer) DeleteReport(ctx context.Context, req *pb.DeleteReportRequest) (*pb.Response, error) {
	if req == nil || req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	ok, err := s.core.Delete(ctx, uid, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete report err: %v", err)
	}
	if !ok {
		return nil, status.Error(codes.NotFound, "not found")
	}
	return &pb.Response{Code: 200, Message: "deleted"}, nil
}

// ExportReport 导出: 支持 format=markdown|csv (默认 markdown)
func (s *ReportServiceServer) ExportReport(ctx context.Context, req *pb.ExportReportRequest) (*pb.ExportReportResponse, error) {
	if req == nil || req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	r, err := s.core.Get(ctx, uid, req.Id)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, status.Error(codes.NotFound, "not found")
		}
		return nil, status.Errorf(codes.Internal, "export get report err: %v", err)
	}
	// 时间范围
	end := r.CreatedAt
	var start time.Time
	switch r.Type {
	case "weekly":
		start = end.AddDate(0, 0, -7)
	case "monthly":
		start = end.AddDate(0, -1, 0)
	default:
		start = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, end.Location())
	}
	// Events
	eventColl := s.db.Collection("events")
	eventCur, _ := eventColl.Find(ctx, bson.M{"user_id": uid, "event_date": bson.M{"$gte": start, "$lte": end}})
	var events []models.Event
	if eventCur != nil {
		defer eventCur.Close(ctx)
		for eventCur.Next(ctx) {
			var ev models.Event
			if eventCur.Decode(&ev) == nil {
				events = append(events, ev)
			}
		}
	}
	// Reminders
	remColl := s.db.Collection("reminders")
	remCur, _ := remColl.Find(ctx, bson.M{"user_id": uid})
	var reminders []models.Reminder
	if remCur != nil {
		defer remCur.Close(ctx)
		idx := map[primitive.ObjectID]models.Event{}
		for _, ev := range events {
			idx[ev.ID] = ev
		}
		for remCur.Next(ctx) {
			var rm models.Reminder
			if remCur.Decode(&rm) == nil {
				if ev, ok := idx[rm.EventID]; ok {
					if ev.EventDate.After(start) && ev.EventDate.Before(end.Add(time.Nanosecond)) {
						reminders = append(reminders, rm)
					}
				}
			}
		}
	}
	format := strings.ToLower(req.Format)
	if format == "" {
		format = "markdown"
	}
	var data []byte
	var filename, ctype string
	switch format {
	case "csv":
		var b strings.Builder
		b.WriteString("metric,value\n")
		b.WriteString(fmt.Sprintf("total_tasks,%d\n", r.Statistics.TotalTasks))
		b.WriteString(fmt.Sprintf("completed_tasks,%d\n", r.Statistics.CompletedTasks))
		b.WriteString(fmt.Sprintf("in_progress_tasks,%d\n", r.Statistics.InProgressTasks))
		b.WriteString(fmt.Sprintf("overdue_tasks,%d\n", r.Statistics.OverdueTasks))
		b.WriteString(fmt.Sprintf("completion_rate,%d%%\n", r.Statistics.CompletionRate))
		b.WriteString(fmt.Sprintf("total_events,%d\n", len(events)))
		b.WriteString(fmt.Sprintf("total_reminders,%d\n", len(reminders)))
		b.WriteString("\n# tasks\n")
		b.WriteString("task_id\n")
		for _, id := range r.Tasks {
			b.WriteString(id + "\n")
		}
		b.WriteString("\n# events\n")
		b.WriteString("event_id,title,event_type,event_date,is_all_day,importance\n")
		for _, ev := range events {
			b.WriteString(fmt.Sprintf("%s,%s,%s,%s,%t,%d\n", ev.ID.Hex(), csvEscape(ev.Title), ev.EventType, ev.EventDate.Format(time.RFC3339), ev.IsAllDay, ev.ImportanceLevel))
		}
		b.WriteString("\n# reminders\n")
		b.WriteString("reminder_id,event_id,reminder_type,is_active,next_send,advance_days,times\n")
		for _, rm := range reminders {
			var nextSend string
			if rm.NextSend != nil {
				nextSend = rm.NextSend.Format(time.RFC3339)
			}
			b.WriteString(fmt.Sprintf("%s,%s,%s,%t,%s,%d,\"%s\"\n", rm.ID.Hex(), rm.EventID.Hex(), rm.ReminderType, rm.IsActive, nextSend, rm.AdvanceDays, strings.Join(rm.ReminderTimes, ";")))
		}
		data = []byte(b.String())
		filename = r.Title + ".csv"
		ctype = "text/csv"
	default: // markdown
		var b strings.Builder
		b.WriteString("# Report: " + r.Title + "\n\n")
		b.WriteString(fmt.Sprintf("Period: %s  Range: %s ~ %s  GeneratedAt: %s\n\n", r.Type, start.Format("2006-01-02"), end.Format("2006-01-02"), time.Now().Format(time.RFC3339)))
		b.WriteString("## Task Statistics\n")
		b.WriteString(fmt.Sprintf("- Total: %d\n- Completed: %d\n- In Progress: %d\n- Overdue: %d\n- Completion Rate: %d%%\n\n", r.Statistics.TotalTasks, r.Statistics.CompletedTasks, r.Statistics.InProgressTasks, r.Statistics.OverdueTasks, r.Statistics.CompletionRate))
		b.WriteString("## Events (" + fmt.Sprintf("%d", len(events)) + ")\n")
		for _, ev := range events {
			b.WriteString(fmt.Sprintf("- %s | %s | %s | Importance:%d\n", ev.EventDate.Format(time.RFC3339), ev.Title, ev.EventType, ev.ImportanceLevel))
		}
		if len(events) == 0 {
			b.WriteString("(none)\n")
		}
		b.WriteString("\n## Reminders (" + fmt.Sprintf("%d", len(reminders)) + ")\n")
		for _, rm := range reminders {
			var nextSend string
			if rm.NextSend != nil {
				nextSend = rm.NextSend.Format(time.RFC3339)
			}
			b.WriteString(fmt.Sprintf("- Reminder %s -> Event %s | Type:%s | Active:%t | Next:%s | Times:%s\n", rm.ID.Hex(), rm.EventID.Hex(), rm.ReminderType, rm.IsActive, nextSend, strings.Join(rm.ReminderTimes, ",")))
		}
		if len(reminders) == 0 {
			b.WriteString("(none)\n")
		}
		if len(r.Tasks) > 0 {
			b.WriteString("\n## Tasks\n")
			for _, id := range r.Tasks {
				b.WriteString("- Task ID: " + id + "\n")
			}
		}
		if r.Content != "" {
			b.WriteString("\n## Raw Content\n" + r.Content + "\n")
		}
		if r.PolishedContent != nil {
			b.WriteString("\n## Polished Content\n" + *r.PolishedContent + "\n")
		}
		data = []byte(b.String())
		filename = r.Title + ".md"
		ctype = "text/markdown"
	}
	return &pb.ExportReportResponse{Response: &pb.Response{Code: 200, Message: "ok"}, Data: data, Filename: filename, ContentType: ctype}, nil
}

// csvEscape 简单转义逗号/引号
func csvEscape(s string) string {
	if strings.ContainsAny(s, ",\"\n") {
		return strings.ReplaceAll(strings.ReplaceAll(s, "\"", "''"), ",", " ")
	}
	return s
}

// helper for start/end times if needed (unused now)
func toTS(t time.Time) *timestamppb.Timestamp { return timestamppb.New(t) }
