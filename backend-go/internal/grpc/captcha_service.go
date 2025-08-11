package grpcserver

import (
	"context"

	pb "github.com/axfinn/todoIngPlus/backend-go/pkg/api/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CaptchaServiceServer 简单占位实现（若后端已有 HTTP 生成逻辑，可在此接入）
type CaptchaServiceServer struct {
	pb.UnimplementedCaptchaServiceServer
}

func NewCaptchaServiceServer() *CaptchaServiceServer { return &CaptchaServiceServer{} }

func (s *CaptchaServiceServer) GetCaptcha(ctx context.Context, req *pb.GetCaptchaRequest) (*pb.GetCaptchaResponse, error) {
	// TODO: 接入真实验证码生成逻辑；当前返回 Unimplemented 方便前端检测
	return nil, status.Error(codes.Unimplemented, "captcha generation not implemented yet")
}
