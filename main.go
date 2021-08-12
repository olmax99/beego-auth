package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"beego-auth/conf"
	"beego-auth/controllers"
	"beego-auth/models"
	_ "beego-auth/routers"

	"github.com/beego/beego/v2/adapter/orm"
	beego "github.com/beego/beego/v2/server/web"
	_ "github.com/mattn/go-sqlite3"
	"github.com/sendgrid/sendgrid-go"
)

func init() {
	beeC := conf.BeeConf("", "db::beego_db_alias")
	dbalias := beeC["beego_db_alias"]

	os.MkdirAll("./data/dev/", 0755)

	orm.RegisterDriver("sqlite3", orm.DRSqlite)
	orm.RegisterDataBase("default", "sqlite3", "./data/dev/default.db")
	if dbalias != "default" {
		orm.RegisterDataBase(dbalias, "sqlite3", "./data/dev/"+dbalias+".db")
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

	// extra configs
	beego.BConfig.WebConfig.Session.SessionOn = true

	// ------------- CHECKS---------------
	if beeC["beego_db_alias"] == "default" {
		log.Printf("WARNING [*] Ensure to update your environment variables if '%v' database is not the one you want to use.", beeC["beego_db_alias"])
	}

	if beeC["beego_vault_token"] == "empty" || beeC["beego_sg_api_key"] == "empty" {
		log.Printf("WARNING [*] Config: Vault token required.. %v", err)
		os.Exit(1)
	}
	// TODO: Move to vault package
	// Vault check
	if nc := controllers.NewCrypter(beeC); nc == nil {
		log.Fatalf("PANIC [-] VaultClient.. Please verify address, token, and key.")
	}
	// TODO: Move to sendgrid package
	// Sendgrid check
	authReq := sendgrid.GetRequest(beeC["beego_sg_api_key"], "/v3/templates", "")
	authResp, err := sendgrid.MakeRequestRetryWithContext(context.TODO(), authReq)
	if err != nil {
		log.Printf("WARNING [-] SendgridClient.. Please verify that the api key corresponds to the sg account.")
	}
	switch authResp.StatusCode {
	case 200:
	case 403:
		log.Fatalf("PANIC [-] SendgridClient.. %d, wrong api key.", authResp.StatusCode)
	default:
		log.Fatalf("PANIC [-] SendgridClient.. %d", authResp.StatusCode)
	}

	// ------------ RUN BEEGO APP---------
	// beego.Run() default run on HttpPort
	beego.Run()

}
