package routers

import (
	"beego-auth/controllers"

	beego "github.com/beego/beego/v2/server/web"
)

func init() {
	// ---- requires login---------
	beego.Router("/home", &controllers.MainController{})
	beego.Router("/user/profile", &controllers.MainController{}, "get,post:Profile")
	beego.Router("/user/remove", &controllers.MainController{}, "get,post:Remove")
	beego.Router("/user/genpass2", &controllers.MainController{}, "get,post:Genpass2")
	beego.Router("/user/logout", &controllers.MainController{}, "get:Logout")
	// ----------------------------
	beego.Router("/user/login/:back", &controllers.MainController{}, "get,post:Login")
	beego.Router("/user/register", &controllers.MainController{}, "get,post:Register")
	beego.Router("/user/verify/:uuid([0-9A-Fa-f]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89ABab][0-9a-fA-F]{3}-[a-fA-F0-9]{12}$)", &controllers.MainController{}, "get:Verify")
	beego.Router("/user/cancel/:uuid([0-9A-Fa-f]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89ABab][0-9a-fA-F]{3}-[a-fA-F0-9]{12}$)", &controllers.MainController{}, "get:Cancel")
	beego.Router("/user/genpass1/:uuid([0-9A-Fa-f]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89ABab][0-9a-fA-F]{3}-[a-fA-F0-9]{12}$)", &controllers.MainController{}, "get:Genpass1")
	beego.Router("/user/reset", &controllers.MainController{}, "get,post:Reset")
	beego.Router("/notice", &controllers.MainController{}, "get:Notice")

	beego.InsertFilter("/user/logout", beego.BeforeRouter, controllers.FilterUser)
	beego.InsertFilter("/user/profile", beego.BeforeRouter, controllers.FilterUser)
	beego.InsertFilter("/user/genpass2", beego.BeforeRouter, controllers.FilterUser)
	beego.InsertFilter("/user/remove", beego.BeforeRouter, controllers.FilterUser)
	beego.InsertFilter("/home", beego.BeforeRouter, controllers.FilterUser)
}
