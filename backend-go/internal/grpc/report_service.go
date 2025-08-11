package grpcserver

import (
	"context"
	"errors"
	"time"

	"github.com/axfinn/todoIngPlus/backend-go/internal/convert"
	"github.com/axfinn/todoIngPlus/backend-go/internal/models"
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
	db *mongo.Database
}

func NewReportServiceServer(db *mongo.Database) *ReportServiceServer {
	return &ReportServiceServer{db: db}
}

func (s *ReportServiceServer) coll() *mongo.Collection { return s.db.Collection("reports") }

// GenerateReport 生成报表（简化: 直接存储基本结构）
func (s *ReportServiceServer) GenerateReport(ctx context.Context, req *pb.GenerateReportRequest) (*pb.GenerateReportResponse, error) {
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	if req == nil || req.Title == "" {
		return nil, status.Error(codes.InvalidArgument, "title required")
	}
	now := time.Now()
	rep := models.Report{ID: primitive.NewObjectID().Hex(), UserID: uid, Type: convert.ProtoToReportType(req.Type), Period: "custom", Title: req.Title, Content: "", Tasks: []string{}, CreatedAt: now, UpdatedAt: now, Statistics: models.Statistics{}}
	if _, err := s.coll().InsertOne(ctx, rep); err != nil {
		return nil, status.Errorf(codes.Internal, "insert report err: %v", err)
	}
	return &pb.GenerateReportResponse{Response: &pb.Response{Code: 201, Message: "created"}, Report: convert.ReportToProto(&rep)}, nil
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
	filter := bson.M{"userId": uid}
	cur, err := s.coll().Find(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "find reports err: %v", err)
	}
	defer cur.Close(ctx)
	var list []models.Report
	for cur.Next(ctx) {
		var r models.Report
		if cur.Decode(&r) == nil {
			list = append(list, r)
		}
	}
	start := (page - 1) * limit
	if start > len(list) {
		start = len(list)
	}
	end := start + limit
	if end > len(list) {
		end = len(list)
	}
	slice := list[start:end]
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
	filter := bson.M{"_id": req.Id, "userId": uid}
	var r models.Report
	if err := s.coll().FindOne(ctx, filter).Decode(&r); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, status.Error(codes.NotFound, "not found")
		}
		return nil, status.Errorf(codes.Internal, "get report err: %v", err)
	}
	return &pb.GetReportResponse{Response: &pb.Response{Code: 200, Message: "ok"}, Report: convert.ReportToProto(&r)}, nil
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
	res, err := s.coll().DeleteOne(ctx, bson.M{"_id": req.Id, "userId": uid})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete report err: %v", err)
	}
	if res.DeletedCount == 0 {
		return nil, status.Error(codes.NotFound, "not found")
	}
	return &pb.Response{Code: 200, Message: "deleted"}, nil
}

// ExportReport 简化: 返回占位 bytes
func (s *ReportServiceServer) ExportReport(ctx context.Context, req *pb.ExportReportRequest) (*pb.ExportReportResponse, error) {
	if req == nil || req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	var r models.Report
	if err := s.coll().FindOne(ctx, bson.M{"_id": req.Id, "userId": uid}).Decode(&r); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, status.Error(codes.NotFound, "not found")
		}
		return nil, status.Errorf(codes.Internal, "export get report err: %v", err)
	}
	content := []byte("Report Export Placeholder: " + r.Title)
	return &pb.ExportReportResponse{Response: &pb.Response{Code: 200, Message: "ok"}, Data: content, Filename: r.Title + ".txt", ContentType: "text/plain"}, nil
}

// helper for start/end times if needed (unused now)
func toTS(t time.Time) *timestamppb.Timestamp { return timestamppb.New(t) }
