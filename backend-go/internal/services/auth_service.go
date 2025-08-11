package services

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/axfinn/todoIngPlus/backend-go/internal/auth"
	"github.com/axfinn/todoIngPlus/backend-go/internal/convert"
	"github.com/axfinn/todoIngPlus/backend-go/internal/email"
	"github.com/axfinn/todoIngPlus/backend-go/internal/models"
	pb "github.com/axfinn/todoIngPlus/backend-go/pkg/api/v1"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// 默认验证码长度
const defaultLoginCodeLen = 6

// AuthService gRPC 认证服务实现
type AuthService struct {
	pb.UnimplementedAuthServiceServer
	db         *mongo.Database
	emailCodes *email.Store
}

// NewAuthService 创建新的认证服务
func NewAuthService(db *mongo.Database, emailCodes *email.Store) *AuthService {
	return &AuthService{db: db, emailCodes: emailCodes}
}

// internal mongo user doc
type mongoUserDoc struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Username  string             `bson:"username"`
	Email     string             `bson:"email"`
	Password  string             `bson:"password"`
	CreatedAt time.Time          `bson:"createdAt"`
}

// Register 用户注册（支持邮箱验证码验证）
func (s *AuthService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	if os.Getenv("DISABLE_REGISTRATION") == "true" {
		return nil, status.Error(codes.PermissionDenied, "registration disabled")
	}
	if req.Username == "" || req.Email == "" || len(req.Password) < 6 {
		return nil, status.Error(codes.InvalidArgument, "invalid fields")
	}
	emailNorm := strings.ToLower(req.Email)
	users := s.db.Collection("users")
	// 邮箱验证码验证（可选）
	if os.Getenv("ENABLE_EMAIL_VERIFICATION") == "true" {
		if s.emailCodes == nil {
			return nil, status.Error(codes.FailedPrecondition, "email code store not configured")
		}
		if err := s.emailCodes.Verify(req.EmailCodeId, emailNorm, req.EmailCode); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}
	// 查重
	if err := users.FindOne(ctx, bson.M{"$or": []bson.M{{"email": emailNorm}, {"username": req.Username}}}).Err(); err == nil {
		return nil, status.Error(codes.AlreadyExists, "user exists")
	} else if err != mongo.ErrNoDocuments {
		return nil, status.Error(codes.Internal, fmt.Sprintf("db error: %v", err))
	}
	pwHash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	doc := mongoUserDoc{Username: req.Username, Email: emailNorm, Password: string(pwHash), CreatedAt: time.Now()}
	res, err := users.InsertOne(ctx, doc)
	if err != nil {
		return nil, status.Error(codes.Internal, "insert failed")
	}
	objID, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, status.Error(codes.Internal, "invalid inserted id")
	}
	user := &models.User{ID: objID.Hex(), Username: doc.Username, Email: doc.Email, Password: "", CreatedAt: doc.CreatedAt}
	token, _ := auth.Generate(user.ID, time.Hour)
	return &pb.RegisterResponse{Response: &pb.Response{Code: 201, Message: "created"}, User: convert.UserToProto(user), Token: token}, nil
}

// Login 用户登录（密码或邮箱验证码）
func (s *AuthService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email required")
	}
	emailNorm := strings.ToLower(req.Email)
	users := s.db.Collection("users")
	// 先尝试以 ObjectID 存储的结构读取
	var record struct {
		ID        primitive.ObjectID `bson:"_id"`
		Username  string             `bson:"username"`
		Email     string             `bson:"email"`
		Password  string             `bson:"password"`
		CreatedAt time.Time          `bson:"createdAt"`
	}
	err := users.FindOne(ctx, bson.M{"email": emailNorm}).Decode(&record)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}
	// 邮箱验证码登录
	if req.EmailCodeId != "" && req.EmailCode != "" && os.Getenv("ENABLE_EMAIL_VERIFICATION") == "true" {
		if s.emailCodes == nil {
			return nil, status.Error(codes.FailedPrecondition, "email code store not configured")
		}
		if err := s.emailCodes.Verify(req.EmailCodeId, emailNorm, req.EmailCode); err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid verification code")
		}
	} else {
		// 密码登录
		if req.Password == "" {
			return nil, status.Error(codes.InvalidArgument, "password required")
		}
		if bcrypt.CompareHashAndPassword([]byte(record.Password), []byte(req.Password)) != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid credentials")
		}
	}
	token, _ := auth.Generate(record.ID.Hex(), time.Hour)
	user := &models.User{ID: record.ID.Hex(), Username: record.Username, Email: record.Email, CreatedAt: record.CreatedAt}
	return &pb.LoginResponse{Response: &pb.Response{Code: 200, Message: "ok"}, Token: token, User: convert.UserToProto(user)}, nil
}

// EmailCodeLogin Deprecated (合并到 Login)
func (s *AuthService) EmailCodeLogin(ctx context.Context, req *pb.EmailCodeLoginRequest) (*pb.LoginResponse, error) {
	return nil, status.Error(codes.Unimplemented, "use Login with email_code fields")
}

// SendLoginEmailCode 发送登录验证码
func (s *AuthService) SendLoginEmailCode(ctx context.Context, req *pb.SendLoginEmailCodeRequest) (*pb.SendLoginEmailCodeResponse, error) {
	if os.Getenv("ENABLE_EMAIL_VERIFICATION") != "true" {
		return nil, status.Error(codes.FailedPrecondition, "email verification disabled")
	}
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email required")
	}
	if s.emailCodes == nil {
		return nil, status.Error(codes.FailedPrecondition, "email code store not configured")
	}
	emailNorm := strings.ToLower(req.Email)
	users := s.db.Collection("users")
	if err := users.FindOne(ctx, bson.M{"email": emailNorm}).Err(); err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	_, _ = s.emailCodes.Generate(emailNorm, defaultLoginCodeLen)
	// 这里省略真实邮件发送（可调用 email.Send 或 SendGeneric）
	return &pb.SendLoginEmailCodeResponse{Response: &pb.Response{Code: 200, Message: "code generated"}}, nil
}

// VerifyToken 验证 JWT token
func (s *AuthService) VerifyToken(ctx context.Context, req *pb.VerifyTokenRequest) (*pb.VerifyTokenResponse, error) {
	if req.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "token required")
	}
	claims, err := auth.Parse(req.Token)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}
	users := s.db.Collection("users")
	var user models.User
	objID, _ := primitive.ObjectIDFromHex(claims.UserID)
	if err := users.FindOne(ctx, bson.M{"_id": objID}).Decode(&user); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "db error")
	}
	user.Password = ""
	return &pb.VerifyTokenResponse{Response: &pb.Response{Code: 200, Message: "valid"}, User: convert.UserToProto(&user)}, nil
}
