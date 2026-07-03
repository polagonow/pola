package actions

import (
	"context"
	"errors"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/flash"
	"github.com/polagonow/pola/i18n"
	"github.com/polagonow/pola/middleware/session"
	"beego-features-demo/services"
)

type Auth struct {
	svc *services.UserService
}

func NewAuth(r *core.Registry) *Auth {
	return &Auth{svc: core.MustInvoke[*services.UserService](r)}
}

func (a *Auth) Login(ctx context.Context, username, password string) (map[string]any, error) {
	user, err := a.svc.Authenticate(ctx, username, password)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}
	session.Set(ctx, "user_id", user.ID)
	session.Set(ctx, "username", user.Username)
	flash.Set(ctx, "success", i18n.T(ctx, "login_success"))
	return map[string]any{
		"id":       user.ID,
		"username": user.Username,
	}, nil
}

func (a *Auth) Register(ctx context.Context, username, password, displayName string) (map[string]any, error) {
	user, err := a.svc.Register(ctx, username, password, displayName)
	if err != nil {
		return nil, errors.New("registration failed: " + err.Error())
	}
	session.Set(ctx, "user_id", user.ID)
	session.Set(ctx, "username", user.Username)
	flash.Set(ctx, "success", i18n.T(ctx, "register_success"))
	return map[string]any{
		"id":       user.ID,
		"username": user.Username,
	}, nil
}

func (a *Auth) Logout(ctx context.Context) error {
	session.Clear(ctx)
	flash.Set(ctx, "success", i18n.T(ctx, "logout_success"))
	return nil
}

func (a *Auth) GetProfile(ctx context.Context) (map[string]any, error) {
	data := session.Get(ctx)
	uid, ok := data["user_id"]
	if !ok {
		return nil, errors.New("not authenticated")
	}
	id, ok := uid.(uint)
	if !ok {
		if f, ok := uid.(float64); ok {
			id = uint(f)
		} else {
			return nil, errors.New("invalid session")
		}
	}
	user, err := a.svc.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"id":           user.ID,
		"username":     user.Username,
		"display_name": user.DisplayName,
	}, nil
}

func (a *Auth) GetFlash(ctx context.Context) (map[string]string, error) {
	return flash.Get(ctx), nil
}

func (a *Auth) IsAuthenticated(ctx context.Context) (bool, error) {
	data := session.Get(ctx)
	_, ok := data["user_id"]
	return ok, nil
}

func (a *Auth) Greet(ctx context.Context, name string) (string, error) {
	return i18n.T(ctx, "greeting", name), nil
}

