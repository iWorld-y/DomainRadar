package data

import (
	"context"
	"database/sql"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/iWorld-y/domain_radar/app/display/internal/biz"
)

type userRepo struct {
	data *Data
	log  *log.Helper
}

func NewUserRepo(data *Data, logger log.Logger) biz.UserRepo {
	return &userRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *userRepo) CreateUser(ctx context.Context, u *biz.User) error {
	_, err := r.data.db.ExecContext(ctx, "INSERT INTO users (username, password_hash) VALUES ($1, $2)", u.Username, u.PasswordHash)
	return err
}

func (r *userRepo) GetUserByUsername(ctx context.Context, username string) (*biz.User, error) {
	row := r.data.db.QueryRowContext(ctx, "SELECT id, username, password_hash FROM users WHERE username = $1", username)
	var u biz.User
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash); err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NotFound("USER_NOT_FOUND", "user not found")
		}
		return nil, err
	}
	return &u, nil
}
