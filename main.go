package main

import (
	_ "beego-onc/routers"
	"database/sql"
	"fmt"
	"os"

	beego "github.com/beego/beego/v2/server/web"
	_ "github.com/mattn/go-sqlite3"
)

func init() {
	// beego.BConfig.WebConfig.Session.SessionOn = true
	// beego.BConfig.WebConfig.Session.SessionProvider = "postgresql"
	// beego.BConfig.WebConfig.Session.SessionProviderConfig = "postgres://pqgotest:password@localhost/pqgotest?sslmode=verify-full"

	// TODO overwrite BeegoConfig
	// beego.LoadAppConfig("yaml", "conf/app.conf")

	// Create DB
	os.MkdirAll("./data/dev/", 0755)
	os.Create("./data/dev/onc.db")

	db, err := sql.Open("sqlite3", "./data/dev/onc.db")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	defer db.Close()

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS `customers` (`c_id` INTEGER PRIMARY KEY AUTOINCREMENT, `client_id` VARCHAR(64) NULL, `first_name` VARCHAR(255) NOT NULL, `last_name` VARCHAR(255) NOT NULL, `guid` VARCHAR(255) NULL, `created` DATETIME NULL, `type` VARCHAR(1))")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {

	beego.Run()

}
