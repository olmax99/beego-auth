package models_test

import (
	"beego-auth/models"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

// NOTE: the sqldump needs to be created externally!! (use 'beego orm sqlall')
// setup Database conn
func setup(dbalias string) {
	orm.RegisterDriver("sqlite3", orm.DRSqlite)
	orm.RegisterDataBase(dbalias, "sqlite3", "./testdb.db")
	orm.RegisterModel(models.NewAuthUserModel())
	orm.Debug = true

	// create db from file
	body, err := ioutil.ReadFile("./testdb.sql")
	if err != nil {
		log.Fatalf("unable to read file: %v", err)
	}
	o := orm.NewOrmUsingDB(dbalias)
	res, err := o.Raw(string(body)).Exec()
	if err == nil {
		num, _ := res.RowsAffected()
		log.Println("mysql row affected nums: ", num)
	}

}

// shutdown ...
func shutdown() {
	if err := os.Remove("./testdb.sql"); err != nil {
		log.Fatalf("err: %v", err)
	}
	if err := os.Remove("./testdb.db"); err != nil {
		log.Fatalf("err: %v", err)
	}
}

const (
	alias = "testdb"
)

func TestMain(m *testing.M) {
	setup(alias)
	code := m.Run()
	shutdown()
	os.Exit(code)
}

// NOTE: InsertTx PASS will create test user for the other tests!! Order matters!!
func TestUserInsertTx(t *testing.T) {
	user := models.NewUser(alias, models.PrepareWrite("mi", "moo31", "mimoo31@example.net", "super_secret", "0000-0000-0000-0000"))
	if err := user.UserInsertTx(); err != nil {
		t.Fatalf("DEBUG [*] TestUserInsertTx err: %s", err)
	}
	exp := models.NewUser(alias, models.WithEmail("mimoo31@example.net"))
	if err := exp.UserConfirm("Email"); err != nil {
		log.Fatalf("not found.. %s", err)
	}
}

// TestAuthUserConfirmImplicitRead tests wheather UserConfirm implicitely 'loads'
// user struct values correctly
func TestAuthUserConfirmImplicitRead(t *testing.T) {
	u1 := models.NewUser(alias, models.WithEmail("mimoo31@example.net"))
	if first := u1.GetFirst(); first != "" {
		t.Fatalf("DEBUG [*] expected '<nil>' ..actual: '%s'", first)
	}
	// UserConfirm implicitely loads user struct values
	if err := u1.UserConfirm("Email"); err != nil {
		t.Fatalf("DEBUG [*] u.UserConfirm: %v", err)
	}
	// Check if implicit read of confirm loaded all values into initialized user struct
	if first := u1.GetFirst(); first != "mi" {
		t.Fatalf("DEBUG [*] expected 'mi' ..actual: '%s'", first)
	}
	if last := u1.GetLast(); last != "moo31" {
		t.Fatalf("DEBUG [*] expected 'moo31' ..actual: '%s'", last)
	}
}

// This shows that there is no implicit loading of struct variables with
// UserUpdate only!! UserConfirm is always required!!
func TestAuthUserImplicitUpdateWithoutConfirm(t *testing.T) {
	u2 := models.NewUser(alias, models.WithEmail("mimoo31@example.net"))
	if act := u2.GetActive(); act != "" {
		t.Fatalf("DEBUG [*] expected '<null>' ..actual: '%s'", act)
	}
	if clc := u2.GetCancelDte(); !clc.Equal(time.Time{}) {
		t.Fatalf("DEBUG [*] expected '<null>' ..actual: '%s'", clc)
	}
	tnow := time.Now().UTC()
	if err := u2.UserUpdateClcDte(tnow); err != nil {
		t.Fatalf("DEBUG [*]  u.UserUpdateClcDte %s", err)
	}
	if clc := u2.GetCancelDte(); !clc.Equal(tnow) {
		t.Fatalf("DEBUG [*] expected '%v' ..actual: '%v'", tnow, clc)
	}
	if first := u2.GetFirst(); first != "" {
		t.Fatalf("DEBUG [*] expected 'mi' ..actual: '%s'", first)
	}
}

func TestAuthUserImplicitUpdate(t *testing.T) {
	u2 := models.NewUser(alias, models.WithEmail("mimoo31@example.net"))
	if err := u2.UserConfirm("Email"); err != nil {
		t.Fatalf("DEBUG [*] u.UserConfirm: %v", err)
	}
	if act := u2.GetActive(); act != "" {
		t.Fatalf("DEBUG [*] expected '<null>' ..actual: '%s'", act)
	}
	if clc := u2.GetCancelDte(); !clc.Equal(time.Time{}) {
		t.Fatalf("DEBUG [*] expected '<null>' ..actual: '%s'", clc)
	}
	tnow := time.Now().UTC()
	if err := u2.UserUpdateClcDte(tnow); err != nil {
		t.Fatalf("DEBUG [*]  u.UserUpdateClcDte %s", err)
	}
	if clc := u2.GetCancelDte(); !clc.Equal(tnow) {
		t.Fatalf("DEBUG [*] expected '%v' ..actual: '%v'", tnow, clc)
	}
	if first := u2.GetFirst(); first != "mi" {
		t.Fatalf("DEBUG [*] expected 'mi' ..actual: '%s'", first)
	}
}
