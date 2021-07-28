package models

import (
	"time"
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
}
