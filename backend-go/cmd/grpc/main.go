package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"

	"github.com/axfinn/todoIngPlus/backend-go/internal/email"
	grpcserver "github.com/axfinn/todoIngPlus/backend-go/internal/grpc"
	obs "github.com/axfinn/todoIngPlus/backend-go/internal/observability"
	pb "github.com/axfinn/todoIngPlus/backend-go/pkg/api/v1"
	"github.com/joho/godotenv"
)

// @title TodoIng gRPC API
// @version 1.0
// @description 这是 TodoIng 项目的 gRPC API 服务
// @host localhost:9001
// @BasePath /

func main() {
	_ = godotenv.Load()
	obs.InitLogger()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		log.Fatal("MONGO_URI not set")
	}

	mdbCtx, mdbCancel := context.WithTimeout(ctx, 10*time.Second)
	defer mdbCancel()
	client, err := mongo.Connect(mdbCtx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	}
	if err = client.Ping(mdbCtx, nil); err != nil {
		log.Fatal(err)
	}
	log.Println("MongoDB connected (gRPC)")
	db := client.Database("todoing")

	// 初始化邮件验证码存储（10 分钟有效，最大 5 次尝试）
	emailStore := email.NewStore(10*time.Minute, 5)

	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "9001"
	}

	server := grpcserver.New(grpcserver.ServerConfig{Port: port}, func(s *grpc.Server) {
		pb.RegisterAuthServiceServer(s, grpcserver.NewAuthServiceServer(db, emailStore))
		pb.RegisterTaskServiceServer(s, grpcserver.NewTaskServiceServer(db))
		pb.RegisterEventServiceServer(s, grpcserver.NewEventServiceServer(db))
		pb.RegisterReminderServiceServer(s, grpcserver.NewReminderServiceServer(db))
		pb.RegisterNotificationServiceServer(s, grpcserver.NewNotificationServiceServer(db))
		pb.RegisterUnifiedServiceServer(s, grpcserver.NewUnifiedServiceServer(db))
		pb.RegisterDashboardServiceServer(s, grpcserver.NewDashboardServiceServer(db))
		pb.RegisterReportServiceServer(s, grpcserver.NewReportServiceServer(db))
		pb.RegisterCaptchaServiceServer(s, grpcserver.NewCaptchaServiceServer())
	})

	// 监听退出信号
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		cancel()
	}()

	if err := server.Start(ctx); err != nil {
		log.Fatalf("gRPC server error: %v", err)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = client.Disconnect(shutdownCtx)
	log.Println("gRPC server exited")
}
