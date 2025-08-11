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
)

type NotificationServiceServer struct {
	pb.UnimplementedNotificationServiceServer
	core *services.NotificationService
}

func NewNotificationServiceServer(db *mongo.Database) *NotificationServiceServer {
	return &NotificationServiceServer{core: services.NewNotificationService(db)}
}

func (s *NotificationServiceServer) CreateNotification(ctx context.Context, req *pb.CreateNotificationRequest) (*pb.CreateNotificationResponse, error) {
	if req == nil || req.Type == "" || req.Message == "" {
		return nil, status.Error(codes.InvalidArgument, "type/message required")
	}
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	userObj, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad user id")
	}
	create := convert.ProtoToNotificationCreate(req, userObj)
	n, err := s.core.Create(ctx, create)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create notification err: %v", err)
	}
	return &pb.CreateNotificationResponse{Response: &pb.Response{Code: 201, Message: "created"}, Notification: convert.NotificationToProto(&n)}, nil
}

func (s *NotificationServiceServer) ListNotifications(ctx context.Context, req *pb.ListNotificationsRequest) (*pb.ListNotificationsResponse, error) {
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
	list, err := s.core.List(ctx, userObj, req.GetUnreadOnly(), limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list notifications err: %v", err)
	}
	out := make([]*pb.Notification, 0, len(list))
	for i := range list {
		out = append(out, convert.NotificationToProto(&list[i]))
	}
	return &pb.ListNotificationsResponse{Response: &pb.Response{Code: 200, Message: "ok"}, Notifications: out, Count: int32(len(out))}, nil
}

func (s *NotificationServiceServer) MarkNotificationRead(ctx context.Context, req *pb.MarkNotificationReadRequest) (*pb.MarkNotificationReadResponse, error) {
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
		return nil, status.Error(codes.InvalidArgument, "bad id")
	}
	if err := s.core.MarkRead(ctx, userObj, id); err != nil {
		if err.Error() == "not_found" {
			return nil, status.Error(codes.NotFound, "not found")
		}
		return nil, status.Errorf(codes.Internal, "mark read err: %v", err)
	}
	return &pb.MarkNotificationReadResponse{Response: &pb.Response{Code: 200, Message: "ok"}, Success: true}, nil
}

func (s *NotificationServiceServer) MarkAllNotificationsRead(ctx context.Context, req *pb.MarkAllNotificationsReadRequest) (*pb.MarkAllNotificationsReadResponse, error) {
	uid, _ := UserIDFromContext(ctx)
	if uid == "" {
		return nil, status.Error(codes.Unauthenticated, "user id missing")
	}
	userObj, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad user id")
	}
	mod, err := s.core.MarkAllRead(ctx, userObj)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "mark all read err: %v", err)
	}
	return &pb.MarkAllNotificationsReadResponse{Response: &pb.Response{Code: 200, Message: "ok"}, Modified: mod}, nil
}
