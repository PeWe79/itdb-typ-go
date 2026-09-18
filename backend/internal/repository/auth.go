package repository

import (
	"context"
	"database/sql"
)

type AuthUserRecord struct {
	ID       int64
	Username string
	Password string
	UserType int64
	Disabled int64
}

func (r *DomainRepository) FindAuthUser(ctx context.Context, username string) (AuthUserRecord, error) {
	var u AuthUserRecord
	e := r.QueryRowContext(ctx, `SELECT u.id,u.username,u.pass,u.usertype,COALESCE(p.disabled,0) FROM users u LEFT JOIN settings_user_profiles p ON p.user_id=u.id WHERE u.username=? LIMIT 1`, username).Scan(&u.ID, &u.Username, &u.Password, &u.UserType, &u.Disabled)
	return u, e
}
func (r *DomainRepository) FindAuthUserByWecom(ctx context.Context, wecomUserid string) (AuthUserRecord, error) {
	var u AuthUserRecord
	e := r.QueryRowContext(ctx, `SELECT u.id,u.username,u.pass,u.usertype,COALESCE(p.disabled,0) FROM settings_user_wecom w JOIN users u ON u.id=w.user_id LEFT JOIN settings_user_profiles p ON p.user_id=u.id WHERE w.wecom_userid=? LIMIT 1`, wecomUserid).Scan(&u.ID, &u.Username, &u.Password, &u.UserType, &u.Disabled)
	return u, e
}
func (r *DomainRepository) UserDescription(ctx context.Context, id int64) (string, error) {
	var value string
	e := r.QueryRowContext(ctx, `SELECT userdesc FROM users WHERE id=?`, id).Scan(&value)
	return value, e
}
func (r *DomainRepository) UpdateAuthPassword(ctx context.Context, id int64, password string) error {
	_, e := r.ExecContext(ctx, `UPDATE users SET pass=? WHERE id=?`, password, id)
	return e
}

var _ sql.Result
