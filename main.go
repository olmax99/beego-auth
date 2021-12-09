package main

import (
	"fmt"
	"log"
	"os"

	"beego-auth/conf"
	"beego-auth/models"
	_ "beego-auth/routers"

	mail "beego-auth/pkg/sendgridv1"
	crypt "beego-auth/pkg/vaultv1"

	"github.com/beego/beego/v2/adapter/orm"
	beego "github.com/beego/beego/v2/server/web"
	_ "github.com/mattn/go-sqlite3"
)

func init() {
	beeC := conf.BeeConf("", "db::beego_db_alias")
	dbalias := beeC["beego_db_alias"]

	// ------------ SQLite--------------
	os.MkdirAll("./data/dev/", 0755)
	if err := orm.RegisterDriver("sqlite3", orm.DRSqlite); err != nil {
		log.Fatalf("PANIC [-] orm.RegisterDriver.. %s", err)
	}
	if err := orm.RegisterDataBase("default", "sqlite3", "./data/dev/default.db"); err != nil {
		log.Fatalf("PANIC [-] orm.RegisterDatabase.. %s", err)
	}
	if err := orm.RegisterDataBase(dbalias, "sqlite3", "./data/dev/"+dbalias+".db"); err != nil {
		log.Fatalf("PANIC [-] orm.RegisterDatabase.. %s", err)
	}
	orm.RegisterModel(new(models.AuthUser))
}

func main() {
	beeC := conf.BeeConf("",
		"db::beego_db_alias",
		"db::beego_db_bootstrap",
		"db::beego_db_debug",
		"vault::beego_vault_token",
		"vault::beego_vault_transit_key",
		"vault::beego_vault_address",
		"sendgrid::beego_sg_own_support",
		"sendgrid::beego_sg_api_key",
	)
	conf.PrettyConf(beeC)

	// ------------- DATABASE-------------
	force := false
	verbose := false
	if beeC["beego_db_debug"] == "true" {
		orm.Debug = true
		// w: use custom io.Writer, os.Stderr by default
		// orm.DebugLog = orm.NewLog(w)
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

	// ------------- CHECKS---------------
	if beeC["beego_db_alias"] == "default" {
		log.Printf("WARNING [*] Ensure to update your environment variables if '%s' database is not the one you want to use.", beeC["beego_db_alias"])
	}
	crypt.Confirm(beeC) // Vault check
	mail.Confirm(beeC)  // Sendgrid check

	// ------------ RUN BEEGO APP---------
	beego.BConfig.WebConfig.Session.SessionOn = true // extra configs

	beego.Run() // beego.Run() default run on HttpPort

}
