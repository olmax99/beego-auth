package routers

import (
	"beego-onc/controllers"

	beego "github.com/beego/beego/v2/server/web"
)

func init() {
	// GET /v1/user/create"
	ns := beego.NewNamespace("/v1",
		beego.NSNamespace("/user",
			beego.NSInclude(
				&controllers.UserController{},
			),
		),
	)
	beego.AddNamespace(ns)

	beego.Router("/", &controllers.MainController{})

}
