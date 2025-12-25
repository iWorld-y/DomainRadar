package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/golang-jwt/jwt/v5"
	"github.com/iWorld-y/domain_radar/app/display/internal/conf"
	"golang.org/x/crypto/bcrypt"
)

// User 用户实体
type User struct {
	ID           int
	Username     string
	PasswordHash string
	Persona      string
}

// UserRepo 用户仓库接口
type UserRepo interface {
	// CreateUser 创建用户
	CreateUser(ctx context.Context, u *User) error
	// GetUserByUsername 根据用户名获取用户
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	// UpdateUserPersona 更新用户画像
	UpdateUserPersona(ctx context.Context, id int, persona string) error
}

// UserUseCase 用户业务逻辑
type UserUseCase struct {
	repo   UserRepo
	log    *log.Helper
	jwtKey string
}

// NewUserUseCase 创建用户业务逻辑实例
func NewUserUseCase(repo UserRepo, auth *conf.Auth, logger log.Logger) *UserUseCase {
	jwtKey := "default-secret"
	if auth != nil && auth.JwtKey != "" {
		jwtKey = auth.JwtKey
	}
	return &UserUseCase{
		repo:   repo,
		log:    log.NewHelper(logger),
		jwtKey: jwtKey,
	}
}

// Register 用户注册
func (uc *UserUseCase) Register(ctx context.Context, username, password string) error {
	// 使用 bcrypt 对密码进行哈希处理
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u := &User{
		Username:     username,
		PasswordHash: string(hashedPassword),
	}
	return uc.repo.CreateUser(ctx, u)
}

// Login 用户登录
func (uc *UserUseCase) Login(ctx context.Context, username, password string) (string, error) {
	u, err := uc.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return "", err
	}
	// 验证密码哈希
	err = bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	if err != nil {
		return "", errors.Unauthorized("AUTH_FAILED", "invalid password")
	}

	// 生成真实 JWT Token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": u.Username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})

	return token.SignedString([]byte(uc.jwtKey))
}

// GetProfile 获取用户信息
func (uc *UserUseCase) GetProfile(ctx context.Context, username string) (*User, error) {
	return uc.repo.GetUserByUsername(ctx, username)
}

// UpdateProfile 更新用户画像
func (uc *UserUseCase) UpdateProfile(ctx context.Context, username, persona string) error {
	u, err := uc.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return err
	}
	return uc.repo.UpdateUserPersona(ctx, u.ID, persona)
}
