package models

import (
	"time"
)

type OncUser struct {
	Id      int
	Name    string
	Email   string      `orm:"unique"`
	Profile *OncProfile `orm:"rel(one)"` // OneToOne relation
}

type OncProfile struct {
	Id      int
	Age     int16
	Answer  string
	Created time.Time `orm:"type(datetime);precision(2)"`
	Balance float64   `orm:"digits(12);decimals(4)"`
	User    *OncUser  `orm:"reverse(one)"` // Reverse relationship (optional)
}

func init() {
	// Initialized before main.init() <- Db create and table initialize in main!!
}
