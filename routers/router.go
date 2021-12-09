package routers

import (
	"beego-auth/controllers"

	beego "github.com/beego/beego/v2/server/web"
)

func init() {
	beego.Router("/user/login/:back", &controllers.DefaultController{}, "get,post:Login")
	beego.Router("/user/register", &controllers.DefaultController{}, "get,post:Register")
	beego.Router("/user/verify/:uuid([0-9A-Fa-f]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89ABab][0-9a-fA-F]{3}-[a-fA-F0-9]{12}$)", &controllers.DefaultController{}, "get:Verify")
	beego.Router("/user/cancel/:uuid([0-9A-Fa-f]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89ABab][0-9a-fA-F]{3}-[a-fA-F0-9]{12}$)", &controllers.DefaultController{}, "get:Cancel")
	beego.Router("/user/genpass1/:uuid([0-9A-Fa-f]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89ABab][0-9a-fA-F]{3}-[a-fA-F0-9]{12}$)", &controllers.DefaultController{}, "get:Genpass1")
	beego.Router("/user/reset", &controllers.DefaultController{}, "get,post:Reset")
	beego.Router("/notice", &controllers.DefaultController{}, "get:Notice")
	// ---- requires login---------
	beego.Router("/console", &controllers.DefaultController{})
	beego.Router("/user/profile", &controllers.DefaultController{}, "get,post:Profile")
	beego.Router("/user/remove", &controllers.DefaultController{}, "get,post:Remove")
	beego.Router("/user/genpass2", &controllers.DefaultController{}, "get,post:Genpass2")
	beego.Router("/user/logout", &controllers.DefaultController{}, "get:Logout")
	// find request handling and filters at https://beego.vip/docs/mvc/
	beego.InsertFilter("/console", beego.BeforeRouter, controllers.FilterUser)
	beego.InsertFilter("/user/profile", beego.BeforeRouter, controllers.FilterUser)
	beego.InsertFilter("/user/remove", beego.BeforeRouter, controllers.FilterUser)
	beego.InsertFilter("/user/genpass2", beego.BeforeRouter, controllers.FilterUser)
	beego.InsertFilter("/user/logout", beego.BeforeRouter, controllers.FilterUser)
}
