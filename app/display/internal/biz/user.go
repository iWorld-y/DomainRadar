package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/crypto/bcrypt"
)

// User 用户实体
type User struct {
	ID           int
	Username     string
	PasswordHash string
}

// UserRepo 用户仓库接口
type UserRepo interface {
	// CreateUser 创建用户
	CreateUser(ctx context.Context, u *User) error
	// GetUserByUsername 根据用户名获取用户
	GetUserByUsername(ctx context.Context, username string) (*User, error)
}

// UserUseCase 用户业务逻辑
type UserUseCase struct {
	repo UserRepo
	log  *log.Helper
}

// NewUserUseCase 创建用户业务逻辑实例
func NewUserUseCase(repo UserRepo, logger log.Logger) *UserUseCase {
	return &UserUseCase{repo: repo, log: log.NewHelper(logger)}
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
	// 生成模拟 Token (实际应用中应使用 JWT 等)
	return "mock-token-" + username, nil
}
