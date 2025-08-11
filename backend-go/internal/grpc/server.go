package grpcserver

import (
	"context"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/axfinn/todoIngPlus/backend-go/internal/auth"
	obs "github.com/axfinn/todoIngPlus/backend-go/internal/observability"
	pb "github.com/axfinn/todoIngPlus/backend-go/pkg/api/v1"
)

// ServerConfig gRPC 服务器配置
type ServerConfig struct {
	Port string
}

// Server 包装 gRPC Server 与依赖
type Server struct {
	cfg    ServerConfig
	server *grpc.Server
}

// New 创建 Server
func New(cfg ServerConfig, register func(s *grpc.Server)) *Server {
	// 链式拦截器
	unaryInterceptors := []grpc.UnaryServerInterceptor{
		loggingInterceptor,
		authInterceptor,
	}

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(unaryInterceptors...),
	)

	// 注册健康检查服务
	h := health.NewServer()
	healthpb.RegisterHealthServer(server, h)
	h.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	// 用户自定义注册（各业务服务）
	register(server)

	// 反射（调试）
	reflection.Register(server)

	return &Server{cfg: cfg, server: server}
}

// Start 启动 gRPC 服务
func (s *Server) Start(ctx context.Context) error {
	port := s.cfg.Port
	if port == "" {
		port = "9001"
	}
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}
	log.Printf("gRPC server listening on %s", port)
	go func() {
		if err := s.server.Serve(lis); err != nil {
			log.Printf("gRPC serve error: %v", err)
		}
	}()
	<-ctx.Done()
	stopped := make(chan struct{})
	go func() {
		st, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.server.GracefulStop()
		close(stopped)
		_ = st
	}()
	select {
	case <-stopped:
	case <-time.After(6 * time.Second):
		log.Println("Force stop gRPC server")
		s.server.Stop()
	}
	return nil
}

// loggingInterceptor 简单日志
func loggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	start := time.Now()
	resp, err = handler(ctx, req)
	code := status.Code(err)
	obs.LogInfo("rpc=%s code=%s duration=%s", info.FullMethod, code.String(), time.Since(start))
	return resp, err
}

// authInterceptor 处理 JWT 鉴权（允许部分公共方法）
func authInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// 允许匿名的方法前缀
	if isPublicMethod(info.FullMethod) {
		return handler(ctx, req)
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}
	authz := ""
	if vals := md.Get("authorization"); len(vals) > 0 {
		authz = vals[0]
	}
	if authz == "" {
		return nil, status.Error(codes.Unauthenticated, "missing authorization token")
	}
	// 支持 "Bearer xxx"
	var token string
	if len(authz) > 7 && (authz[:7] == "Bearer " || authz[:7] == "bearer ") {
		token = authz[7:]
	} else {
		token = authz
	}
	claims, err := auth.Parse(token)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}
	// 将 userId 放入 context
	ctx = context.WithValue(ctx, ctxKeyUserID{}, claims.UserID)
	return handler(ctx, req)
}

// isPublicMethod 判断是否公共方法
func isPublicMethod(fullMethod string) bool {
	// AuthService 的注册 & 登录 & 发送验证码
	return fullMethod == pb.AuthService_Register_FullMethodName ||
		fullMethod == pb.AuthService_Login_FullMethodName ||
		fullMethod == pb.AuthService_EmailCodeLogin_FullMethodName ||
		fullMethod == pb.AuthService_SendLoginEmailCode_FullMethodName ||
		fullMethod == "/grpc.health.v1.Health/Check" ||
		fullMethod == "/grpc.health.v1.Health/Watch"
}

type ctxKeyUserID struct{}

// UserIDFromContext 获取用户ID
func UserIDFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(ctxKeyUserID{})
	if v == nil {
		return "", false
	}
	id, ok := v.(string)
	return id, ok
}
