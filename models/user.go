package models

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

type authUser struct {
	Id       int
	First    string
	Last     string
	Email    string `orm:"unique"`
	Password string
	Reg_key  string    // determining user verification, i.e. Reg_key = "" -> user.Verified
	Reg_date time.Time `orm:"auto_now_add;type(datetime)"`
	Active   string    `orm:"null"`
	Clc_date time.Time `orm:"null"`
	Reset    string    `orm:"null"`
	Rst_date time.Time `orm:"null"`
}

type user struct {
	ormer orm.Ormer
	user  *authUser
	lock  sync.RWMutex
}

// GetModel register model
func NewAuthUserModel() *authUser {
	return &authUser{} // equivalent new(authUser)
}

// GetFirst return field value 'First'
func (u *user) GetFirst() string {
	return u.user.First
}

func (u *user) GetLast() string {
	return u.user.Last
}

func (u *user) GetEmail() string {
	return u.user.Email
}

func (u *user) GetActive() string {
	return u.user.Active
}

func (u *user) GetReset() string {
	return u.user.Reset
}

func (u *user) GetResetDte() time.Time {
	return u.user.Rst_date
}

func (u *user) GetCancelDte() time.Time {
	return u.user.Clc_date
}

// GetPasswd return field value 'Password' (encrypted)
func (u *user) GetPasswd() string {
	return u.user.Password
}

// GetRegkey return field value 'Reg_key'
func (u *user) GetRegkey() string {
	return u.user.Reg_key
}

// UserActive check if either Reg_key or Active token is set
func (u *user) UserActive() bool {
	return u.user.Reg_key != "" || u.user.Active != ""
}

func (u *user) UserConfirm(field string) error {
	err := u.ormer.Read(u.user, field) // DQL Interface, calls ReadWithCtx(ctx.Background(),..)
	switch err {
	case nil:
		return nil
	case orm.ErrNoRows:
		fmt.Printf("DEBUG [-] UserConfirm with field: '%s'.. No result found.", field)
		return err
	case orm.ErrMissPK:
		fmt.Printf("DEBUG [-] UserConfirm with field: '%s'.. No primary key found.", field)
		return err
	default:
		fmt.Printf("DEBUG [-] UserConfirm with field: '%s'.. unhandled exception.", field)
		return orm.ErrNotImplement
	}
}

// UserInsert inserts single user (Tx for single task not needed)
func (u *user) UserInsertTx() error {
	err := u.ormer.DoTx(
		func(ctx context.Context, txOrm orm.TxOrmer) error {
			_, e := txOrm.Insert(u.user)
			return e
		})
	if err != nil {
		return err
	}
	return nil
}

// UserRemoveRegKey
func (u *user) UserRemoveRegKey() error {
	u.user.Reg_key = ""
	_, err := u.ormer.Update(u.user)
	if err != nil {
		return err
	}
	return nil
}

// UserUpdateClc updates cancel time token
func (u *user) UserUpdateClcDte(clc time.Time) error {
	u.user.Active = ""
	u.user.Clc_date = clc
	_, err := u.ormer.Update(u.user)
	if err != nil {
		return err
	}
	return nil
}

// UserUpdateRst updates reset time token
func (u *user) UserUpdateRstDte(rst time.Time) error {
	u.user.Rst_date = rst
	_, err := u.ormer.Update(u.user)
	if err != nil {
		return err
	}
	return nil
}

func (u *user) UserUpdatePasswd(passwd, rst string) error {
	u.user.Password = passwd
	if rst != "" {
		u.user.Reset = rst
	}
	_, err := u.ormer.Update(u.user)
	if err != nil {
		return err
	}
	return nil
}

func (u *user) UserUpdateInfo(first, last string) error {
	u.user.First = first
	u.user.Last = last
	_, err := u.ormer.Update(u.user)
	if err != nil {
		return err
	}
	return nil
}

func (u *user) UserUpdateActive(act string) error {
	u.user.Active = act
	_, err := u.ormer.Update(u.user)
	if err != nil {
		return err
	}
	return nil
}

// authUserConf config closure for authUser
type authUserConf func(user *user)

// NewauthUser initializes a new authUser
func NewUser(dbalias string, conf ...authUserConf) *user {
	u := &user{}
	u.lock.Lock()
	defer u.lock.Unlock()
	u.ormer = orm.NewOrmUsingDB(dbalias)
	u.user = &authUser{}
	for _, c := range conf {
		c(u)
	}
	return u
}

// PrepareWrite User config option generally applicable for initializing writes
func PrepareWrite(first, last, email, passwd, regkey string) authUserConf {
	return func(u *user) {
		u.user.First = first
		u.user.Last = last
		u.user.Email = email
		u.user.Password = passwd
		u.user.Reg_key = regkey
	}
}

// WithEmail User config option use when reading user by email
func WithEmail(email string) authUserConf {
	return func(u *user) {
		u.user.Email = email
	}
}

// WithEmail User config option use when reading user by Reg_Key
func WithRegKey(regkey string) authUserConf {
	return func(u *user) {
		u.user.Reg_key = regkey
	}
}

// WithEmail User config option use when reading user by Active
func WithActive(uid string) authUserConf {
	return func(u *user) {
		u.user.Active = uid
	}
}

// WithEmail User config option use when reading user by Active
func WithReset(rst string) authUserConf {
	return func(u *user) {
		u.user.Reset = rst
	}
}
