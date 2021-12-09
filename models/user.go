package models

import (
	"fmt"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// Reg_key: determining user verification, i.e. Reg_key = "" -> user.Verified
type AuthUser struct {
	Id       int
	First    string
	Last     string
	Email    string `orm:"unique"`
	Password string
	Reg_key  string
	Reg_date time.Time `orm:"auto_now_add;type(datetime)"`
	Active   string    `orm:"null"`
	Clc_date time.Time `orm:"null"`
	Reset    string    `orm:"null"`
	Rst_date time.Time `orm:"null"`
}

func (u *AuthUser) OneByEmail(o orm.Ormer, e string) {
	user := &AuthUser{Email: e}
	err := o.Read(&user) // DQL Interface, calls ReadWithCtx(ctx.Background(),..)
	switch err {
	case orm.ErrNoRows:
		fmt.Println("No result found.")
	case orm.ErrMissPK:
		fmt.Println("No primary key found.")
	default:
		fmt.Println(user.Id, user.First)
	}
}
