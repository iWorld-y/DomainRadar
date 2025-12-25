package data

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/iWorld-y/domain_radar/app/display/internal/biz"
	"github.com/iWorld-y/domain_radar/app/common/ent"
	"github.com/iWorld-y/domain_radar/app/common/ent/user"
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
	_, err := r.data.db.User.Create().
		SetUsername(u.Username).
		SetPasswordHash(u.PasswordHash).
		Save(ctx)
	return err
}

func (r *userRepo) GetUserByUsername(ctx context.Context, username string) (*biz.User, error) {
	u, err := r.data.db.User.Query().
		Where(user.Username(username)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.NotFound("USER_NOT_FOUND", "user not found")
		}
		return nil, err
	}
	return &biz.User{
		ID:           int(u.ID),
		Username:     u.Username,
		PasswordHash: u.PasswordHash,
	}, nil
}
