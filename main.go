package main

import (
	"fmt"
	"os"

	"beego-auth/conf"
	"beego-auth/models"
	_ "beego-auth/routers"

	"github.com/beego/beego/v2/adapter/orm"
	beego "github.com/beego/beego/v2/server/web"
	_ "github.com/mattn/go-sqlite3"
)

func init() {
	dbalias, err := beego.AppConfig.String("db::beego_db_alias")
	if err != nil {
		fmt.Println(err)
	}

	os.MkdirAll("./data/dev/", 0755)

	orm.RegisterDriver("sqlite3", orm.DRSqlite)
	orm.RegisterDataBase("default", "sqlite3", "./data/dev/default.db")
	if dbalias != "default" {
		orm.RegisterDataBase(dbalias, "sqlite3", "./data/dev/"+dbalias+".db")
	}
	orm.RegisterModel(new(models.AuthUser))
}

func main() {
	force := false
	verbose := false
	beeC := conf.BeeConf(
		"db::beego_db_alias",
		"db::beego_db_bootstrap",
		"db::beego_db_debug",
		"vault::beego_vault_token",
	)
	if beeC["beego_db_debug"] == "true" {
		orm.Debug = true
	}
	if beeC["beego_db_bootstrap"] == "true" {
		force = true
		verbose = true
	}
	name := beeC["beego_db_alias"]
	err := orm.RunSyncdb(name, force, verbose)
	if err != nil {
		fmt.Println(err)
	}
	orm.RunCommand()

	beego.BConfig.WebConfig.Session.SessionOn = true

	// hard requirement check
	if beeC["beego_vault_token"] == "empty" || beeC["beego_sg_api_key"] == "empty" {
		fmt.Printf("PANIC [*] Config: Vault token required.. %v", err)
		os.Exit(1)
	}
	beego.Run()

}
