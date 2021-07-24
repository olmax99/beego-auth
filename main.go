package main

import (
	_ "beego-auth/models"
	_ "beego-auth/routers"
	"fmt"
	"os"

	"github.com/astaxie/beego/orm"
	beego "github.com/beego/beego/v2/server/web"
	_ "github.com/mattn/go-sqlite3"

	"github.com/beego/beego/v2/core/config"
)

func init() {
	// Create DB
	os.MkdirAll("./data/dev/", 0755)
	os.Create("./data/dev/auth.db")

	orm.RegisterDriver("sqlite3", orm.DRSqlite)
	orm.RegisterDataBase("default", "sqlite3", "./data/dev/default.db")
	orm.RegisterDataBase("authdb", "sqlite3", "./data/dev/auth.db")

}

func main() {
	// TODO config + RunSyncDb to func initDb()
	// Load configs
	iniconf, err := config.NewConfig("ini", "conf/app.conf")
	if err != nil {
		fmt.Println(err)
	}

	db_boot, err := iniconf.String("db::beego_db_bootstrap")
	if err != nil {
		fmt.Println(err)
	}
	db_debug, err := iniconf.String("db::beego_db_debug")
	if err != nil {
		fmt.Println(err)
	}

	if db_debug == "true" {
		orm.Debug = true
	}

	// orm.RunSyncdb needs to run at every startup
	if db_boot == "true" {
		name := "authdb"
		force := true
		verbose := true
		err := orm.RunSyncdb(name, force, verbose)
		if err != nil {
			fmt.Println(err)
		}
	} else {
		name := "authdb"
		force := false
		verbose := false
		err := orm.RunSyncdb(name, force, verbose)
		if err != nil {
			fmt.Println(err)
		}
	}

	// Enable cmd: orm syncdb
	orm.RunCommand()

	beego.BConfig.WebConfig.Session.SessionOn = true
	beego.Run()

}
