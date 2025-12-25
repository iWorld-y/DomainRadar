package biz

import (
	"context"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

type User struct {
	ID           int
	Username     string
	PasswordHash string
}

type UserRepo interface {
	CreateUser(ctx context.Context, u *User) error
	GetUserByUsername(ctx context.Context, username string) (*User, error)
}

type UserUseCase struct {
	repo UserRepo
	log  *log.Helper
}

func NewUserUseCase(repo UserRepo, logger log.Logger) *UserUseCase {
	return &UserUseCase{repo: repo, log: log.NewHelper(logger)}
}

func (uc *UserUseCase) Register(ctx context.Context, username, password string) error {
	// In a real application, use bcrypt to hash the password
	u := &User{
		Username:     username,
		PasswordHash: password, 
	}
	return uc.repo.CreateUser(ctx, u)
}

func (uc *UserUseCase) Login(ctx context.Context, username, password string) (string, error) {
	u, err := uc.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return "", err
	}
	if u.PasswordHash != password {
		return "", errors.Unauthorized("AUTH_FAILED", "invalid password")
	}
	// Generate a simple token
	return "mock-token-" + username, nil
}
