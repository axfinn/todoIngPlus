package grpcserver

import (
	"context"

	"github.com/axfinn/todoIngPlus/backend-go/internal/email"
	"github.com/axfinn/todoIngPlus/backend-go/internal/services"
	pb "github.com/axfinn/todoIngPlus/backend-go/pkg/api/v1"
	"go.mongodb.org/mongo-driver/mongo"
)

// AuthServiceServer 薄包装，委托给核心 services.AuthService
type AuthServiceServer struct {
	pb.UnimplementedAuthServiceServer
	core *services.AuthService
}

// NewAuthServiceServer 创建包装（内部实例化真正的 AuthService）
func NewAuthServiceServer(db *mongo.Database, emailStore *email.Store) *AuthServiceServer {
	return &AuthServiceServer{core: services.NewAuthService(db, emailStore)}
}

// Register 用户注册
func (s *AuthServiceServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	return s.core.Register(ctx, req)
}

// Login 用户登录（仅密码; 邮箱验证码登录后续实现）
func (s *AuthServiceServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	return s.core.Login(ctx, req)
}

// EmailCodeLogin Deprecated: 仍返回未实现
func (s *AuthServiceServer) EmailCodeLogin(ctx context.Context, req *pb.EmailCodeLoginRequest) (*pb.LoginResponse, error) {
	return s.core.EmailCodeLogin(ctx, req)
}

// SendLoginEmailCode 发送邮箱验证码（占位）
func (s *AuthServiceServer) SendLoginEmailCode(ctx context.Context, req *pb.SendLoginEmailCodeRequest) (*pb.SendLoginEmailCodeResponse, error) {
	return s.core.SendLoginEmailCode(ctx, req)
}

// VerifyToken 解析并验证 token
func (s *AuthServiceServer) VerifyToken(ctx context.Context, req *pb.VerifyTokenRequest) (*pb.VerifyTokenResponse, error) {
	return s.core.VerifyToken(ctx, req)
}
