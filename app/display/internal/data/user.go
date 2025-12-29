package data

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/iWorld-y/domain_radar/app/common/ent"
	"github.com/iWorld-y/domain_radar/app/common/ent/user"
	"github.com/iWorld-y/domain_radar/app/display/internal/usecase"
)

type userRepo struct {
	data *Data
	log  *log.Helper
}

func NewUserRepo(data *Data, logger log.Logger) usecase.UserRepo {
	return &userRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *userRepo) CreateUser(ctx context.Context, u *usecase.User) error {
	_, err := r.data.db.User.Create().
		SetUsername(u.Username).
		SetPasswordHash(u.PasswordHash).
		Save(ctx)
	return err
}

func (r *userRepo) GetUserByUsername(ctx context.Context, username string) (*usecase.User, error) {
	u, err := r.data.db.User.Query().
		Where(user.Username(username)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.NotFound("USER_NOT_FOUND", "user not found")
		}
		return nil, err
	}
	domains := u.Domains
	if domains == nil {
		domains = []string{}
	}
	return &usecase.User{
		ID:           u.ID,
		Username:     u.Username,
		PasswordHash: u.PasswordHash,
		Persona:      u.Persona,
		Domains:      domains,
	}, nil
}

func (r *userRepo) UpdateUserProfile(ctx context.Context, id int, persona string, domains []string) error {
	if domains == nil {
		domains = []string{}
	}
	return r.data.db.User.UpdateOneID(id).
		SetPersona(persona).
		SetDomains(domains).
		Exec(ctx)
}
