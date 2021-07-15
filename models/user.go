package models

import (
	"database/sql"
	"math/rand"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func init() {
	// Initialized before main.init() <- Db create and table initialize in main!!
}

type oncUser struct {
	clientId  int64
	firstName string
	lastName  string
	guid      int64
	created   time.Time
	oncType   string
}

func (o *oncUser) randint() int {
	rand.Seed(time.Now().UnixNano())
	min := 100
	max := 999
	n := rand.Intn(max-min+1) + min
	return n
}

func (o *oncUser) NewOncUser(f, l string, g int64) *oncUser {
	oncid := o.randint()
	p := &oncUser{
		clientId: int64(oncid),
		created:  time.Now(),
		oncType:  "o",
	}
	p.firstName = f
	p.lastName = l
	p.guid = g
	return p
}

func AddUser() int64 {
	db, err := sql.Open("sqlite3", "./data/dev/onc.db")
	if err != nil {
		panic(err)
	}

	defer db.Close()

	stmt, err := db.Prepare("INSERT INTO customers(client_id, first_name, last_name, guid, created, type) values(?,?,?,?,?,?)")
	if err != nil {
		panic(err)
	}
	o := &oncUser{}
	ou := o.NewOncUser("tina", "turner", int64(001))
	_, err = stmt.Exec(ou.clientId, ou.firstName, ou.lastName, ou.guid, ou.created, ou.oncType)
	if err != nil {
		panic(err)
	}
	return ou.clientId

}
